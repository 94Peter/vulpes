package mgo

import (
	"context"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/otel/attribute"
)

func ReplaceOne[T DocInter](
	ctx context.Context, doc T, filter any,
	opts ...options.Lister[options.ReplaceOptions],
) (int64, error) {
	if dataStore == nil {
		return 0, ErrNotConnected
	}
	data, _ := json.Marshal(filter)
	collectionName := doc.C()
	_, span := dataStore.startTraceSpan(ctx, collectionName, "save", data)
	defer span.End()
	result, err := dataStore.ReplaceOne(ctx, doc.C(), filter, doc, opts...)
	if err != nil {
		return 0, spanErrorHandler(fmt.Errorf("%w: %w", ErrWriteFailed, err), span)
	}
	span.SetAttributes(attribute.Int64("db.affected_number_of_documents", result.MatchedCount))
	return result.UpsertedCount, spanErrorHandler(nil, span)
}

func (m *mongoStore) ReplaceOne(
	ctx context.Context, collection string, filter any, replacement any,
	opts ...options.Lister[options.ReplaceOptions],
) (*mongo.UpdateResult, error) {
	return m.getCollection(collection).ReplaceOne(ctx, filter, replacement, opts...)
}
