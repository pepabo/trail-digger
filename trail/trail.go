package trail

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
	"github.com/zhangyunhao116/skipmap"
	"github.com/zhangyunhao116/wyhash"
	"golang.org/x/sync/errgroup"
)

const datePathFormat = "2006/01/02"

type LogData struct {
	Records []*Record `json:"Records"`
}

type Record struct {
	EventVersion string `json:"eventVersion"`
	UserIdentity struct {
		Type        string `json:"type"`
		InvokedBy   string `json:"invokedBy"`
		PrincipalID string `json:"principalId"`
		Arn         string `json:"arn"`
		AccountID   string `json:"accountId"`
		AccessKeyID string `json:"accessKeyId"`
		UserName    string `json:"userName"`
	} `json:"userIdentity"`
	EventTime         time.Time `json:"eventTime"`
	EventSource       string    `json:"eventSource"`
	EventName         string    `json:"eventName"`
	AwsRegion         string    `json:"awsRegion"`
	SourceIPAddress   string    `json:"sourceIPAddress"`
	UserAgent         string    `json:"userAgent"`
	RequestParameters struct {
		BucketName               string `json:"bucketName"`
		Host                     string `json:"Host"`
		XAmzACL                  string `json:"x-amz-acl"`
		XAmzServerSideEncryption string `json:"x-amz-server-side-encryption"`
		Key                      string `json:"key"`
	} `json:"requestParameters,omitempty"`
	ResponseElements struct {
		XAmzServerSideEncryption string `json:"x-amz-server-side-encryption"`
		XAmzVersionID            string `json:"x-amz-version-id"`
	} `json:"responseElements"`
	AdditionalEventData struct {
		SignatureVersion     string  `json:"SignatureVersion"`
		CipherSuite          string  `json:"CipherSuite"`
		BytesTransferredIn   float64 `json:"bytesTransferredIn"`
		SSEApplied           string  `json:"SSEApplied"`
		AuthenticationMethod string  `json:"AuthenticationMethod"`
		XAmzID2              string  `json:"x-amz-id-2"`
		BytesTransferredOut  float64 `json:"bytesTransferredOut"`
	} `json:"additionalEventData,omitempty"`
	RequestID string `json:"requestID"`
	EventID   string `json:"eventID"`
	ReadOnly  bool   `json:"readOnly"`
	Resources []struct {
		Type      string `json:"type"`
		Arn       string `json:"ARN"`
		AccountID string `json:"accountId,omitempty"`
	} `json:"resources"`
	EventType          string `json:"eventType"`
	ManagementEvent    bool   `json:"managementEvent"`
	RecipientAccountID string `json:"recipientAccountId"`
	SharedEventID      string `json:"sharedEventID"`
	EventCategory      string `json:"eventCategory"`
}

type Option struct {
	DatePath      string
	StartDatePath string
	EndDatePath   string
	Accounts      []string
	Regions       []string
	AllAccounts   bool
	AllRegions    bool
}

type WalkEventsFunc func(r *Record) error

