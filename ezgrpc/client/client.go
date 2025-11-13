package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	rppb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/94peter/vulpes/log"
)

const defaultConnTimeout = 10 * time.Second

type client struct {
	conns    map[string]*grpc.ClientConn
	services map[string]ServiceInvoker
	mu       sync.RWMutex
	timeout  time.Duration
}

type serviceInfo struct {
	conn    *grpc.ClientConn
	Name    string
	Methods map[string]*methodInfo
}

type methodInfo struct {
	Name       string
	InputType  protoreflect.MessageDescriptor
	OutputType protoreflect.MessageDescriptor
	IsStream   bool
}

type Option func(*client)

func WithTimeout(timeout time.Duration) Option {
	return func(c *client) {
		c.timeout = timeout
	}
}

func NewClient(opts ...Option) Client {
	c := &client{
		conns:    make(map[string]*grpc.ClientConn),
		services: make(map[string]ServiceInvoker),
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.timeout == 0 {
		c.timeout = defaultConnTimeout
	}
	return c
}

func (c *client) Close() error {
	var firstErr error
	for addr, conn := range c.conns {
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close connection to %s: %w", addr, err)
		}
	}
	return firstErr
}

func (c *client) Invoke(ctx context.Context, addr, service, method string, jsonbody []byte) ([]byte, error) {
	serviceInvoker, err := c.GetServiceInvoker(ctx, addr, service)
	if err != nil {
		return nil, fmt.Errorf("%w:failed to get gRPC service info for service '%s' at '%s': %w", ErrServiceNotFound, service, addr, err)
	}
	return serviceInvoker.Invoke(ctx, method, jsonbody)
}

func (c *client) GetServiceInvoker(ctx context.Context, address, serviceName string) (ServiceInvoker, error) {
	cacheKey := address + "/" + serviceName

	// Check cache first with a read lock.
	c.mu.RLock()
	info, exists := c.services[cacheKey]
	c.mu.RUnlock()

	if exists {
		return info, nil
	}

	// If not in cache, acquire a write lock to fetch and store it.
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check if another goroutine populated the cache while we were waiting for the lock.
	info, exists = c.services[cacheKey]
	if exists {
		return info, nil
	}

	conn, err := c.getOrCreateConn(address)
	if err != nil {
		return nil, err
	}

	fetchedInfo, err := c.fetchServiceInfoFromServer(ctx, conn, serviceName)
	if err != nil {
		// Don't close the connection here, as it might be shared.
		// Connections are only closed by the Client's Close() method.
		return nil, fmt.Errorf("%w: %w", ErrFetchServerInfoFailed, err)
	}

	fetchedInfo.conn = conn
	invoker := newServiceInvoker(fetchedInfo)
	c.services[cacheKey] = invoker // Cache the newly fetched info.

	return invoker, nil
}

func (c *client) getOrCreateConn(address string) (*grpc.ClientConn, error) {
	if conn, ok := c.conns[address]; ok {
		return conn, nil
	}
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  500 * time.Millisecond,
				Multiplier: 1.1,
				Jitter:     0.1,
				MaxDelay:   3 * time.Second,
			},
			MinConnectTimeout: c.timeout,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to connect to %s: %w", ErrConnectionFailed, address, err)
	}

	c.conns[address] = conn // Add the new connection to the pool.
	return conn, nil
}

func (c *client) fetchServiceInfoFromServer(ctx context.Context, conn *grpc.ClientConn, serviceName string) (*serviceInfo, error) {
	reflectClient := rppb.NewServerReflectionClient(conn)
	stream, err := reflectClient.ServerReflectionInfo(ctx, grpc.WaitForReady(true))
	if err != nil {
		return nil, fmt.Errorf("failed to create reflection stream: %v", err)
	}

	if err := stream.Send(&rppb.ServerReflectionRequest{
		MessageRequest: &rppb.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: serviceName},
	}); err != nil {
		return nil, fmt.Errorf("failed to send file request: %v", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive response: %v", err)
	}

	fileResp := resp.GetFileDescriptorResponse()
	if fileResp == nil {
		if errResp := resp.GetErrorResponse(); errResp != nil {
			return nil, fmt.Errorf("server reflection error: %s (code: %d)", errResp.ErrorMessage, errResp.ErrorCode)
		}
		return nil, fmt.Errorf("unexpected response type, not FileDescriptorResponse or ErrorResponse")
	}

	fileDescriptorProtos := make([]*descriptorpb.FileDescriptorProto, 0, len(fileResp.FileDescriptorProto))
	for _, b := range fileResp.FileDescriptorProto {
		fdp := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(b, fdp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal file descriptor proto: %v", err)
		}
		fileDescriptorProtos = append(fileDescriptorProtos, fdp)
	}

	return parseServiceDescriptor(fileDescriptorProtos, serviceName)
}

// parseServiceDescriptor parses file descriptors to extract metadata for a specific service.
func parseServiceDescriptor(fileDescriptorProtos []*descriptorpb.FileDescriptorProto, serviceName string) (*serviceInfo, error) {
	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{File: fileDescriptorProtos})
	if err != nil {
		return nil, fmt.Errorf("failed to create file descriptor from response: %v", err)
	}

	var targetService protoreflect.ServiceDescriptor
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Services().Len(); i++ {
			service := fd.Services().Get(i)
			if string(service.FullName()) == serviceName {
				targetService = service
				return false // Stop iterating once found.
			}
		}
		return true // Continue iterating.
	})

	if targetService == nil {
		return nil, fmt.Errorf("service '%q' not found in the provided file descriptors", serviceName)
	}

	methods := make(map[string]*methodInfo)
	for i := 0; i < targetService.Methods().Len(); i++ {
		method := targetService.Methods().Get(i)
		methods[string(method.Name())] = &methodInfo{
			Name:       string(method.Name()),
			InputType:  method.Input(),
			OutputType: method.Output(),
			IsStream:   method.IsStreamingClient() || method.IsStreamingServer(),
		}
	}

	return &serviceInfo{
		Name:    string(targetService.FullName()),
		Methods: methods,
	}, nil
}

func newServiceInvoker(info *serviceInfo) ServiceInvoker {
	return &serviceInvoker{info: info}
}

type serviceInvoker struct {
	info *serviceInfo
}

func (s *serviceInvoker) Invoke(ctx context.Context, method string, jsonbody []byte) ([]byte, error) {
	info := s.info
	methodInfo, ok := info.Methods[method]
	service := info.Name
	if !ok {
		return nil, fmt.Errorf("%w: method '%s' not found in service '%s'", ErrMethodNotFound, method, service)
	}

	if methodInfo.IsStream {
		return nil, fmt.Errorf("streaming RPCs are not supported (method: '%s')", method)
	}

	requestProto := dynamicpb.NewMessage(methodInfo.InputType)
	if jsonbody != nil {
		if err := protojson.Unmarshal(jsonbody, requestProto); err != nil {
			return nil, fmt.Errorf("%w: failed to unmarshal JSON into request: %w", ErrInvalidRequest, err)
		}
	}
	responseProto := dynamicpb.NewMessage(methodInfo.OutputType)
	fullMethod := fmt.Sprintf("/%s/%s", service, method)

	if err := info.conn.Invoke(ctx, fullMethod, requestProto, responseProto); err != nil {
		log.Errorf("Error Invoke service  '%s': gRPC call failed: %v", fullMethod, err)
		return nil, err
	}

	return protojson.Marshal(responseProto)
}

func (s *serviceInvoker) IsMethodExists(methodName string) bool {
	_, ok := s.info.Methods[methodName]
	return ok
}
