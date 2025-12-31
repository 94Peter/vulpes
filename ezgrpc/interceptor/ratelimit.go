// Package interceptor provides gRPC unary server interceptors for common concerns
// such as logging, metrics, rate limiting, and panic recovery.
package interceptor

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// ipRateLimiter holds the rate limiters for each IP address.
// NOTE: In a production environment with many clients, this map can grow indefinitely.
// Consider using a library with automatic cleanup of old entries (e.g., based on LRU).
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
}

// newIPRateLimiter creates a new rate limiter for IP addresses.
func newIPRateLimiter(r rate.Limit, b int) *ipRateLimiter {
	return &ipRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    b,
	}
}

// getLimiter returns the rate limiter for the given IP address.
func (l *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, exists := l.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(l.rate, l.burst)
		l.limiters[ip] = limiter
	}

	return limiter
}

// UnaryServerInterceptor returns a new unary server interceptor that performs rate limiting.
func (l *ipRateLimiter) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		p, ok := peer.FromContext(ctx)
		if !ok {
			return nil, status.Error(codes.Internal, "could not retrieve peer information")
		}

		limiter := l.getLimiter(p.Addr.String())
		if !limiter.Allow() {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded for %s", p.Addr.String())
		}

		return handler(ctx, req)
	}
}

var (
	// Default to 10 requests per second with a burst of 20.
	rateLimiter          = newIPRateLimiter(10, 20)
	rateLimitInterceptor = rateLimiter.UnaryServerInterceptor()
)
