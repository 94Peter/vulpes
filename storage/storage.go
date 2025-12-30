package storage

import (
	"context"
	"io"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Storage interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string) error
	SignedDownloadUrl(ctx context.Context, key string, expires time.Duration) (string, error)
}

func New(ctx context.Context, opts ...Option) (Storage, error) {
	tracer := otel.Tracer("Storage")
	opts = append(opts, withTracer(tracer))
	return newR2Storage(ctx, opts...)
}

func spanErrorHandler(err error, span trace.Span) error {
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "ok")
	}
	return err
}