func WalkEvents(sess *session.Session, dsn string, opt Option, fn WalkEventsFunc) error {
	bucket, prefixes, err := generatePrefixes(sess, dsn, opt, true)
	if err != nil {
		return err
	}

	s3c := s3.New(sess)
	var t *string
	em := map[string]*skipmap.Float64Map{}

	days, err := datePaths(opt, true)
	if err != nil {
		return err
	}
	st, err := time.Parse("2006/01/02", days[0])
	if err != nil {
		return err
	}
	et, err := time.Parse("2006/01/02", days[len(days)-1])
	if err != nil {
		return err
	}
	stn := st.UnixNano()
	etn := et.UnixNano()

	for _, pd := range prefixes {
		day := pd.day.Format(datePathFormat)
		em[day] = skipmap.NewFloat64()
		eg := errgroup.Group{}
		for _, prefix := range pd.prefixes {
			log.Info().Str("prefix", prefix).Msg("Digging trail logs")
			func(bucket, prefix string) {
				eg.Go(func() error {
					for {
						o, err := s3c.ListObjectsV2(&s3.ListObjectsV2Input{
							Bucket:            aws.String(bucket),
							Prefix:            aws.String(prefix),
							ContinuationToken: t,
						})
						if err != nil {
							return err
						}
						for _, c := range o.Contents {
							obj, err := s3c.GetObject(&s3.GetObjectInput{
								Bucket: aws.String(bucket),
								Key:    c.Key,
							})
							if err != nil {
								return err
							}
							buf := new(bytes.Buffer)
							if _, err := io.Copy(buf, obj.Body); err != nil {
								_ = obj.Body.Close()
								return err
							}
							if err := obj.Body.Close(); err != nil {
								return err
							}
							td := LogData{}
							if err := json.Unmarshal(buf.Bytes(), &td); err != nil {
								return err
							}
							for _, r := range td.Records {
								tf := r.EventTime.Format(datePathFormat)
								tn := r.EventTime.UnixNano()
								if tn < stn || etn < tn {
									continue
								}
								k, err := strconv.ParseFloat(fmt.Sprintf("%d.%d", r.EventTime.Unix(), wyhash.Sum64String(r.EventID)), 64)
								if err != nil {
									return err
								}
								em[tf].Store(k, r)
							}
						}
						if o.NextContinuationToken == nil {
							break
						}
						t = o.NextContinuationToken
					}
					return nil
				})
			}(bucket, prefix)
		}
		if err := eg.Wait(); err != nil {
			return err
		}
		ptd := pd.day.AddDate(0, 0, -1).Format(datePathFormat)
		if prev, ok := em[ptd]; ok {
			var err error
			prev.Range(func(k float64, v interface{}) bool {
				r := v.(*Record)
				err = fn(r)
				if err != nil {
					log.Debug().Err(err)
					return false
				}
				return true
			})
			if err != nil {
				return err
			}
			em[ptd] = nil
		}
	}
	ld := prefixes[len(prefixes)-1].day.Format(datePathFormat)
	em[ld].Range(func(k float64, v interface{}) bool {
		r := v.(*Record)
		err = fn(r)
		if err != nil {
			log.Debug().Err(err)
			return false
		}
		return true
	})

	return nil
}

type Prefixes []*PrefixesGroupPerDay

type PrefixesGroupPerDay struct {
	day      time.Time
	prefixes []string
}

// generatePrefixes generate prefix per day order day
func generatePrefixes(sess *session.Session, dsn string, opt Option, after1Day bool) (string, Prefixes, error) {
	if !strings.HasPrefix(dsn, "s3://") {
		return "", nil, fmt.Errorf("invalid s3 bucket url: %s", dsn)
	}
	splitted := strings.SplitN(strings.TrimPrefix(dsn, "s3://"), "/", 2)
	if len(splitted) == 0 || splitted[0] == "" {
		return "", nil, fmt.Errorf("invalid s3 bucket url: %s", dsn)
	}
	days, err := datePaths(opt, after1Day)
	if err != nil {
		return "", nil, err
	}
	bucket := splitted[0]
	prefix := "AWSLogs"
	if len(splitted) > 1 && splitted[1] != "" {
		prefix = splitted[1]
	}
	s3c := s3.New(sess)

	accounts := []string{}
	switch {
	case opt.AllAccounts:
		o, err := s3c.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:    aws.String(bucket),
			Prefix:    aws.String(fmt.Sprintf("%s/", prefix)),
			Delimiter: aws.String("/"),
		})
		if err != nil {
			return "", nil, err
		}
		for _, p := range o.CommonPrefixes {
			accounts = append(accounts, strings.Trim(strings.Replace(*p.Prefix, prefix, "", -1), "/"))
		}
	case len(opt.Accounts) > 0:
		accounts = opt.Accounts
	default:
		stsc := sts.New(sess)
		i, err := stsc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		if err != nil {
			return "", nil, err
		}
		accountId := *i.Account
		accounts = []string{accountId}
	}

	roots := []string{}
	for _, a := range accounts {
		regions := []string{}
		switch {
		case opt.AllRegions:
			prefix := fmt.Sprintf("%s/", path.Join(prefix, a, "CloudTrail"))
			o, err := s3c.ListObjectsV2(&s3.ListObjectsV2Input{
				Bucket:    aws.String(bucket),
				Prefix:    aws.String(prefix),
				Delimiter: aws.String("/"),
			})
			if err != nil {
				return "", nil, err
			}
			for _, p := range o.CommonPrefixes {
				regions = append(regions, strings.Trim(strings.Replace(*p.Prefix, prefix, "", -1), "/"))
			}
		case len(regions) > 0:
			regions = opt.Regions
		default:
			region := *sess.Config.Region
			regions = []string{region}
		}
		for _, r := range regions {
			roots = append(roots, path.Join(prefix, a, "CloudTrail", r))
		}
	}
	prefixes := Prefixes{}
	for _, d := range days {
		dt, err := time.Parse(datePathFormat, d)
		if err != nil {
			return "", nil, err
		}
		pd := &PrefixesGroupPerDay{
			day:      dt,
			prefixes: []string{},
		}
		for _, r := range roots {
			pd.prefixes = append(pd.prefixes, fmt.Sprintf("%s/", path.Join(r, d)))
		}
		prefixes = append(prefixes, pd)
	}
	return bucket, prefixes, nil
}

