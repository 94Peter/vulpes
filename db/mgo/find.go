package mgo

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/otel/codes"
)

func Find[T DocInter](
	ctx context.Context, doc T, filter any, limit uint16,
	opts ...options.Lister[options.FindOptions],
) ([]T, error) {
	if dataStore == nil {
		return nil, ErrNotConnected
	}
	if limit == 0 {
		limit = 100
	}
	collectionName := doc.C()
	_, span := dataStore.startTraceSpan(ctx, collectionName, "find", filter)
	defer span.End()

	finalArgs := make([]options.Lister[options.FindOptions], 0, len(opts)+1)
	finalArgs = append(finalArgs, options.Find().SetLimit(int64(limit)))
	finalArgs = append(finalArgs, opts...)

	result, err := dataStore.Find(ctx, doc.C(), filter, finalArgs...)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			span.SetStatus(codes.Ok, "ok")
			return nil, err
		}
		return nil, spanErrorHandler(fmt.Errorf("%w: %w", ErrReadFailed, err), span)
	}
	ret, err := cursorToSlice[T](ctx, result, result.RemainingBatchLength())
	if err != nil {
		return nil, spanErrorHandler(fmt.Errorf("%w: %w", ErrReadFailed, err), span)
	}
	return ret, spanErrorHandler(nil, span)
}

func FindOne[T DocInter](
	ctx context.Context, doc T, filter any,
	opts ...options.Lister[options.FindOneOptions],
) error {
	if dataStore == nil {
		return ErrNotConnected
	}
	collectionName := doc.C()
	_, span := dataStore.startTraceSpan(ctx, collectionName, "findOne", filter)
	defer span.End()
	err := dataStore.FindOne(ctx, doc.C(), filter, opts...).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			span.SetStatus(codes.Ok, "ok")
			return err
		}
		return spanErrorHandler(fmt.Errorf("%w: %w", ErrReadFailed, err), span)
	}
	return spanErrorHandler(nil, span)
}

func FindById[T DocInter](ctx context.Context, doc T) error {
	if dataStore == nil {
		return ErrNotConnected
	}
	return FindOne(ctx, doc, bson.M{"_id": doc.GetId()})
}

func (m *mongoStore) Find(
	ctx context.Context, collectionName string, filter any,
	opts ...options.Lister[options.FindOptions],
) (*mongo.Cursor, error) {
	collection := m.getCollection(collectionName)
	cursor, err := collection.Find(ctx, filter, opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadFailed, err)
	}
	return cursor, nil
}

func (m *mongoStore) FindOne(
	ctx context.Context, collectionName string, filter any,
	opts ...options.Lister[options.FindOneOptions],
) *mongo.SingleResult {
	collection := m.getCollection(collectionName)
	return collection.FindOne(ctx, filter, opts...)
}

func cursorToSlice[T any](ctx context.Context, cursor *mongo.Cursor, cap int) ([]T, error) {
	ret := make([]T, 0, cap)
	for cursor.Next(ctx) {
		var t T
		// 如果 T 是指標類型 (例如 *ComplexStruct)，需要初始化
		// 這裡利用 any(t) 進行 UnmarshalBSON 斷言，實現高效解碼
		if unmarshaler, ok := any(&t).(bson.Unmarshaler); ok {
			if err := unmarshaler.UnmarshalBSON(cursor.Current); err != nil {
				return nil, err
			}
		} else {
			// 備援方案：若沒實作 UnmarshalBSON 則走預設解碼
			if err := cursor.Decode(&t); err != nil {
				return nil, err
			}
		}
		ret = append(ret, t)
	}
	return ret, nil
}
