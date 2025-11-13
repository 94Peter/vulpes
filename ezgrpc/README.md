# ezgrpc

`ezgrpc` 是一個 Go 套件，旨在簡化 gRPC 服務與 RESTful JSON 閘道的設定和部署。它提供了一個預先配置好的伺服器，內建了常見的中介軟體 (interceptors)、Session 管理，並與 [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) 無縫整合。

## 核心功能

- **一鍵啟動 gRPC 伺服器**：內建日誌、Prometheus 指標、Panic 復原、速率限制和請求驗證等必要的攔截器。
- **HTTP/gRPC 流量複用**：在單一 Port 上同時提供 gRPC 和 JSON RESTful API 服務。
- **內建 Session 管理**：輕鬆在 HTTP 閘道和 gRPC 服務之間共享 Session 資料。
- **使用者資訊傳遞**：自動將 HTTP Header 中的使用者資訊 (如 `X-User-ID`) 轉發到 gRPC 的 metadata 中。
- **模組化服務註冊**：鼓勵將服務註冊邏輯放到各個服務自己的 `init()` 函式中，讓主程式更簡潔。

## 安裝

```bash
go get github.com/94peter/vulpes/ezgrpc
```

## 快速入門

以下是一個簡單的範例，展示如何使用 `ezgrpc` 快速建立一個 gRPC 服務。

### 1. 定義您的服務 (`.proto`)

首先，定義您的 gRPC 服務，例如 `greeter.proto`：

```protobuf
syntax = "proto3";

package greeter;

option go_package = "path/to/your/greeter";

import "google/api/annotations.proto";

service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply) {
    option (google.api.http) = {
      post: "/v1/greeter/say_hello",
      body: "*"
    };
  }
}

message HelloRequest {
  string name = 1;
}

message HelloReply {
  string message = 1;
}
```

### 2. 產生 Go 程式碼

使用 `protoc` 來產生 gRPC 和 grpc-gateway 的程式碼。

```bash
protoc -I . --go_out=. --go-grpc_out=. --grpc-gateway_out=. your_service.proto
```

### 3. 實作並註冊您的服務 (`greeter/greeter.go`)

在您的服務套件中，實作服務邏輯，並使用 `init()` 函式自動註冊服務。

```go
package greeter

import (
	"context"
	"log"

	"github.com/94peter/vulpes/ezgrpc"
	"google.golang.org/grpc"
	
	// 引入您產生的 pb 檔案
	pb "path/to/your/greeter"
)

// server 用於實作您的 gRPC 服務。
type server struct {
	pb.UnimplementedGreeterServer
}

// init 函數會在套件被導入時自動執行。
// 我們在這裡註冊 gRPC 服務和 Gateway 處理器。
func init() {
	// 註冊 gRPC 服務實作。
	ezgrpc.InjectGrpcService(func(s *grpc.Server) {
		pb.RegisterGreeterServer(s, &server{})
	})

	// 註冊 gRPC-Gateway 處理器。
	ezgrpc.RegisterHandlerFromEndpoint(pb.RegisterGreeterHandlerFromEndpoint)
}

// SayHello 實作了 Greeter 服務。
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	// 範例：使用 Session 資料
	session, err := ezgrpc.GetSessionData[map[string]string](ctx)
	if err == nil {
		log.Printf("發現 Session 資料: %v", session)
	}

	// 範例：從 Headers 獲取使用者資訊
	user, _ := ezgrpc.GetUser(ctx)
	if user != nil {
		log.Printf("來自使用者的請求: %s (ID: %s)", user.Name, user.ID)
	}

	log.Printf("收到請求: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}
```

### 4. 建立您的主程式 (`main.go`)

主程式現在變得非常簡潔。您只需要初始化 `ezgrpc` 並啟動伺服器。

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/94peter/vulpes/ezgrpc"
	
	// 透過空白導入來觸發 greeter 服務的 init() 函數
	_ "path/to/your/greeter"
)

func main() {
	// 1. 初始化 Session 儲存。
	ezgrpc.InitSessionStore()

	// 2. 在 Port 8080 上執行伺服器。
	// 服務註冊已在各自的套件中透過 init() 自動完成。
	fmt.Println("伺服器正在 Port 8080 上監聽")
	if err := ezgrpc.RunGrpcGateway(context.Background(), 8080); err != nil {
		log.Fatalf("伺服器啟動失敗: %v", err)
	}
}
```

## 詳細功能說明

### Session 管理

You can set and read session data in any gRPC handler. `ezgrpc` automatically handles cookie setting and reading.

```go
// MySessionData is your custom session struct
type MySessionData struct {
    UserID   string
    Username string
}

