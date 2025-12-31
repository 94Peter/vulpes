// Package log provides a simplified and opinionated interface for structured logging,
// built on top of the high-performance zap logger.
package log

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Field is a type alias for zapcore.Field, representing a single key-value pair in a structured log.
// Using a type alias provides a convenient, shorter way to reference this type.
type Field = zapcore.Field

// --- Field Constructors ---
// These functions are convenient wrappers around zap's field constructors,
// allowing for a more concise and readable logging API.

// Int creates a Field with an integer value.
func Int(key string, val int) Field {
	return zap.Int(key, val)
}

// Int32 creates a Field with an int32 value.
func Int32(key string, val int32) Field {
	return zap.Int32(key, val)
}

// Int64 creates a Field with an int64 value.
func Int64(key string, val int64) Field {
	return zap.Int64(key, val)
}

// Uint creates a Field with an unsigned integer value.
func Uint(key string, val uint) Field {
	return zap.Uint(key, val)
}

// Uint32 creates a Field with a uint32 value.
func Uint32(key string, val uint32) Field {
	return zap.Uint32(key, val)
}

// Uint64 creates a Field with a uint64 value.
func Uint64(key string, val uint64) Field {
	return zap.Uint64(key, val)
}

// Uintptr creates a Field with a uintptr value.
func Uintptr(key string, val uintptr) Field {
	return zap.Uintptr(key, val)
}

// Float64 creates a Field with a float64 value.
func Float64(key string, val float64) Field {
	return zap.Float64(key, val)
}

// Bool creates a Field with a boolean value.
func Bool(key string, val bool) Field {
	return zap.Bool(key, val)
}

// String creates a Field with a string value.
func String(key string, val string) Field {
	return zap.String(key, val)
}

// ByteString creates a Field with a byte slice value.
func ByteString(key string, val []byte) Field {
	return zap.ByteString(key, val)
}

// Stringer creates a Field with a value that implements the fmt.Stringer interface.
func Stringer(key string, val fmt.Stringer) Field {
	return zap.Stringer(key, val)
}

// Time creates a Field with a time.Time value.
func Time(key string, val time.Time) Field {
	return zap.Time(key, val)
}

// Duration creates a Field with a time.Duration value.
func Duration(key string, val time.Duration) Field {
	return zap.Duration(key, val)
}

// Err creates a Field with an error value. It is a shorthand for zap.Error.
func Err(err error) Field {
	return zap.Error(err)
}

// Any creates a Field with a value of any type, using reflection for serialization.
// It is useful for logging complex types like structs, slices, and maps.
func Any(key string, val any) Field {
	return zap.Any(key, val)
}
