package storage

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config holds configuration for S3 storage.
type S3Config struct {
	Bucket    string
	Region    string
	Prefix    string
	Endpoint  string
	AccessKey string
	SecretKey string
}

// S3Backend stores content in an S3-compatible object store.
type S3Backend struct {
	client *s3.Client
	bucket string
	prefix string
}

// NewS3Backend creates a new S3 storage backend.
func NewS3Backend(ctx context.Context, cfg S3Config) (*S3Backend, error) {
	var opts []func(*awsconfig.LoadOptions) error

	if cfg.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.Region))
	}

	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var clientOpts []func(*s3.Options)
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, clientOpts...)

	return &S3Backend{
		client: client,
		bucket: cfg.Bucket,
		prefix: cfg.Prefix,
	}, nil
}

func (s *S3Backend) Store(ctx context.Context, key string, content io.Reader, size int64) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.objectKey(key)),
		Body:   content,
	}
	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}
	return nil
}

func (s *S3Backend) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.objectKey(key)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get from S3: %w", err)
	}
	return output.Body, nil
}

func (s *S3Backend) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.objectKey(key)),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}
	return nil
}

func (s *S3Backend) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.objectKey(key)),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (s *S3Backend) objectKey(key string) string {
	if s.prefix != "" {
		return path.Join(s.prefix, key)
	}
	return key
}
