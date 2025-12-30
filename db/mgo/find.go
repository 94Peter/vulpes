package mgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func Find[T DocInter](ctx context.Context, doc T, filter any, opts ...options.Lister[options.FindOptions]) ([]T, error) {
	if dataStore == nil {
		return nil, ErrNotConnected
	}
	data, _ := json.Marshal(filter)
	collectionName := doc.C()
	_, span := dataStore.startTraceSpan(ctx,
		fmt.Sprintf("mongo.find.%s", collectionName),
		attribute.String("db.collection", collectionName),
		attribute.String("db.operation", "find"),
		attribute.String("db.statement", string(data)),
	)
	defer span.End()
	result, err := dataStore.Find(ctx, doc.C(), filter, opts...)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			span.SetStatus(codes.Ok, "ok")
			return nil, err
		}
		return nil, spanErrorHandler(fmt.Errorf("%w: %w", ErrReadFailed, err), span)
	}
	var ret []T
	err = result.All(ctx, &ret)
	if err != nil {
		return nil, spanErrorHandler(fmt.Errorf("%w: %w", ErrReadFailed, err), span)
	}
	return ret, spanErrorHandler(nil, span)
}

func FindOne[T DocInter](ctx context.Context, doc T, filter any, opts ...options.Lister[options.FindOneOptions]) error {
	if dataStore == nil {
		return ErrNotConnected
	}
	data, _ := json.Marshal(filter)
	collectionName := doc.C()
	_, span := dataStore.startTraceSpan(ctx,
		fmt.Sprintf("mongo.findOne.%s", collectionName),
		attribute.String("db.collection", collectionName),
		attribute.String("db.operation", "findOne"),
		attribute.String("db.statement", string(data)),
	)
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

func (m *mongoStore) Find(ctx context.Context, collectionName string, filter any, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, error) {
	collection := m.getCollection(collectionName)
	cursor, err := collection.Find(ctx, filter, opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadFailed, err)
	}
	return cursor, nil
}

func (m *mongoStore) FindOne(ctx context.Context, collectionName string, filter any, opts ...options.Lister[options.FindOneOptions]) *mongo.SingleResult {
	collection := m.getCollection(collectionName)
	return collection.FindOne(ctx, filter, opts...)
}
