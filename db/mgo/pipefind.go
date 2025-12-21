package mgo

import (
	"context"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.opentelemetry.io/otel/attribute"
)

type MgoAggregate interface {
	GetPipeline(q bson.M) mongo.Pipeline
	Index
}

func PipeFind[T MgoAggregate](ctx context.Context, aggr T, filter bson.M) ([]T, error) {
	if dataStore == nil {
		return nil, ErrNotConnected
	}
	pipeline := aggr.GetPipeline(filter)
	data, _ := json.Marshal(pipeline)
	collectionName := aggr.C()
	_, span := dataStore.startTraceSpan(ctx,
		fmt.Sprintf("mongo.pipeFind.%s", collectionName),
		attribute.String("db.collection", collectionName),
		attribute.String("db.operation", "pipeFind"),
		attribute.String("db.statement", string(data)),
	)
	defer span.End()
	sortCursor, err := dataStore.PipeFind(ctx, collectionName, pipeline)
	if err != nil {
		return nil, spanErrorHandler(fmt.Errorf("%w: %w", ErrReadFailed, err), span)
	}
	var slice []T
	err = sortCursor.All(ctx, &slice)
	if err != nil {
		return nil, spanErrorHandler(fmt.Errorf("%w: %w", ErrReadFailed, err), span)
	}
	return slice, spanErrorHandler(nil, span)
}

func PipeFindOne[T MgoAggregate](ctx context.Context, aggr T, filter bson.M) error {
	if dataStore == nil {
		return ErrNotConnected
	}
	pipeline := aggr.GetPipeline(filter)
	data, _ := json.Marshal(pipeline)
	collectionName := aggr.C()
	_, span := dataStore.startTraceSpan(ctx,
		fmt.Sprintf("mongo.pipeFind.%s", collectionName),
		attribute.String("db.collection", collectionName),
		attribute.String("db.operation", "pipeFind"),
		attribute.String("db.statement", string(data)),
	)
	defer span.End()
	err := dataStore.PipeFindOne(ctx, collectionName, pipeline).Decode(&aggr)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrReadFailed, err)
	}
	return nil
}

func (m *mongoStore) PipeFind(ctx context.Context, collection string, pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	c := m.getCollection(collection)
	sortCursor, err := c.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadFailed, err)
	}
	return sortCursor, nil
}

func (m *mongoStore) PipeFindOne(ctx context.Context, collection string, pipeline mongo.Pipeline) *mongo.SingleResult {
	c := m.getCollection(collection)
	sortCursor, err := c.Aggregate(ctx, pipeline)
	if err != nil {
		return mongo.NewSingleResultFromDocument(bson.D{}, err, nil)
	}
	if !sortCursor.Next(ctx) {
		return mongo.NewSingleResultFromDocument(bson.D{}, mongo.ErrNoDocuments, nil)
	}
	return mongo.NewSingleResultFromDocument(sortCursor.Current, nil, nil)
}
