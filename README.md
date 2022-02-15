# trail-digger

`trail-digger` is a tool for digging trail log files of AWS CloudTrail.

**NOTICE: If Amazon Athena is available, [most issues can be solved with Amazon Athena](https://docs.aws.amazon.com/athena/latest/ug/cloudtrail-logs.html)**

## Usage

### `trail-digger events`

`trail-digger events` show AWS CloudTrail events (JSONL) **in order of timeline** using trail logs.

#### Show the events of 2022/02/03 for AWS account/default region of `my-profile` in order of timeline

``` console
$ env AWS_PROFILE=my-profile trail-digger events s3://your-trail-log-bucket --date 2022/02/03
```

#### Show the events of 2022/02 for AWS account/us-west-2 of `my-profile` in order of timeline

``` console
$ env AWS_PROFILE=my-profile trail-digger events s3://your-trail-log-bucket --date 2022/02 --region us-west-2
```

#### Show the events of 2022/01 for AWS account(1234567890)/all regions in order of timeline

``` console
$ env AWS_PROFILE=my-profile trail-digger events s3://your-trail-log-bucket --date 2022/01 --account 1234567890 --all-regions 
```

#### Show the events of 2022/01/04 for all AWS accounts/all regions in order of timeline

``` console
$ env AWS_PROFILE=my-profile trail-digger events s3://your-trail-log-bucket --date 2022/01/04 --all-accounts --all-regions 
```

### `trail-digger analyze`

`trail-digger analyze` analyze AWS CloudTrail events using trail logs.

The usage is the same as `trail-digger analyze`, but it outputs the analysis results.

``` console
$ AWS_PROFILE=my-profile trail-digger analyze s3://your-trail-log-bucket
2022-02-15T07:27:48+09:00 INF Digging trail logs prefix=AWSLogs/1234567890/CloudTrail/ap-northeast-1/2022/02/15/
2022-02-15T07:28:07+09:00 INF Digging trail logs prefix=AWSLogs/1234567890/CloudTrail/ap-northeast-1/2022/02/16/

                                                             Count

  Event Type            Management Event:                 12345678
                        Data Event:                          12345

  Event Source          access-analyzer.amazonaws.com:       XXXXX
                        autoscaling.amazonaws.com:            XXXX
                        cloudformation.amazonaws.com:         XXXX
                        cloudtrail.amazonaws.com:            XXXXX
                        codepipeline.amazonaws.com:           XXXX
                        config.amazonaws.com:                  XXX
                        dax.amazonaws.com:                   XXXXX
                        dms.amazonaws.com:                     XXX
                        ec2.amazonaws.com:                      XX
                        ecr.amazonaws.com:                     XXX
                        ecs.amazonaws.com:                   XXXXX
                        elasticfilesystem.amazonaws.com:       XXX
                        elasticloadbalancing.amazonaws.com:   XXXX
                        elasticmapreduce.amazonaws.com:       XXXX
                        es.amazonaws.com:                      XXX
                        guardduty.amazonaws.com:             　XXX
                        kms.amazonaws.com:                     XXX
                        lambda.amazonaws.com:                XXXXX
                        redshift.amazonaws.com:               XXXX
                        s3.amazonaws.com:                   XXXXXX
                        sagemaker.amazonaws.com:               XXX
                        secretsmanager.amazonaws.com:         XXXX
                        ssm.amazonaws.com:                     XXX
                        sts.amazonaws.com:                    XXXX

  Region                ap-northeast-1:                   12358023

  Recipient Account ID  1234567890:                       12358023

```

### `trail-digger size`

`trail-digger size` show size of trail logs.

The usage is the same as `trail-digger analyze`, but it outputs the size of trail log S3 objects.

In addition, for `trail-digger events` and `trail-digger analyze`, the aggregation range is determined by `eventTime`, but for `trail-digger size`, the aggregation range is determined by the date path of the S3 bucket.
