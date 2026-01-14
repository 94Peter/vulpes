package mgo

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
)

func CountDocument(ctx context.Context, collectionName string, filter any) (int64, error) {
	if dataStore == nil {
		return 0, ErrNotConnected
	}
	data, _ := json.Marshal(filter)
	_, span := dataStore.startTraceSpan(ctx,
		"mongo.countDocument."+collectionName,
		attribute.String("db.collection", collectionName),
		attribute.String("db.operation", "count_document"),
		attribute.String("db.statement", string(data)),
	)
	defer span.End()
	result, err := dataStore.CountDocument(ctx, collectionName, filter)
	if err != nil {
		return 0, spanErrorHandler(fmt.Errorf("%w: %w", ErrReadFailed, err), span)
	}
	return result, spanErrorHandler(nil, span)
}

func (m *mongoStore) CountDocument(ctx context.Context, collectionName string, filter any) (int64, error) {
	collection := m.getCollection(collectionName)
	result, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrReadFailed, err)
	}
	return result, nil
}
