// Package ezgrpc provides a simplified setup for gRPC services with a grpc-gateway.
// It includes utilities for handling cookies, sessions, and standard interceptors.
package ezgrpc

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

// CookieForwarder is a grpc-gateway option that forwards Set-Cookie headers from gRPC metadata to the HTTP response.
// This allows gRPC handlers to set cookies on the client's browser.
var CookieForwarder = runtime.WithForwardResponseOption(setCookieForwarder)

const (
	// setCookieKey is the metadata key used to pass cookie values from the gRPC service to the gateway.
	setCookieKey = "set-cookie-header"
	// deleteCookieKey is the metadata key used to signal that a cookie should be deleted.
	deleteCookieKey = "delete-cookie"
	// valueTrue is a constant for the string "true" to avoid magic strings.
	valueTrue = "true"
)

// SetCookie sends a cookie to the client by embedding it in the gRPC header metadata.
// The grpc-gateway, configured with CookieForwarder, will translate this into a standard HTTP Set-Cookie header.
//
// ctx: The context of the gRPC call.
// key: The name of the cookie.
// value: The value of the cookie.
// path: The path for which the cookie is valid.
// maxAge: The maximum age of the cookie in seconds.
func SetCookie(ctx context.Context, key, value string, path string, maxAge int) error {
	cookieValue := http.Cookie{
		Name:     key,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	// 將 cookie 資訊作為 gRPC metadata (trailers) 發送
	// grpc-gateway 將會接收到這個 metadata 並設定 HTTP Set-Cookie header
	return grpc.SetHeader(ctx, metadata.Pairs(setCookieKey, cookieValue.String()))
}

// setCookieForwarder is the response forwarder function for grpc-gateway.
// It inspects the gRPC metadata for "set-cookie-header" and "delete-cookie" keys
// and modifies the HTTP response writer to add the appropriate "Set-Cookie" headers.
func setCookieForwarder(ctx context.Context, w http.ResponseWriter, _ proto.Message) error {
	// 從 context 中獲取 gRPC 伺服器 metadata
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return nil
	}

	// 檢查是否有自訂的 "set-cookie-header"
	if cookieValues := md.HeaderMD.Get(setCookieKey); len(cookieValues) > 0 {
		// 如果有多個 cookie，遍歷所有 "set-cookie-header" 的值
		for _, cookieStr := range cookieValues {
			// 直接將 cookie 字串作為 "Set-Cookie" header 傳遞
			w.Header().Add("Set-Cookie", cookieStr)
		}

		// 刪除 metadata header，這樣它就不會以 Grpc-Metadata-X-Set-Cookie 的形式出現在 HTTP header 中
		delete(md.HeaderMD, setCookieKey)
	}

	// 檢查是否有刪除 cookie 的標記
	if deleteCookieFlag := md.HeaderMD.Get(deleteCookieKey); len(deleteCookieFlag) > 0 {
		if strings.ToLower(deleteCookieFlag[0]) == valueTrue {
			// 設定 Max-Age 為 -1 或 Expires 為過去的時間來刪除 cookie
			past := time.Now().Add(-time.Hour).UTC().Format(time.RFC1123)
			deleteCookieStr := fmt.Sprintf("session_token=; Path=/; Expires=%s; Max-Age=0; HttpOnly; SameSite=Lax", past)
			w.Header().Add("Set-Cookie", deleteCookieStr)
			delete(md.HeaderMD, deleteCookieKey)
		}
	}

	return nil
}
