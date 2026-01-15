package mgo

import (
	"context"
	"encoding/json"
	"io"

	"go.mongodb.org/mongo-driver/bson"
)

func Import(
	ctx context.Context, collectionName string, reader io.Reader,
) error {
	if dataStore == nil {
		return ErrNotConnected
	}
	return dataStore.Import(ctx, collectionName, reader)
}

func (m *mongoStore) Import(
	ctx context.Context, collectionName string, reader io.Reader,
) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	collection := m.getCollection(collectionName)
	var rawDocs []json.RawMessage
	if err := json.Unmarshal(data, &rawDocs); err != nil {
		return err
	}

	// 3. 轉換為 BSON Documents
	var docs []any
	for _, raw := range rawDocs {
		var doc bson.M
		// 使用 true 開啟 Canonical/Relaxed 模式支援 $date, $oid
		if err := bson.UnmarshalExtJSON(raw, true, &doc); err != nil {
			return err
		}
		docs = append(docs, doc)
	}
	_, err = collection.InsertMany(ctx, docs)
	return err
}
