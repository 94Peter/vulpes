// Package codec provides a flexible framework for encoding and decoding data structures.
// It supports multiple encoding formats (GOB, MessagePack) and uses generics for type safety.
package codec

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
)

// gobCodec implements the Codec interface using Go's built-in GOB serialization.
// The binary output is then encoded into a Base64 string for safe transport.
type gobCodec[T any] struct{}

// Encode serializes the value `v` first using GOB, then encodes the result into a Base64 string.
func (_ *gobCodec[T]) Encode(v T) (string, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(v)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrGobEncodeFailed, err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// Decode first decodes the Base64 string `s` into bytes, then deserializes the bytes using GOB.
func (_ *gobCodec[T]) Decode(s string) (T, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return *new(T), fmt.Errorf("%w: %w", ErrBase64DecodeFailed, err)
	}
	var buf bytes.Buffer
	buf.Write(data)
	var result T
	err = gob.NewDecoder(&buf).Decode(&result)
	if err != nil {
		return *new(T), fmt.Errorf("%w: %w", ErrGobDecodeFailed, err)
	}
	return result, nil
}

// Method returns the GOB codec method identifier.
func (_ *gobCodec[T]) Method() CodecMethod {
	return GOB
}

// encodeGOB is a package-level helper function for GOB encoding.
func encodeGOB[T any](v T) (string, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(v)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrGobEncodeFailed, err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// decodeGOB is a package-level helper function for GOB decoding.
func decodeGOB[T any](s string) (T, error) {
	var out T
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return out, fmt.Errorf("%w: %w", ErrBase64DecodeFailed, err)
	}
	err = gob.NewDecoder(bytes.NewReader(data)).Decode(&out)
	if err != nil {
		return out, fmt.Errorf("%w: %w", ErrGobDecodeFailed, err)
	}
	return out, nil
}
