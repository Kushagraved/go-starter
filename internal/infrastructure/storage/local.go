package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type localStorage struct {
	baseDir string
}

// NewLocal returns a FileStorage backed by the local filesystem.
func NewLocal(baseDir string) FileStorage {
	_ = os.MkdirAll(baseDir, 0755)
	return &localStorage{baseDir: baseDir}
}

func (l *localStorage) Upload(_ context.Context, key string, r io.Reader) error {
	path := filepath.Join(l.baseDir, key)
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (l *localStorage) Download(_ context.Context, key string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(l.baseDir, key))
}

func (l *localStorage) Delete(_ context.Context, key string) error {
	return os.Remove(filepath.Join(l.baseDir, key))
}
