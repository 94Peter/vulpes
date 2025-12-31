package interceptor

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	mockResponse = "response"
)

const handlerErrorKey contextKey = "handler-error"

// --- Mocks and Helpers ---

// mockHandler is a mock grpc.UnaryHandler for testing.
func mockHandler(ctx context.Context, req any) (any, error) {
	// Check if the handler is supposed to panic
	if p, ok := req.(string); ok && p == "panic" {
		panic("handler panicked")
	}
	// Check if the handler is supposed to be slow
	if d, ok := req.(time.Duration); ok {
		time.Sleep(d)
	}
	// Return a predefined error from the context, if any
	if err, ok := ctx.Value(handlerErrorKey).(error); ok {
		return nil, err
	}
	return mockResponse, nil
}

// mockValidator is a mock for the validator interface.
type mockValidator struct {
	err error
}

func (m *mockValidator) Validate() error {
	return m.err
}

func (m *mockValidator) ValidateAll() error {
	return m.err
}

// mockMultiError simulates a validation error with multiple fields.
type mockMultiError struct {
	errs []error
}

func (*mockMultiError) Error() string {
	return "multiple errors"
}

func (m *mockMultiError) Errors() []error {
	return m.errs
}

// mockFieldError simulates a single field validation error.
type mockFieldError struct {
	field string
	reas  string
}

func (m *mockFieldError) Error() string {
	return m.reas
}

func (m *mockFieldError) Field() string {
	return m.field
}

func (m *mockFieldError) Reason() string {
	return m.reas
}

func (*mockFieldError) Key() bool {
	return false
}

func (*mockFieldError) Cause() error {
	return nil
}

func (*mockFieldError) ErrorName() string {
	return "FieldError"
}

var mockInfo = &grpc.UnaryServerInfo{
	FullMethod: "/test.Service/TestMethod",
}

// --- Tests ---

func TestRequestIDInterceptor(t *testing.T) {
	t.Run("GeneratesNewRequestID", func(t *testing.T) {
		var handlerCtx context.Context
		handler := func(ctx context.Context, _ any) (any, error) {
			handlerCtx = ctx
			return mockResponse, nil
		}

		_, err := requestIDInterceptor(context.Background(), "request", mockInfo, handler)
		require.NoError(t, err)

		reqID := GetRequestID(handlerCtx)
		assert.NotEmpty(t, reqID, "request ID should be generated")
		_, err = uuid.Parse(reqID)
		assert.NoError(t, err, "generated request ID should be a valid UUID")
	})

	t.Run("UsesExistingRequestID", func(t *testing.T) {
		existingID := "existing-id-123"
		md := metadata.New(map[string]string{RequestIDKey: existingID})
		ctx := metadata.NewIncomingContext(context.Background(), md)

		var handlerCtx context.Context
		handler := func(ctx context.Context, _ any) (any, error) {
			handlerCtx = ctx
			return mockResponse, nil
		}

		_, err := requestIDInterceptor(ctx, "request", mockInfo, handler)
		require.NoError(t, err)

		reqID := GetRequestID(handlerCtx)
		assert.Equal(t, existingID, reqID, "should use existing request ID from metadata")
	})
}

func TestRecoveryInterceptor(t *testing.T) {
	t.Run("RecoversFromPanic", func(t *testing.T) {
		_, err := recoveryInterceptor(context.Background(), "panic", mockInfo, mockHandler)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok, "error should be a gRPC status")
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "internal error: handler panicked")
	})

	t.Run("NoPanic", func(t *testing.T) {
		resp, err := recoveryInterceptor(context.Background(), "request", mockInfo, mockHandler)
		require.NoError(t, err)
		assert.Equal(t, mockResponse, resp)
	})
}

func TestRateLimitInterceptor(t *testing.T) {
	// Use a new rate limiter for this test to avoid state from other tests
	limiter := newIPRateLimiter(rate.Limit(1), 2) // 1 token per second, burst of 2
	interceptor := limiter.UnaryServerInterceptor()

	p := &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}}
	ctx := peer.NewContext(context.Background(), p)

	// First two requests should be allowed
	_, err := interceptor(ctx, "req1", mockInfo, mockHandler)
	require.NoError(t, err)
	_, err = interceptor(ctx, "req2", mockInfo, mockHandler)
	require.NoError(t, err)

	// Third request should be rate limited
	_, err = interceptor(ctx, "req3", mockInfo, mockHandler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, st.Code())

	// Wait for the limiter to replenish a token
	time.Sleep(1 * time.Second)

	// Fourth request should now be allowed
	_, err = interceptor(ctx, "req4", mockInfo, mockHandler)
	assert.NoError(t, err)
}

func TestValidateInterceptor(t *testing.T) {
	t.Run("NoValidator", func(t *testing.T) {
		req := "not a validator"
		resp, err := validateUnaryInterceptor(context.Background(), req, mockInfo, mockHandler)
		require.NoError(t, err)
		assert.Equal(t, mockResponse, resp)
	})

	t.Run("ValidRequest", func(t *testing.T) {
		req := &mockValidator{err: nil}
		resp, err := validateUnaryInterceptor(context.Background(), req, mockInfo, mockHandler)
		require.NoError(t, err)
		assert.Equal(t, mockResponse, resp)
	})

	t.Run("SingleError", func(t *testing.T) {
		req := &mockValidator{err: errors.New("validation failed: name is required")}
		_, err := validateUnaryInterceptor(context.Background(), req, mockInfo, mockHandler)
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Equal(t, "Validation failed", st.Message())
		details := st.Details()
		require.Len(t, details, 1)
	})

	t.Run("MultiError", func(t *testing.T) {
		req := &mockValidator{
			err: &mockMultiError{
				errs: []error{
					&mockFieldError{field: "name", reas: "name is required"},
					&mockFieldError{field: "email", reas: "email is invalid"},
				},
			},
		}
		_, err := validateUnaryInterceptor(context.Background(), req, mockInfo, mockHandler)
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		details := st.Details()
		require.Len(t, details, 1)
	})
}

func TestLoggerInterceptor(t *testing.T) {
	// Note: Testing the actual log output would require a more complex setup
	// with a mocked logger. Here, we primarily test that the handler is called
	// and errors are propagated correctly.

	t.Run("SuccessfulCall", func(t *testing.T) {
		resp, err := loggerInterceptor(context.Background(), "request", mockInfo, mockHandler)
		require.NoError(t, err)
		assert.Equal(t, mockResponse, resp)
	})

	t.Run("FailedCall", func(t *testing.T) {
		expectedErr := status.Error(codes.NotFound, "not found")
		ctx := context.WithValue(context.Background(), handlerErrorKey, expectedErr)
		_, err := loggerInterceptor(ctx, "request", mockInfo, mockHandler)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("SlowCall", func(t *testing.T) {
		// We don't need to modify the constant. We just need to sleep
		// for a duration longer than the constant to trigger the slow log.
		// Add a small buffer to account for any scheduling delays.
		slowCallDuration := slowThreshold + (10 * time.Millisecond)

		// This call will be logged as slow
		_, err := loggerInterceptor(context.Background(), slowCallDuration, mockInfo, mockHandler)
		assert.NoError(t, err)
	})
}
