package ezgrpc

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/94peter/vulpes/ezgrpc/client"
)

// mockClient is a mock implementation of the client.Client interface for testing.
type mockClient struct {
	resp []byte
	err  error
}

// Invoke simulates the behavior of the real client's Invoke method.
func (m *mockClient) Invoke(_ context.Context, _, _, _ string, _ []byte) ([]byte, error) {
	return m.resp, m.err
}

func (*mockClient) Close() error {
	return nil
}

func (*mockClient) GetServiceInvoker(_ context.Context, _, _ string) (client.ServiceInvoker, error) {
	return nil, nil
}

func TestInvoke(t *testing.T) {
	originalClient := grpcClt
	defer func() {
		grpcClt = originalClient
	}()

	t.Run("Successful invoke", func(t *testing.T) {
		// Setup mock
		mockRespData := map[string]any{"message": "success"}
		mockRespBytes, _ := json.Marshal(mockRespData)
		grpcClt = &mockClient{
			resp: mockRespBytes,
			err:  nil,
		}

		// Call the function
		ctx := context.Background()
		req := map[string]any{"data": "some-data"}
		resp, err := Invoke[map[string]any, map[string]any](
			ctx, "localhost:8081", "mediaService.ImageService", "SyncImageCount", req)

		// Assertions
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "success", resp["message"])
	})

	t.Run("Invoke returns an error", func(t *testing.T) {
		// Setup mock
		grpcClt = &mockClient{
			resp: nil,
			err:  errors.New("grpc error"),
		}

		// Call the function
		ctx := context.Background()
		req := map[string]any{"data": "some-data"}
		_, err := Invoke[map[string]any, map[string]any](
			ctx, "localhost:8081", "mediaService.ImageService", "SyncImageCount", req)

		// Assertions
		require.Error(t, err)
		assert.Equal(t, "grpc error", err.Error())
	})

	t.Run("Response unmarshal error", func(t *testing.T) {
		// Setup mock with invalid JSON response
		grpcClt = &mockClient{
			resp: []byte("invalid-json"),
			err:  nil,
		}

		// Call the function
		ctx := context.Background()
		req := map[string]any{"data": "some-data"}
		_, err := Invoke[map[string]any, map[string]any](
			ctx, "localhost:8081", "mediaService.ImageService", "SyncImageCount", req)

		// Assertions
		require.Error(t, err)
		assert.ErrorIs(t, err, &json.SyntaxError{})
	})
}
