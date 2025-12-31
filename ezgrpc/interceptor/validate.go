// Package interceptor provides gRPC unary server interceptors for common concerns
// such as logging, metrics, rate limiting, and panic recovery.
package interceptor

import (
	"context"
	"strings"

	epb "google.golang.org/genproto/googleapis/rpc/errdetails"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// validator is an interface that defines the validation methods.
// protoc-gen-validate generates this method for your messages.
type validator interface {
	Validate() error
	ValidateAll() error // Changed to ValidateAll
}

// fieldError is an interface for accessing field-specific error details.
type fieldError interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
}

// DisableValidateInterceptor disables the validation interceptor.
func DisableValidateInterceptor() {
	enableValidate = false
}

var (
	enableValidate = true

	// validateUnaryInterceptor is a gRPC unary server interceptor that automatically validates incoming requests.
	// It checks if the request message implements the validator interface and, if so, runs the validation.
	// If validation fails, it returns a gRPC error with detailed information about the validation failures.
	validateUnaryInterceptor grpc.UnaryServerInterceptor = func(
		ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (any, error) {
		// Check if the request implements the Validator interface
		if !enableValidate {
			return handler(ctx, req)
		}
		v, ok := req.(validator)
		if !ok {
			return handler(ctx, req)
		}
		// If validation passes, proceed to the next handler
		err := v.ValidateAll()
		if err == nil {
			return handler(ctx, req)
		}
		// If validation fails, create a gRPC status error with details
		st := status.New(codes.InvalidArgument, "Validation failed")
		br := &epb.BadRequest{} // Create a BadRequest message

		// Check if the error is a MultiError type
		if multiErr, isMultiErr := err.(interface {
			Errors() []error
		}); isMultiErr {
			for _, singleErr := range multiErr.Errors() {
				// Try to convert each individual error to a *validate.FieldError
				if fieldErr, isFieldErr := singleErr.(fieldError); isFieldErr {
					// If it's a FieldError, use its Field and Reason directly
					br.FieldViolations = append(br.FieldViolations, &epb.BadRequest_FieldViolation{
						Field:       fieldErr.Field(),
						Description: fieldErr.Reason(),
					})
				} else {
					br.FieldViolations = append(br.FieldViolations, &epb.BadRequest_FieldViolation{
						Field:       "unknown", // Or you can try to parse from singleErr.Error()
						Description: singleErr.Error(),
					})
				}
			}
		} else {
			// If the error is not a MultiError type (e.g., only one error, or a non-validate error)
			// Try to parse it as a FieldViolation, or handle it as a generic error
			parts := strings.SplitN(err.Error(), ": ", 2)
			field := "unknown"
			description := err.Error()
			if len(parts) == 2 {
				field = parts[0]
				description = parts[1]
			}
			br.FieldViolations = append(br.FieldViolations, &epb.BadRequest_FieldViolation{
				Field:       field,
				Description: description,
			})
		}

		// Attach the BadRequest message as details to the gRPC status
		stWithDetails, err := st.WithDetails(br)
		if err != nil {
			// If attaching details fails (e.g., due to size), fall back to a simple InvalidArgument error
			return nil, status.Errorf(codes.InvalidArgument, "Validation failed: %s", err.Error())
		}
		// Return the gRPC error with details
		return nil, stWithDetails.Err()
	}
)
