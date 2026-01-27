package mgo

import (
	"context"
	"fmt"
)

func CountDocument(ctx context.Context, collectionName string, filter any) (int64, error) {
	if dataStore == nil {
		return 0, ErrNotConnected
	}
	_, span := dataStore.startTraceSpan(ctx, collectionName, "count_document", filter)
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
