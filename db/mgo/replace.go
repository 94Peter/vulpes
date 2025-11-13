package mgo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func ReplaceOne[T DocInter](ctx context.Context, doc T, filter any, opts ...options.Lister[options.ReplaceOptions]) (int64, error) {
	if dataStore == nil {
		return 0, ErrNotConnected
	}
	result, err := dataStore.ReplaceOne(ctx, doc.C(), filter, doc, opts...)
	if err != nil {
		return 0, err
	}
	doc.SetId(result.UpsertedID)
	return result.UpsertedCount, nil
}

func (m *mongoStore) ReplaceOne(ctx context.Context, collection string, filter any, replacement any, opts ...options.Lister[options.ReplaceOptions]) (*mongo.UpdateResult, error) {
	return m.getCollection(collection).ReplaceOne(ctx, filter, replacement, opts...)
}
