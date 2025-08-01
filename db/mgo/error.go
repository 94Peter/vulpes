// Package mgo provides a high-level abstraction layer over the official MongoDB Go driver,
// simplifying connection management, document operations, and schema definitions.
package mgo

import (
	"errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Standardized errors for the mgo package, providing consistent error types for common database operations.
var (
	// ErrInvalidDocument is returned when a document fails validation or is otherwise malformed.
	ErrInvalidDocument = errors.New("invalid document")
	// ErrNotConnected is returned when a database operation is attempted before a connection is established.
	ErrNotConnected = errors.New("mongodb not connected")
	// ErrConnectionFailed is returned when the initial connection attempt to the MongoDB server fails.
	ErrConnectionFailed = errors.New("mongodb connection failed")
	// ErrPingFailed is returned when the client fails to ping the MongoDB server to verify the connection.
	ErrPingFailed = errors.New("mongodb ping failed")
	// ErrCreateIndexFailed is returned when an attempt to create one or more indexes fails.
	ErrCreateIndexFailed = errors.New("mongodb create index failed")
	// ErrListCollectionFailed is returned when listing the collections in the database fails.
	ErrListCollectionFailed = errors.New("mongodb list collection failed")
	// ErrWriteFailed is returned when a write operation fails.
	ErrWriteFailed = errors.New("mongodb write failed")
	// ErrReadFailed is returned when a read operation fails.
	ErrReadFailed = errors.New("mongodb read failed")

	StatusMongoDBInvalidDocument      = status.New(codes.InvalidArgument, "mongodb invalid document")
	StatusMongoDBNotConnected         = status.New(codes.Aborted, "mongodb not connected")
	StatusMongoDBConnectionFailed     = status.New(codes.Aborted, "mongodb connection failed")
	StatusMongoDBPingFailed           = status.New(codes.Aborted, "mongodb ping failed")
	StatusMongoDBCreateIndexFailed    = status.New(codes.Aborted, "mongodb create index failed")
	StatusMongoDBListCollectionFailed = status.New(codes.Aborted, "mongodb list collection failed")
	StatusMongoDBWriteFailed          = status.New(codes.Internal, "mongodb write failed")
	StatusMongoDBReadFailed           = status.New(codes.Internal, "mongodb read failed")
)

func ToStatus(err error) *status.Status {
	if err == nil {
		return nil
	}
	var baseSt *status.Status

	switch {
	case errors.Is(err, ErrInvalidDocument):
		baseSt = StatusMongoDBInvalidDocument
	case errors.Is(err, ErrNotConnected):
		baseSt = StatusMongoDBNotConnected
	case errors.Is(err, ErrConnectionFailed):
		baseSt = StatusMongoDBConnectionFailed
	case errors.Is(err, ErrPingFailed):
		baseSt = StatusMongoDBPingFailed
	case errors.Is(err, ErrCreateIndexFailed):
		baseSt = StatusMongoDBCreateIndexFailed
	case errors.Is(err, ErrListCollectionFailed):
		baseSt = StatusMongoDBListCollectionFailed
	case errors.Is(err, ErrWriteFailed):
		baseSt = StatusMongoDBWriteFailed
	case errors.Is(err, ErrReadFailed):
		baseSt = StatusMongoDBReadFailed
	default:
		return status.New(codes.Internal, err.Error())
	}
	unwrapErr := errors.Unwrap(err)
	if unwrapErr == nil {
		unwrapErr = err
	}
	// Add more details to the status, such as the type of violation and a description.
	st, myErr := baseSt.WithDetails(
		&errdetails.PreconditionFailure{
			Violations: []*errdetails.PreconditionFailure_Violation{
				{
					Type:        "MGO",
					Subject:     unwrapErr.Error(),
					Description: err.Error(),
				},
			},
		},
	)
	if myErr != nil {
		// If adding details fails, return the original base status.
		return baseSt
	}
	return st
}
