// Package codec provides a flexible framework for encoding and decoding data structures.
// It supports multiple encoding formats (GOB, MessagePack) and uses generics for type safety.
// The primary use case is to serialize complex types into a string format for transport or storage,
// for example, in session data.
package codec

import (
	"fmt"

	"github.com/94peter/vulpes/log"
)

// Codec defines the interface for any encoding/decoding implementation.
// It uses generics to ensure type safety between encoding and decoding operations.
type Codec[T any] interface {
	// Encode serializes a given value of type T into a string.
	Encode(v T) (string, error)
	// Decode deserializes a string back into a value of type T.
	Decode(s string) (T, error)
	// Method returns the specific encoding method used by the codec.
	Method() CodecMethod
}

// CodecMethod is a type that defines the available encoding formats.
// Using a custom type provides better type safety than using plain strings.
type CodecMethod string

// Constants for the supported encoding methods.
const (
	GOB     CodecMethod = "gob"     // GOB is a Go-specific binary encoding format.
	MSGPACK CodecMethod = "msgpack" // MessagePack is a fast, compact binary serialization format.
)

// defaultCodeMethod holds the globally configured encoding method. It defaults to GOB.
var defaultCodeMethod CodecMethod = GOB

// Encode serializes a value of any type into a string using the globally configured default codec.
// The value is first encoded into a binary format (GOB or MessagePack) and then into a Base64 string.
func Encode(v any) (string, error) {
	log.Debugf("Using codec method for encoding: %s", defaultCodeMethod)
	switch defaultCodeMethod {
	case GOB:
		return encodeGOB(v)
	case MSGPACK:
		return encodeMsgPack(v)
	default:
		return "", fmt.Errorf("%w: unsupported codec method [%s]", ErrUnknownCodecMethod, defaultCodeMethod)
	}
}

// Decode deserializes a string back into a specific type T using the globally configured default codec.
// The string is expected to be a Base64 representation of the binary data (GOB or MessagePack).
func Decode[T any](s string) (T, error) {
	switch defaultCodeMethod {
	case GOB:
		return decodeGOB[T](s)
	case MSGPACK:
		return decodeMsgPack[T](s)
	default:
		return *new(T), fmt.Errorf("%w: unsupported codec method [%s]", ErrUnknownCodecMethod, defaultCodeMethod)
	}
}
