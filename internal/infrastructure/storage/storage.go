package storage

import (
	"context"
	"io"
)

// FileStorage abstracts file upload/download — backed by local disk in dev, S3 in prod.
type FileStorage interface {
	Upload(ctx context.Context, key string, r io.Reader) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}
