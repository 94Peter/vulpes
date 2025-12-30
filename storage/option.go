package storage

import "go.opentelemetry.io/otel/trace"

type Option func(*r2Config)

func WithEndpoint(endpoint string) Option {
	return func(c *r2Config) {
		c.r2EndpointURL = endpoint
	}
}

func WithAccessKey(accessKey string) Option {
	return func(c *r2Config) {
		c.r2AccessKeyID = accessKey
	}
}

func WithSecretKey(secretKey string) Option {
	return func(c *r2Config) {
		c.r2SecretAccessKey = secretKey
	}
}

func WithBucket(bucket string) Option {
	return func(c *r2Config) {
		c.r2BucketName = bucket
	}
}

func withTracer(tracer trace.Tracer) Option {
	return func(rc *r2Config) {
		rc.tracer = tracer
	}
}
