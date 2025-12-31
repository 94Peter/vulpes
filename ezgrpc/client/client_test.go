package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// Using helloworld as a well-defined, simple proto for the test server.
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

// mockGreeterServer implements the helloworld.GreeterServer interface for testing.
type mockGreeterServer struct {
	pb.UnimplementedGreeterServer
}

// SayHello is the implementation of the SayHello RPC.
func (*mockGreeterServer) SayHello(_ context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	if in.Name == "error" {
		return nil, fmt.Errorf("mock error")
	}
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

// startTestServer starts a gRPC server with the mockGreeterServer and reflection enabled.
// It returns the address of the server and a function to stop it.
func startTestServer(t *testing.T) (string, func()) {
	lis, err := net.Listen("tcp", "localhost:7080")
	require.NoError(t, err, "failed to listen on a random port")

	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &mockGreeterServer{})
	reflection.Register(s)

	go func() {
		if err := s.Serve(lis); err != nil {
			// We expect an error when the listener is closed, so we don't log it as fatal.
			log.Printf("gRPC server exited with error: %v", err)
		}
	}()

	stop := func() {
		s.Stop()
		lis.Close()
	}

	return lis.Addr().String(), stop
}

func TestClient_Invoke_Integration(t *testing.T) {
	addr, stop := startTestServer(t)
	defer stop()
	// We create a new client for each subtest to ensure isolation.
	t.Run("Successful Invoke", func(t *testing.T) {
		grpcClt := NewClient()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		reqBody := `{"name": "vulpes"}`
		resp, err := grpcClt.Invoke(ctx, addr, "helloworld.Greeter", "SayHello", []byte(reqBody))

		require.NoError(t, err)
		require.NotNil(t, resp)

		var result map[string]string
		err = json.Unmarshal(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, "Hello vulpes", result["message"])
	})

	t.Run("Service Info and Connection is Cached", func(t *testing.T) {
		grpcClt := NewClient()
		c, ok := grpcClt.(*client)
		require.True(t, ok)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// First call to populate the caches
		reqBody := `{"name": "cache_test"}`
		_, err := grpcClt.Invoke(ctx, addr, "helloworld.Greeter", "SayHello", []byte(reqBody))
		require.NoError(t, err)

		// Check caches
		c.mu.RLock()
		assert.Contains(t, c.conns, addr, "Connection should be cached")
		assert.Contains(t, c.services, addr+"/helloworld.Greeter", "Service info should be cached")
		firstConn := c.conns[addr]
		c.mu.RUnlock()

		// Second call should use the cached items
		_, err = grpcClt.Invoke(ctx, addr, "helloworld.Greeter", "SayHello", []byte(reqBody))
		require.NoError(t, err)

		c.mu.RLock()
		secondConn := c.conns[addr]
		c.mu.RUnlock()

		// Verify it's the exact same connection object
		assert.Same(t, firstConn, secondConn, "Should reuse the same connection object")
	})

	t.Run("Service Not Found", func(t *testing.T) {
		grpcClt := NewClient()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		const reqBody = `{"name": "test"}`
		_, err := grpcClt.Invoke(ctx, addr, "nonexistent.Service", "SayHello", []byte(reqBody))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrServiceNotFound, "should return ErrServiceNotFound")
	})

	t.Run("Method Not Found", func(t *testing.T) {
		grpcClt := NewClient()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		const reqBody = `{"name": "test"}`
		_, err := grpcClt.Invoke(ctx, addr, "helloworld.Greeter", "NonExistentMethod", []byte(reqBody))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMethodNotFound)
	})

	t.Run("Invalid JSON Request", func(t *testing.T) {
		grpcClt := NewClient()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		reqBody := `{"name": "vulpes"` // Invalid JSON
		_, err := grpcClt.Invoke(ctx, addr, "helloworld.Greeter", "SayHello", []byte(reqBody))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidRequest)
	})

	t.Run("Connection Failure", func(t *testing.T) {
		grpcClt := NewClient(WithTimeout(2 * time.Second))
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Use an address that is not listening
		invalidAddr := "localhost:1"
		reqBody := `{"name": "test"}`
		_, err := grpcClt.Invoke(ctx, invalidAddr, "helloworld.Greeter", "SayHello", []byte(reqBody))

		require.Error(t, err)

		assert.ErrorIs(t, err, ErrFetchServerInfoFailed)
	})
}
