// Package ezgrpc provides a simplified setup for gRPC services with a grpc-gateway.
// It includes utilities for handling cookies, sessions, and standard interceptors.
package ezgrpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/94peter/vulpes/constant"
	"github.com/94peter/vulpes/ezgrpc/interceptor"
	"github.com/94peter/vulpes/log"

	"github.com/gorilla/mux"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)

// Constants for user-related metadata keys.
const (
	keyUserID       = "user-id"
	keyUserAccount  = "user-account"
	keyUserEmail    = "user-email"
	keyUserName     = "user-name"
	keyUserLanguage = "user-language"
	keyMerchantID   = "merchant-id"
)

var (
	// grpcService is a gRPC server with a chain of interceptors for common concerns like logging, metrics, and recovery.
	grpcService = interceptor.NewGrpcServerWithInterceptors()

	// opts provides default dialing options for the gRPC client, using insecure credentials for simplicity.
	opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// headerTransMap defines the mapping from incoming HTTP headers to gRPC metadata keys.
	headerTransMap = map[string]string{
		"x-user-id":       keyUserID,
		"x-user-account":  keyUserAccount,
		"x-user-email":    keyUserEmail,
		"x-user-name":     keyUserName,
		"x-user-language": keyUserLanguage,
		"x-merchant-id":   keyMerchantID,
	}

	// DefaultHeaderMatcher is a grpc-gateway option that maps incoming HTTP headers to gRPC metadata.
	// This allows user information from headers to be passed to the gRPC service.
	DefaultHeaderMatcher = runtime.WithIncomingHeaderMatcher(func(k string) (string, bool) {
		// Map the HTTP header "user" to the gRPC metadata "user"
		if v, ok := headerTransMap[strings.ToLower(k)]; ok {
			return v, true
		}
		return runtime.DefaultHeaderMatcher(k)
	})

	// DefaultServeMuxOpts provides default options for the grpc-gateway's ServeMux.
	DefaultServeMuxOpts = []runtime.ServeMuxOption{
		DefaultHeaderMatcher,
	}

	// endpointHandlers stores a list of functions that register gRPC service handlers.
	endpointHandlers []RegisterHandlerFromEndpointFunc

	// router is a Gorilla Mux router for handling HTTP requests.
	router = mux.NewRouter()
)

// user represents the user information extracted from gRPC metadata.
type user struct {
	ID       string
	Account  string
	Email    string
	Name     string
	Language string
	Merchant string
}

// GetUser extracts user information from the incoming gRPC context.
// It returns a user struct if the required metadata is present.
func GetUser(ctx context.Context) (*user, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get metadata from context")
	}
	if len(md.Get(keyUserID)) == 0 {
		return nil, nil
	}
	log.Debug("get user: ",
		log.String(keyUserID, getMetadataString(md, keyUserID)),
		log.String(keyUserAccount, getMetadataString(md, keyUserAccount)),
		log.String(keyUserEmail, getMetadataString(md, keyUserEmail)),
		log.String(keyUserName, getMetadataString(md, keyUserName)),
		log.String(keyUserLanguage, getMetadataString(md, keyUserLanguage)),
		log.String(keyMerchantID, getMetadataString(md, keyMerchantID)),
	)
	return &user{
		ID:       getMetadataString(md, keyUserID),
		Account:  getMetadataString(md, keyUserAccount),
		Email:    getMetadataString(md, keyUserEmail),
		Name:     getMetadataString(md, keyUserName),
		Language: getMetadataString(md, keyUserLanguage),
		Merchant: getMetadataString(md, keyMerchantID),
	}, nil
}

// getMetadataString safely retrieves a string value from metadata.
func getMetadataString(md metadata.MD, key string) string {
	if len(md.Get(key)) == 0 {
		return ""
	}
	return md.Get(key)[0]
}

