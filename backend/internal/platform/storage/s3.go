package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Storage struct {
	client *s3.Client
	bucket string
}

func NewS3Storage(region, bucket, accessKey, secretKey, endpoint string) (*S3Storage, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if endpoint != "" {
			return aws.Endpoint{
				URL:           endpoint,
				SigningRegion: region,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, err
	}

	return &S3Storage{
		client: s3.NewFromConfig(cfg, func(o *s3.Options) {
			if endpoint != "" {
				o.UsePathStyle = true
			}
		}),
		bucket: bucket,
	}, nil
}

func (s *S3Storage) Put(ctx context.Context, in PutObjectInput) (PutObjectOutput, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(in.ObjectKey),
		Body:        in.Body,
		ContentType: aws.String(in.ContentType),
	})
	if err != nil {
		return PutObjectOutput{}, fmt.Errorf("s3 put failed: %w", err)
	}
	return PutObjectOutput{
		Provider:  "s3",
		ObjectKey: in.ObjectKey,
	}, nil
}

func (s *S3Storage) Open(ctx context.Context, in GetObjectInput) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(in.ObjectKey),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 get failed: %w", err)
	}
	return out.Body, nil
}

func (s *S3Storage) Delete(ctx context.Context, in GetObjectInput) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(in.ObjectKey),
	})
	if err != nil {
		return fmt.Errorf("s3 delete failed: %w", err)
	}
	return nil
}

