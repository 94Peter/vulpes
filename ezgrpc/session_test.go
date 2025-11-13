// Package ezgrpc provides a simplified setup for gRPC services with a grpc-gateway.
// It includes utilities for handling cookies, sessions, and standard interceptors.
package ezgrpc

import (
	"context"
	"testing"

	"github.com/94peter/vulpes/codec"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

// sessionData is a test struct for session data.
type sessionData struct {
	User string
	ID   int
}

// TestSetAndGetSessionData tests the process of setting and retrieving session data.
// It simulates the flow from a gRPC client setting data to a server retrieving it.
func TestSetAndGetSessionData(t *testing.T) {
	// 1. SETUP: Initial data and context
	data := sessionData{User: "test", ID: 123}
	ctx := context.Background()

	// 2. EXECUTE: Set the session data into the context
	encoded, err := codec.Encode(data)
	assert.NoError(t, err)
	ctxWithData := metadata.NewOutgoingContext(ctx, metadata.Pairs(setSessionDataKey, encoded))

	// 3. VERIFY: Extract outgoing metadata from the context
	md, ok := metadata.FromOutgoingContext(ctxWithData)
	assert.True(t, ok, "metadata should be set in outgoing context")

	// 4. SETUP: Create an incoming context with the extracted metadata
	// This simulates the gRPC server-side receiving the metadata
	encodedData := md.Get(setSessionDataKey)
	assert.Len(t, encodedData, 1, "encoded data should have one value")

	incomingCtx := metadata.NewIncomingContext(ctx, metadata.Pairs(sessionDataKey, encodedData[0]))

	// 5. EXECUTE & VERIFY: Get the session data from the incoming context
	retrievedData, err := GetSessionData[sessionData](incomingCtx)
	assert.NoError(t, err, "GetSessionData should not return an error")
	assert.Equal(t, data, retrievedData, "retrieved data should match original data")
}

// TestGetSessionData_Errors tests the error cases for GetSessionData.
func TestGetSessionData_Errors(t *testing.T) {
	t.Run("NoMetadataInContext", func(t *testing.T) {
		// Execute: Try to get data from a context without any metadata
		_, err := GetSessionData[sessionData](context.Background())

		// Verify: Check for the specific error
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("NoSessionDataInMetadata", func(t *testing.T) {
		// Setup: Create a context with empty metadata
		ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})

		// Execute: Try to get data
		_, err := GetSessionData[sessionData](ctx)

		// Verify: Check for the specific error
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("CorruptedData", func(t *testing.T) {
		// Setup: Create a context with corrupted (not base64) data
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(sessionDataKey, "corrupted-data"))

		// Execute: Try to get data
		_, err := GetSessionData[sessionData](ctx)

		// Verify: Check that a codec error is returned
		assert.Error(t, err)
		assert.ErrorIs(t, err, codec.ErrBase64DecodeFailed)
	})
}
