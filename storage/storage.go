package storage

import (
	"context"
	"io"
	"time"
)

type Storage interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string) error
	SignedDownloadUrl(ctx context.Context, key string, expires time.Duration) (string, error)
}

func New(ctx context.Context, opts ...Option) (Storage, error) {
	return newR2Storage(ctx, opts...)
}
