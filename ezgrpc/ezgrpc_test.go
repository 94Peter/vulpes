package ezgrpc

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/94peter/vulpes/ezgrpc/client"
	"github.com/stretchr/testify/assert"
)

// mockClient is a mock implementation of the client.Client interface for testing.
type mockClient struct {
	resp []byte
	err  error
}

// Invoke simulates the behavior of the real client's Invoke method.
func (m *mockClient) Invoke(ctx context.Context, address, serviceName, methodName string, req []byte) ([]byte, error) {
	return m.resp, m.err
}

func (m *mockClient) Close() error {
	return nil
}

func (m *mockClient) GetServiceInvoker(ctx context.Context, address, serviceName string) (client.ServiceInvoker, error) {
	return nil, nil
}

// func TestRealInvoke(t *testing.T) {
// 	ctx := context.Background()
// 	// req := map[string]any{
// 	// 	"id":      "c2c8ea6c-e453-4805-455f-ad2079e02800",
// 	// 	"variant": "public",
// 	// }
// 	type imageRequest struct {
// 		ID      string `json:"id"`
// 		Variant string `json:"variant"`
// 	}
// 	type imageResponse struct {
// 		URI string `json:"uri"`
// 	}
// 	resp, err := Invoke[*imageRequest, imageResponse](ctx, "localhost:8081", "mediaService.ImageService", "GetImageURI", &imageRequest{
// 		ID:      "c2c8ea6c-e453-4805-455f-ad2079e02800",
// 		Variant: "public",
// 	})
// 	fmt.Println(resp, err)
// 	assert.False(t, true)
// }

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
		resp, err := Invoke[map[string]any, map[string]any](ctx, "localhost:8081", "mediaService.ImageService", "SyncImageCount", req)

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, resp)
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
		_, err := Invoke[map[string]any, map[string]any](ctx, "localhost:8081", "mediaService.ImageService", "SyncImageCount", req)

		// Assertions
		assert.Error(t, err)
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
		_, err := Invoke[map[string]any, map[string]any](ctx, "localhost:8081", "mediaService.ImageService", "SyncImageCount", req)

		// Assertions
		assert.Error(t, err)
		assert.IsType(t, &json.SyntaxError{}, err)
	})
}
