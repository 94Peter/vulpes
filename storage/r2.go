package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type r2Config struct {
	tracer            trace.Tracer
	r2EndpointURL     string
	r2AccessKeyID     string
	r2SecretAccessKey string
	r2Region          string
	r2BucketName      string
}

var defaultR2Config = r2Config{
	r2Region: "auto",
}

// R2 使用 'auto' 作為預設 Region，這是連線 R2 所需的。
const R2Region = "auto"

// 測試用的 Bucket 和 Object 設定
// const R2BucketName = "seanaigent"
// const R2ObjectName = "test-upload/hello-r2.txt"
// const UploadContent = "Hello from Golang and Cloudflare R2!"

func newR2Storage(ctx context.Context, opts ...Option) (*r2Storage, error) {
	for _, opt := range opts {
		opt(&defaultR2Config)
	}
	if defaultR2Config.r2EndpointURL == "" {
		return nil, fmt.Errorf("r2 endpoint url is required")
	}
	if defaultR2Config.r2AccessKeyID == "" {
		return nil, fmt.Errorf("r2 access key id is required")
	}
	if defaultR2Config.r2SecretAccessKey == "" {
		return nil, fmt.Errorf("r2 secret access key is required")
	}
	if defaultR2Config.r2BucketName == "" {
		return nil, fmt.Errorf("r2 bucket name is required")
	}
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(R2Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				defaultR2Config.r2AccessKeyID, defaultR2Config.r2SecretAccessKey, ""),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("fail load config: %w", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		// 設定 R2 的 Base Endpoint URL
		o.BaseEndpoint = aws.String(defaultR2Config.r2EndpointURL)

		// 確保簽名區域 (Signing Region) 正確設定為 R2 所需的 'auto'
		// 此設置會覆蓋 cfg 中的 Region 設定。
		o.Region = "auto"
	})
	return &r2Storage{
		client: client,
		bucket: defaultR2Config.r2BucketName,
		tracer: defaultR2Config.tracer,
	}, nil
}

type r2Storage struct {
	tracer trace.Tracer
	client *s3.Client
	bucket string
}

func (r *r2Storage) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
	ctx, span := r.startTraceSpan(ctx, "storage.upload", attribute.String("storage.func", "Upload"))
	defer span.End()
	span.SetAttributes(attribute.String("storage.bucket", r.bucket))
	span.SetAttributes(attribute.String("storage.key", key))
	span.SetAttributes(attribute.String("storage.contentType", contentType))
	uploadInput := &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        body, // 將字串作為檔案內容
		ContentType: aws.String(contentType),
	}
	_, err := r.client.PutObject(ctx, uploadInput)
	return spanErrorHandler(err, span)
}

func (r *r2Storage) SignedDownloadUrl(ctx context.Context, key string, expires time.Duration) (string, error) {
	ctx, span := r.startTraceSpan(
		ctx,
		"storage.signed_download_url",
		attribute.String("storage.func", "SignedDownloadUrl"),
	)
	defer span.End()
	// 創建一個 PresignClient
	presigner := s3.NewPresignClient(r.client)
	// 設定預簽名 GetObject 的輸入參數
	// 這裡設定 URL 的有效期限為 5 分鐘。
	input := &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}
	span.SetAttributes(attribute.String("storage.bucket", r.bucket))
	span.SetAttributes(attribute.String("storage.key", key))
	resp, err := presigner.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expires
	})
	if err != nil {
		return "", spanErrorHandler(err, span)
	}
	span.SetAttributes(attribute.String("storage.url", resp.URL))
	return resp.URL, spanErrorHandler(nil, span)
}

var r2StorageService = "cloudflare_r2"

func (r *r2Storage) startTraceSpan(
	ctx context.Context,
	name string,
	attributes ...attribute.KeyValue,
) (context.Context, trace.Span) {
	ctx, span := r.tracer.Start(ctx, name, trace.WithSpanKind(trace.SpanKindClient))
	span.SetAttributes(
		append([]attribute.KeyValue{
			attribute.String("storage.service", r2StorageService),
		}, attributes...)...,
	)
	return ctx, span
}
