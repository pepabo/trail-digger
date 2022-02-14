package trail

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type WalkObjectsFunc func(o *s3.Object) error

func WalkObjects(sess *session.Session, dsn string, opt Option, fn WalkObjectsFunc) error {
	bucket, prefixes, err := generatePrefixes(sess, dsn, opt, false)
	if err != nil {
		return err
	}
	s3c := s3.New(sess)
	var t *string
	for _, pd := range prefixes {
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
							if err := fn(c); err != nil {
								return err
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
	}
	return nil
}
