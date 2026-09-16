package infrastructure

import (
	"context"
	"time"
)

// FileStorage abstracts object storage. Backed by S3 (LocalStack locally).
type FileStorage interface {
	// StoreFile writes data at key and returns the stored path.
	StoreFile(ctx context.Context, bucket, key string, data []byte, params *FileStorageParams) (string, error)
	// GetFile returns constants.RecordDoesntExists when the object is absent.
	GetFile(ctx context.Context, bucket, key string) ([]byte, error)
	RemoveFile(ctx context.Context, bucket, key string) error
	ListObjects(ctx context.Context, bucket, prefix string) ([]ListObjectsResponse, error)
}

// FileStorageParams carries optional per-call settings.
type FileStorageParams struct {
	ContentType string
}

// ListObjectsResponse is one entry returned by ListObjects.
type ListObjectsResponse struct {
	FileName     string
	LastModified time.Time
}
