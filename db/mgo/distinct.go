package mgo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Distinct[T any](
	ctx context.Context, collectionName string, field string, filter any,
	opts ...options.Lister[options.DistinctOptions],
) ([]T, error) {
	if dataStore == nil {
		return nil, ErrNotConnected
	}
	_, span := dataStore.startTraceSpan(ctx, collectionName, "distinct", filter)
	defer span.End()
	values, err := dataStore.Distinct(ctx, collectionName, field, filter, opts...)
	if err != nil {
		return nil, spanErrorHandler(fmt.Errorf("%w: %w", ErrReadFailed, err), span)
	}
	var ret []T
	for _, v := range values {
		var t T
		err := v.Unmarshal(&t)
		if err != nil {
			return nil, spanErrorHandler(fmt.Errorf("%w: %w", ErrReadFailed, err), span)
		}
		ret = append(ret, t)
	}
	return ret, spanErrorHandler(nil, span)
}

func (m *mongoStore) Distinct(
	ctx context.Context, collectionName string, field string, filter any,
	opts ...options.Lister[options.DistinctOptions],
) ([]bson.RawValue, error) {
	collection := m.getCollection(collectionName)
	result := collection.Distinct(ctx, field, filter, opts...)
	rows, err := result.Raw()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadFailed, err)
	}
	return rows.Values()
}
