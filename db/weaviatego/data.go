package weaviatego

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Data interface {
	ClassName() string
	ID() uuid.UUID
}

func (sdk *weaviateSdk) CreateData(ctx context.Context, data Data) error {
	if sdk.clt == nil {
		return fmt.Errorf("weaviate client is not initialized")
	}
	_, err := sdk.clt.Data().Creator().
		WithClassName(data.ClassName()).
		WithProperties(data).
		WithID(data.ID().String()).
		Do(ctx)
	return err
}

// 核心邏輯：新增或更新資料
func (sdk *weaviateSdk) CreateOrUpdateData(ctx context.Context, data Data) error {
	if sdk.clt == nil {
		return fmt.Errorf("weaviate client is not initialized")
	}

	dataID := data.ID().String()

	// 1. 嘗試使用 Creator() 進行新增操作
	// 如果 ID 已經存在，Weaviate 會返回一個錯誤
	_, err := sdk.clt.Data().Creator().
		WithClassName(data.ClassName()).
		WithProperties(data).
		WithID(dataID).
		Do(ctx)

	// 2. 判斷錯誤類型
	if err == nil {
		// 如果沒有錯誤，表示新增成功
		return nil
	}

	// 3. 如果新增失敗，檢查是否為「ID 衝突」錯誤
	// 如果錯誤訊息包含 "already exists" 或相關字眼，我們視為 ID 衝突
	if strings.Contains(err.Error(), "already exists") {
		// 4. 切換到 Updater() 進行部分更新（PATCH）
		// Updater 只會更新你提供的屬性。
		updateErr := sdk.clt.Data().Updater().
			WithClassName(data.ClassName()).
			WithID(dataID).
			WithProperties(data). // 注意：這裡會執行 PATCH
			Do(ctx)

		if updateErr != nil {
			return fmt.Errorf("failed to update data with ID %s: %w", dataID, updateErr)
		}
		return nil
	}
	return fmt.Errorf("failed to create data with ID %s: %w", dataID, err)
}
