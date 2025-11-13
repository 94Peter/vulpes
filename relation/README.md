# 關聯 (Relation) 套件

`relation` 套件提供了一個 Ory Keto 的 Go 客戶端，旨在簡化基於關係元組 (Relation Tuples) 的權限控制。它封裝了 Keto 的 gRPC API，提供了一系列易於使用的方法來建立、查詢、檢查和刪除權限關係。

## 功能

-   **初始化連線**: 輕鬆設定與 Keto 讀取和寫入服務的 gRPC 連線。
-   **元組管理**: 使用 `TupleBuilder` 流暢地建立和刪除關係元組。
-   **權限檢查**: 快速驗證某個主體 (Subject) 是否對某個物件 (Object) 具有特定關係 (Relation)。
-   **關係查詢**: 查詢與特定物件或主體相關的所有關係。
-   **高階角色管理**: 提供 `AddUserResourceRole` 輔助函式，簡化為使用者分配資源角色（如 `owner`, `editor`, `viewer`）的流程，並自動處理角色繼承。
-   **錯誤處理**: 將內部錯誤轉換為標準的 gRPC `status.Status`，便於服務端進行統一錯誤處理。

## 安裝

這是一個內部套件，可直接在您的專案中匯入：

```go
import "github.com/94peter/vulpes/relation"
```

## 使用方法

### 1. 初始化

在應用程式啟動時，使用 Keto 服務的地址初始化套件。

```go
import "github.com/94peter/vulpes/relation"

func main() {
    // 初始化 Keto 連線
    relation.Initialize(
        relation.WithWriteAddr("keto.dev.orb.local:4467"), // Keto 寫入 API 地址
        relation.WithReadAddr("keto.dev.orb.local:4466"),  // Keto 讀取 API 地址
    )
    // 確保在應用程式關閉時關閉連線
    defer relation.Close()

    // ... 您的應用程式邏輯
}
```

### 2. 新增使用者角色

使用 `AddUserResourceRole` 可以方便地為使用者賦予某個資源的角色。此函式會自動處理角色繼承：
-   `RoleOwner` 會自動繼承 `RoleEditor` 和 `RoleViewer` 的權限。
-   `RoleEditor` 會自動繼承 `RoleViewer` 的權限。

```go
import (
    "context"
    "log"
    "github.com/94peter/vulpes/relation"
)

func AssignRole() {
    ctx := context.Background()
    userID := "user:peter"
    resourceNamespace := "Image"
    resourceID := "image:travel-photo-01"

    // 將 "user:peter" 設為 "image:travel-photo-01" 的擁有者
    err := relation.AddUserResourceRole(ctx, userID, resourceNamespace, resourceID, relation.RoleOwner)
    if err != nil {
        log.Fatalf("新增角色失敗: %v", err)
    }
    log.Println("角色新增成功！")
}
```

### 3. 檢查權限

使用 `Check` 或 `CheckBySubjectId` 來驗證權限。

```go
import (
    "context"
    "log"
    "github.com/94peter/vulpes/relation"
)

func CheckPermission() {
    ctx := context.Background()
    userID := "user:peter"
    resourceNamespace := "Image"
    resourceID := "image:travel-photo-01"

    // 檢查 "user:peter" 是否有 "viewer" 權限
    // 因為 peter 是 owner，而 owner 繼承了 viewer，所以這裡會回傳 true
    allowed, err := relation.CheckBySubjectId(ctx, resourceNamespace, resourceID, string(relation.RoleViewer), userID)
    if err != nil {
        log.Fatalf("權限檢查失敗: %v", err)
    }

    if allowed {
        log.Println("存取允許！")
    } else {
        log.Println("存取被拒絕！")
    }
}
```

### 4. 查詢關係

您可以查詢與特定物件或主體相關的關係。

#### 查詢誰可以存取某個物件

```go
// 查詢誰是 "image:travel-photo-01" 的 "editor"
resp, err := relation.QuerySubjectByObjectRelation(ctx, "Image", "image:travel-photo-01", "editor")
if err != nil {
    log.Fatalf("查詢失敗: %v", err)
}
// resp.SubjectIds 將包含所有具有 editor 角色的使用者 ID
```

#### 查詢某個使用者可以存取哪些物件

```go
// 查詢 "user:peter" 作為 "viewer" 可以存取哪些 "Image"
resp, err := relation.QueryObjectBySubjectIdRelation(ctx, "Image", "user:peter", "viewer")
if err != nil {
    log.Fatalf("查詢失敗: %v", err)
}
// resp.Objects 將包含所有 "user:peter" 可以查看的圖片
```

### 5. 刪除關係

#### 使用 Tuple Builder 刪除特定關係

```go
tuples := relation.NewTupleBuilder()
tuples.AppendDeleteTupleWithSubjectId("Image", "image:travel-photo-01", "viewer", "user:john")

err := relation.WriteTuple(ctx, tuples)
if err != nil {
    log.Fatalf("刪除關係失敗: %v", err)
}
```

#### 刪除與物件相關的所有關係

```go
err := relation.DeleteObjectId(ctx, "Image", "image:travel-photo-01")
if err != nil {
    log.Fatalf("刪除物件失敗: %v", err)
}
```

## API 概覽

-   `Initialize(opts ...Option)`: 初始化 Keto 連線。
-   `Close()`: 關閉與 Keto 的連線。
-   `NewTupleBuilder()`: 建立一個新的元組產生器。
-   `WriteTuple(ctx, tuples)`: 寫入（新增或刪除）元組事務。
-   `Check(ctx, ...)`: 檢查基於 `SubjectSet` 的權限。
-   `CheckBySubjectId(ctx, ...)`: 檢查基於 `SubjectId` 的權限。
-   `QueryObjectBySubjectIdRelation(ctx, ...)`: 根據 `SubjectId` 查詢物件。
-   `QueryObjectBySubjectSetRelation(ctx, ...)`: 根據 `SubjectSet` 查詢物件。
-   `QuerySubjectByObjectRelation(ctx, ...)`: 根據物件查詢主體。
-   `DeleteObjectId(ctx, ...)`: 刪除與某個物件 ID 相關的所有元組。
-   `AddUserResourceRole(ctx, ...)`: 為使用者新增資源角色並處理繼承。
-   `ToStatus(err)`: 將套件錯誤轉換為 gRPC `status.Status`。