func datePaths(opt Option, after1Day bool) ([]string, error) {
	paths := []string{}
	if opt.StartDatePath != "" && opt.EndDatePath != "" {
		st, err := time.Parse("2006/01/02", opt.StartDatePath)
		if err != nil {
			return []string{}, fmt.Errorf("invalid start date format: %s", opt.StartDatePath)
		}
		et, err := time.Parse("2006/01/02", opt.EndDatePath)
		if err != nil {
			return []string{}, fmt.Errorf("invalid end date format: %s", opt.EndDatePath)
		}
		dt := et.Sub(st)
		dd := int(dt.Hours()) / 24
		if after1Day {
			dd += 1
		}
		for d := 0; d <= dd; d++ {
			t := st.AddDate(0, 0, d)
			paths = append(paths, fmt.Sprintf("%04d/%02d/%02d", t.Year(), t.Month(), t.Day()))
		}
		return paths, nil
	}

	splitted := strings.Split(opt.DatePath, "/")
	if len(splitted) > 3 || splitted[0] == "" {
		return []string{}, fmt.Errorf("invalid date format: %s", opt.DatePath)
	}
	year, err := strconv.Atoi(splitted[0])
	if err != nil {
		return []string{}, fmt.Errorf("invalid date format: %s", opt.DatePath)
	}
	months := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	if len(splitted) >= 2 {
		month, err := strconv.Atoi(splitted[1])
		if err != nil {
			return []string{}, fmt.Errorf("invalid date format: %s", opt.DatePath)
		}
		months = []int{month}
	}
	var a1d time.Time
	if len(splitted) == 3 {
		// 2006/01/02
		day, err := strconv.Atoi(splitted[2])
		if err != nil {
			return []string{}, fmt.Errorf("invalid date format: %s", opt.DatePath)
		}
		paths = append(paths, fmt.Sprintf("%04d/%02d/%02d", year, months[0], day))
		a1d = time.Date(year, time.Month(months[0]), day, 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
	} else {
		// 2006 or 2006/01
		for _, month := range months {
			l := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1).Day()
			for day := 1; day <= l; day++ {
				paths = append(paths, fmt.Sprintf("%04d/%02d/%02d", year, month, day))
			}
		}
		lastMonth := months[len(months)-1]
		a1d = time.Date(year, time.Month(lastMonth+1), 1, 0, 0, 0, 0, time.UTC)
	}
	if after1Day {
		paths = append(paths, a1d.Format(datePathFormat))
	}

	return paths, nil
}
