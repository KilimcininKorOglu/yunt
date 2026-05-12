package storage

import (
	"context"
	"io"
)

// Backend provides an abstraction for storing and retrieving binary content.
type Backend interface {
	Store(ctx context.Context, key string, content io.Reader, size int64) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}
