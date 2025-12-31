package ezgrpc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

var (
	RedirectResponseOption   = runtime.WithForwardResponseOption(redirectResponseOptionHandler)
	RedirectResponseModifier = runtime.WithForwardResponseRewriter(redirectResponseModifierHandler)
)

const redirectHeader = "Location"

func redirectResponseOptionHandler(ctx context.Context, w http.ResponseWriter, _ proto.Message) error {
	// gRPC-Gateway 會將 gRPC header metadata 的 "location" 鍵轉換為 HTTP 響應的 "Grpc-Metadata-Location"
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return nil
	}

	if location := md.HeaderMD.Get(redirectHeader); len(location) > 0 {
		// 如果找到 "Grpc-Metadata-Location"，則進行 301 重定向
		w.WriteHeader(http.StatusMovedPermanently) // 301
	}
	return nil
}

func redirectResponseModifierHandler(ctx context.Context, resp proto.Message) (any, error) {
	// gRPC-Gateway 會將 gRPC header metadata 的 "location" 鍵轉換為 HTTP 響應的 "Grpc-Metadata-Location"
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return resp, nil
	}

	if location := md.HeaderMD.Get(redirectHeader); len(location) > 0 {
		// var buf bytes.Buffer
		return map[string]any{}, nil
	}
	return resp, nil
}

func SetRedirectUrl(ctx context.Context, url string) error {
	md := metadata.Pairs(redirectHeader, url)
	if err := grpc.SetHeader(ctx, md); err != nil {
		return fmt.Errorf("failed to send header: %w", err)
	}
	return nil
}
