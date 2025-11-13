// Package interceptor provides gRPC unary server interceptors for common concerns
// such as logging, metrics, rate limiting, and panic recovery.
package interceptor

import (
	"context"

	"github.com/94peter/vulpes/log"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// RequestIDKey is the key used for the request ID in gRPC metadata.
const RequestIDKey = "x-request-id"

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

// ctxKeyRequestID is the context key for the request ID.
const ctxKeyRequestID = contextKey("request-id")

// withRequestID embeds the request ID into the context.
func withRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, requestID)
}

// GetRequestID extracts the request ID from the context, for use in logging and tracing.
func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(ctxKeyRequestID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// requestIDInterceptor is a gRPC unary server interceptor that ensures each request has a unique ID.
// It checks for an existing request ID in the incoming metadata, and generates a new one if not present.
// The request ID is then added to the context for downstream use.
var requestIDInterceptor grpc.UnaryServerInterceptor = func(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	var requestID string
	if ok {
		if rid := md.Get(RequestIDKey); len(rid) > 0 {
			requestID = rid[0]
		}
	}

	if requestID == "" {
		requestID = uuid.NewString()
	}

	// Add the request ID to the context for downstream handlers.
	ctx = withRequestID(ctx, requestID)

	// Example of logging the request ID.
	if p, ok := peer.FromContext(ctx); ok {
		log.Info("gRPC request received", log.String("peer.address", p.Addr.String()), log.String("grpc.method", info.FullMethod), log.String("request_id", requestID))
	}

	return handler(ctx, req)
}
