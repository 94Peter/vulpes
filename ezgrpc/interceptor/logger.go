// Package interceptor provides gRPC unary server interceptors for common concerns
// such as logging, metrics, rate limiting, and panic recovery.
package interceptor

import (
	"context"
	"time"

	"github.com/94peter/vulpes/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// slowThreshold defines the duration after which a gRPC request is considered slow.
const slowThreshold = time.Second * 3

// loggerInterceptor is a gRPC unary server interceptor that logs incoming requests and their outcomes.
// It records the method, request ID, peer address, status code, and duration.
// It also logs errors and slow requests with a higher severity.
var loggerInterceptor grpc.UnaryServerInterceptor = func(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp any, err error) {
	startTime := time.Now()
	requestID := GetRequestID(ctx) // Depends on requestIDInterceptor being executed first

	// Log the incoming request
	reqFields := []log.Field{
		log.String("grpc.method", info.FullMethod),
		log.String("request_id", requestID),
	}
	if p, ok := peer.FromContext(ctx); ok {
		reqFields = append(reqFields, log.String("peer.address", p.Addr.String()))
	}
	log.Info("gRPC request received", reqFields...)

	// Call the next handler in the chain
	resp, err = handler(ctx, req)

	// Log the completion of the request
	duration := time.Since(startTime)
	statusCode := status.Code(err)

	resFields := []log.Field{
		log.String("grpc.method", info.FullMethod),
		log.String("request_id", requestID),
		log.String("grpc.status_code", statusCode.String()),
		log.Duration("grpc.duration", duration),
	}

	if err != nil {
		errorFields := append(resFields, log.String("error", err.Error()))
		if statusCode == codes.Internal || statusCode == codes.Unknown {
			log.Error("gRPC request failed", errorFields...)
		} else {
			log.Info("gRPC request completed with client error", errorFields...)
		}
	} else if duration > slowThreshold {
		log.Error("gRPC request slow", resFields...)
	} else {
		log.Info("gRPC request completed", resFields...)
	}

	return resp, err
}
