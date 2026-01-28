# Codec Package

`codec` is a Go package that provides a flexible and type-safe framework for encoding and decoding data structures. It is designed to serialize complex Go types into a string representation, making them suitable for transport over the network or for storage in systems that require a string format (e.g., session data in cookies or metadata).

## Core Features

- **Multiple Encoding Formats**: Supports both `GOB` (Go's native binary encoding) and `MessagePack` (a fast, compact binary format) out of the box.
- **Type-Safe Generics**: Uses Go generics to ensure that the type of data you encode is the same type you get back when you decode, preventing runtime type assertion errors.
- **Global Configuration**: Allows you to set a default encoding method for the entire application once during initialization.
- **Base64 Encoding**: Automatically encodes the binary output of GOB or MessagePack into a URL-safe Base64 string, making it easy to use in text-based protocols like HTTP headers or JSON fields.
- **gRPC Error Integration**: Provides a `ToStatus` function to convert codec errors into detailed gRPC status errors, improving client-side error handling.

## How to Use

### 1. Set the Default Codec (Optional)

In your application's `main` or `init` function, you can set the desired encoding method. This step is optional; if you don't set it, the default is `GOB`.

This can only be done once.

```go
package main

import "github.com/94peter/vulpes/codec"

func main() {
    // Set MessagePack as the default codec for the application.
    // This must be done before any encoding or decoding operations.
    codec.WithCodecMethod(codec.MSGPACK)

    // ... rest of your application logic
}
```

### 2. Encode and Decode Data

Use the `Encode` and `Decode` functions to serialize and deserialize your data.

```go
package main

import (
	"fmt"
	"log"

	"github.com/94peter/vulpes/codec"
)

// Define the data structure you want to work with.
type SessionData struct {
	UserID   string
	Username string
	Roles    []string
}

func main() {
	// Set the desired codec (optional, defaults to GOB).
	codec.WithCodecMethod(codec.MSGPACK)

	// 1. Create an instance of your data structure.
	mySession := SessionData{
		UserID:   "user-12345",
		Username: "alice",
		Roles:    []string{"admin", "editor"},
	}

	// 2. Encode the data into a string.
	encodedString, err := codec.Encode(mySession)
	if err != nil {
		log.Fatalf("Failed to encode session: %v", err)
	}

	fmt.Printf("Encoded Session: %s\n", encodedString)
	// This string is now safe to be stored in a cookie, a database, or sent in an HTTP header.

	// 3. Decode the string back into the original data structure.
	// You must specify the target type using generics.
	decodedSession, err := codec.Decode[SessionData](encodedString)
	if err != nil {
		log.Fatalf("Failed to decode session: %v", err)
	}

	fmt.Printf("Decoded UserID: %s\n", decodedSession.UserID)
	fmt.Printf("Decoded Username: %s\n", decodedSession.Username)

	// Verify that the decoded data matches the original.
	if mySession.UserID != decodedSession.UserID {
		log.Fatal("Data mismatch after decoding!")
	}
}
```

## API Reference

- `WithCodecMethod(method CodecMethod)`: Sets the global default encoding method (`codec.GOB` or `codec.MSGPACK`). Can only be called once.
- `Encode(v any) (string, error)`: Encodes any Go type into a Base64 string using the default codec.
- `Decode[T any](s string) (T, error)`: Decodes a Base64 string back into a specific Go type `T`.
- `ToStatus(err wrapperErr.ErrorWithMessage) *status.Status`: Converts a package-specific error into a gRPC status, useful for API error responses.

```