// SetServeMuxOpts allows overriding the default ServeMux options.
func SetServeMuxOpts(opts ...runtime.ServeMuxOption) {
	DefaultServeMuxOpts = opts
}

// RegisterHandlerFromEndpointFunc is a function type for registering gRPC endpoint handlers.
type RegisterHandlerFromEndpointFunc func(
	ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption,
) (err error)

// RegisterHandlerFromEndpoint adds a handler registration function to the list of handlers to be executed.
func RegisterHandlerFromEndpoint(f RegisterHandlerFromEndpointFunc) {
	endpointHandlers = append(endpointHandlers, f)
}

// InjectGrpcService allows gRPC services to be registered with the central gRPC server.
func InjectGrpcService(f func(grpc.ServiceRegistrar)) {
	f(grpcService)
}

// RunGrpcGateway starts the gRPC gateway and HTTP server.
// It listens on the specified port and serves both gRPC and HTTP traffic.
func RunGrpcGateway(ctx context.Context, port int) error {
	gwmux := runtime.NewServeMux(DefaultServeMuxOpts...)

	portStr := fmt.Sprintf(":%d", port)
	for _, handler := range endpointHandlers {
		if err := handler(ctx, gwmux, portStr, opts); err != nil {
			return fmt.Errorf("failed to register handler: %w", err)
		}
	}
	lis, err := getNetListen(ctx, port)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpc_prometheus.Register(grpcService)
	reflection.Register(grpcService)
	router.Path("/metrics").Handler(promhttp.Handler())
	router.PathPrefix("/").Handler(formToJSONMiddleware(gwmux))

	gwServer := &http.Server{
		Handler:           handlerFunc(grpcService, router),
		ReadHeaderTimeout: constant.DefaultReadHeaderTimeout,
		ReadTimeout:       constant.DefaultReadTimeout,
		WriteTimeout:      constant.DefaultWriteTimeout,
		IdleTimeout:       constant.DefaultIdleTimeout,
	}
	log.Info("Serving on " + portStr)
	return gwServer.Serve(lis)
}

// handlerFunc wraps the gRPC server and an HTTP handler, allowing them to be served on the same port.
// It uses h2c to handle HTTP/2 cleartext traffic, routing gRPC requests to the gRPC server
// and other requests to the provided HTTP handler.
func handlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && r.Header.Get("Content-Type") == "application/grpc" {
			grpcServer.ServeHTTP(w, r)
		} else {
			if otherHandler == nil {
				http.NotFound(w, r)
				return
			}
			otherHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}

func Run(ctx context.Context, port int) error {
	return runServe(ctx, port, nil)
}

func RunGrpcWithHttp(ctx context.Context, port int, httpHandler http.Handler) error {
	if httpHandler == nil {
		return Run(ctx, port)
	}
	return runServe(ctx, port, httpHandler)
}

func runServe(ctx context.Context, port int, httpHandler http.Handler) error {
	lis, err := getNetListen(ctx, port)
	if err != nil {
		return err
	}
	reflection.Register(grpcService)
	gwServer := &http.Server{
		Handler:           handlerFunc(grpcService, httpHandler),
		ReadHeaderTimeout: constant.DefaultReadHeaderTimeout,
		ReadTimeout:       constant.DefaultReadTimeout,
		WriteTimeout:      constant.DefaultWriteTimeout,
		IdleTimeout:       constant.DefaultIdleTimeout,
	}
	log.Info(fmt.Sprintf("Serving on :%d", port))
	return gwServer.Serve(lis)
}

func getNetListen(ctx context.Context, port int) (net.Listener, error) {
	lc := net.ListenConfig{
		// 這裡可以設置 KeepAlive 時間或其他選項
		KeepAlive: constant.DefaultKeepAlive,
	}
	portStr := fmt.Sprintf(":%d", port)
	// 傳入 context，讓啟動過程也受 context 控管
	return lc.Listen(ctx, "tcp", portStr)
}
