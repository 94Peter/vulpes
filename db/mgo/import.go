package mgo

import (
	"context"
	"fmt"
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
	collection := m.getCollection(collectionName)
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	var rawData []bson.M
	if err := bson.UnmarshalExtJSON(data, false, &rawData); err != nil {
		return fmt.Errorf("BSON UnmarshalExtJSON failed: %w", err)
	}
	if len(rawData) == 0 {
		return nil
	}
	_, err = collection.InsertMany(ctx, rawData)
	return err
}
