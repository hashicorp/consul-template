package physical

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/armon/go-metrics"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3Backend is a physical backend that stores data
// within an S3 bucket.
type S3Backend struct {
	bucket string
	client *s3.S3
}

// newS3Backend constructs a S3 backend using a pre-existing
// bucket. Credentials can be provided to the backend, sourced
// from the environment, AWS credential files or by IAM role.
func newS3Backend(conf map[string]string) (Backend, error) {

	bucket := os.Getenv("AWS_S3_BUCKET")
	if bucket == "" {
		bucket = conf["bucket"]
		if bucket == "" {
			return nil, fmt.Errorf("'bucket' must be set")
		}
	}

	access_key, ok := conf["access_key"]
	if !ok {
		access_key = ""
	}
	secret_key, ok := conf["secret_key"]
	if !ok {
		secret_key = ""
	}
	session_token, ok := conf["session_token"]
	if !ok {
		session_token = ""
	}
	endpoint := os.Getenv("AWS_S3_ENDPOINT")
	if endpoint == "" {
		endpoint = conf["endpoint"]
	}
	region := os.Getenv("AWS_DEFAULT_REGION")
	if region == "" {
		region = conf["region"]
		if region == "" {
			region = "us-east-1"
		}
	}

	creds := credentials.NewChainCredentials([]credentials.Provider{
		&credentials.StaticProvider{Value: credentials.Value{
			AccessKeyID:     access_key,
			SecretAccessKey: secret_key,
			SessionToken:    session_token,
		}},
		&credentials.EnvProvider{},
		&credentials.SharedCredentialsProvider{Filename: "", Profile: ""},
		&ec2rolecreds.EC2RoleProvider{Client: ec2metadata.New(session.New())},
	})

	s3conn := s3.New(session.New(&aws.Config{
		Credentials: creds,
		Endpoint:    aws.String(endpoint),
		Region:      aws.String(region),
	}))

	_, err := s3conn.HeadBucket(&s3.HeadBucketInput{Bucket: &bucket})
	if err != nil {
		return nil, fmt.Errorf("unable to access bucket '%s': %v", bucket, err)
	}

	s := &S3Backend{
		client: s3conn,
		bucket: bucket,
	}
	return s, nil
}

// Put is used to insert or update an entry
func (s *S3Backend) Put(entry *Entry) error {
	defer metrics.MeasureSince([]string{"s3", "put"}, time.Now())

	_, err := s.client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(entry.Key),
		Body:   bytes.NewReader(entry.Value),
	})

	if err != nil {
		return err
	}

	return nil
}

// Get is used to fetch an entry
func (s *S3Backend) Get(key string) (*Entry, error) {
	defer metrics.MeasureSince([]string{"s3", "get"}, time.Now())

	resp, err := s.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if awsErr, ok := err.(awserr.RequestFailure); ok {
		// Return nil on 404s, error on anything else
		if awsErr.StatusCode() == 404 {
			return nil, nil
		} else {
			return nil, err
		}
	}

	data := make([]byte, *resp.ContentLength)
	_, err = io.ReadFull(resp.Body, data)
	if err != nil {
		return nil, err
	}

	ent := &Entry{
		Key:   key,
		Value: data,
	}

	return ent, nil
}

// Delete is used to permanently delete an entry
func (s *S3Backend) Delete(key string) error {
	defer metrics.MeasureSince([]string{"s3", "delete"}, time.Now())

	_, err := s.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return err
	}

	return nil
}

// List is used to list all the keys under a given
// prefix, up to the next prefix.
func (s *S3Backend) List(prefix string) ([]string, error) {
	defer metrics.MeasureSince([]string{"s3", "list"}, time.Now())

	resp, err := s.client.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, err
	}

	keys := []string{}
	for _, key := range resp.Contents {
		key := strings.TrimPrefix(*key.Key, prefix)

		if i := strings.Index(key, "/"); i == -1 {
			// Add objects only from the current 'folder'
			keys = append(keys, key)
		} else if i != -1 {
			// Add truncated 'folder' paths
			keys = appendIfMissing(keys, key[:i+1])
		}
	}

	sort.Strings(keys)

	return keys, nil
}

func appendIfMissing(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}
