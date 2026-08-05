package storage

import (
	"context"
	"io"
)

type PutObjectInput struct {
	TenantID    string
	Module      string
	ObjectKey   string
	ContentType string
	Body        io.Reader
}

type PutObjectOutput struct {
	Provider  string
	ObjectKey string
}

type GetObjectInput struct {
	ObjectKey string
}

type ObjectStorage interface {
	Put(ctx context.Context, in PutObjectInput) (PutObjectOutput, error)
	Open(ctx context.Context, in GetObjectInput) (io.ReadCloser, error)
	Delete(ctx context.Context, in GetObjectInput) error
}

