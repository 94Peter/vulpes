// Package interceptor provides gRPC unary server interceptors for common concerns
// such as logging, metrics, rate limiting, and panic recovery.
package interceptor

import (
	"github.com/94peter/vulpes/log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
)

// recoveryHandler is a function that recovers from panics and returns a gRPC error.
// It logs the panic and returns a gRPC status with an internal error code.
func recoveryHandler(p any) error {
	log.Error("panic occurred and recovery", log.Any("error", p))
	return status.Errorf(codes.Internal, "internal error: %v", p)
}

// recoveryInterceptor is a gRPC unary server interceptor that recovers from panics.
// It uses the recoveryHandler to process the panic and return a gRPC error.
var recoveryInterceptor = grpc_recovery.UnaryServerInterceptor(
	grpc_recovery.WithRecoveryHandler(recoveryHandler),
)