// Set session in login handler
func (s *server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
    // ... authenticate user ...
    session := MySessionData{UserID: "user-123", Username: "testuser"}
    if err := ezgrpc.SetSessionData(ctx, session); err != nil {
        return nil, status.Error(codes.Internal, "Failed to set session")
    }
    return &pb.LoginResponse{Success: true}, nil
}

// Read session in protected endpoint
func (s *server) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.ProfileResponse, error) {
    session, err := ezgrpc.GetSessionData[MySessionData](ctx)
    if err != nil {
        return nil, status.Error(codes.Unauthenticated, "Not logged in")
    }
    // ... use session.UserID and session.Username ...
    return &pb.ProfileResponse{Id: session.UserID, Name: session.Username}, nil
}
```

### User Information Forwarding

When clients send HTTP headers with specific prefixes to the gateway, `ezgrpc` automatically forwards them to the gRPC context metadata.

Supported Headers include:
- `X-User-ID`
- `X-User-Account`
- `X-User-Email`
- `X-User-Name`
- `X-User-Language`

In your gRPC handler, you can retrieve user information like this:

```go
func (s *server) MyHandler(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    user, err := ezgrpc.GetUser(ctx)
    if err != nil {
        // Handle error
    }
    if user != nil {
        log.Printf("Request from User ID: %s, Language: %s", user.ID, user.Language)
    }
    // ...
}
```

### Default Interceptors (Interceptors)

`ezgrpc` enables a series of interceptors by default, in the following order:

1.  **Recovery**: Catches panics to prevent server crashes.
2.  **Prometheus**: Monitors gRPC request metrics.
3.  **RequestID**: Generates a unique ID for each request.
4.  **Logger**: Logs detailed request information, relying on RequestID.
5.  **RateLimit**: IP-based request rate limiting.
6.  **Validation**: Automatically validates requests conforming to `protoc-gen-validate` rules.

### Dynamic gRPC Client (`ezclient`)

`ezgrpc` includes a dynamic gRPC client that allows you to call any gRPC service without relying on pre-compiled protobuf stubs (`.pb.go` files).

This feature's core is **gRPC Server Reflection**. As long as the target gRPC server has reflection enabled, `ezclient` can automatically discover services, methods, and request/response message structures at runtime.

> **Note**: When you start the server using `ezgrpc.RunGrpcGateway` or `ezgrpc.Run`, the reflection service is automatically enabled.

#### Usage Example

The following example demonstrates how to use the generic function `ezgrpc.Invoke` to call a gRPC service.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/94peter/vulpes/ezgrpc"
)

// Assume we want to call a service with a GetImageURI method
// We don't need to import the service's .pb.go file, just define the request and response structs

// ImageRequest defines the structure of the request
type ImageRequest struct {
	ID      string `json:"id"`
	Variant string `json:"variant"`
}

// ImageResponse defines the expected response structure
type ImageResponse struct {
	URI string `json:"uri"`
}

func main() {
	ctx := context.Background()

	// Prepare request data
	req := &ImageRequest{
		ID:      "c2c8ea6c-e453-4805-455f-ad2079e02800",
		Variant: "public",
	}

	// Call the remote method using ezgrpc.Invoke
	resp, err := ezgrpc.Invoke[ImageRequest, ImageResponse](
		ctx,
		"localhost:8081",                 // gRPC server address
		"mediaService.ImageService",      // Full service name (package.Service)
		"GetImageURI",                    // Method name to call
		req,                              // Request data
	)

	if err != nil {
		log.Fatalf("gRPC call failed: %v", err)
	}

	fmt.Printf("Successfully received response: %+v\n", resp)
	// Expected output: Successfully received response: {URI:http://path/to/your/image.jpg}
}
```

`ezgrpc.Invoke` function accepts the following parameters:
- `ctx`: `context.Context`.
- `addr`: The address of the gRPC server, in the format `host:port`.
- `service`: The full gRPC service name, in the format `package_name.ServiceName`.
- `method`: The name of the method you want to call.
- `req`: Your request data, which can be a struct or a map. `ezclient` will automatically serialize it to JSON.

The response will be automatically deserialized from JSON into the generic type `R` you specified.
