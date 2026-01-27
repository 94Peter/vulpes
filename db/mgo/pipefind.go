package mgo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MgoAggregate interface {
	GetPipeline(q bson.M) mongo.Pipeline
	C() string
}

func PipeFindByPipeline[T any](
	ctx context.Context,
	collectionName string,
	pipeline mongo.Pipeline,
	limit uint16,
) ([]T, error) {
	if dataStore == nil {
		return nil, ErrNotConnected
	}
	_, span := dataStore.startTraceSpan(ctx, collectionName, "pipeFindByPipeline", pipeline)
	defer span.End()

	sortCursor, err := dataStore.PipeFind(ctx, collectionName, pipeline)
	if err != nil {
		return nil, spanErrorHandler(fmt.Errorf("%w: %w", ErrReadFailed, err), span)
	}
	if limit == 0 {
		limit = 100
	}
	slice, err := cursorToSlice[T](ctx, sortCursor, int(limit))
	if err != nil {
		return nil, spanErrorHandler(fmt.Errorf("%w: %w", ErrReadFailed, err), span)
	}
	return slice, spanErrorHandler(nil, span)
}

func PipeFind[T MgoAggregate](
	ctx context.Context, aggr T, filter bson.M, limit uint16,
) ([]T, error) {
	return PipeFindByPipeline[T](ctx, aggr.C(), aggr.GetPipeline(filter), limit)
}

func PipeFindOne[T MgoAggregate](ctx context.Context, aggr T, filter bson.M) error {
	if dataStore == nil {
		return ErrNotConnected
	}
	pipeline := aggr.GetPipeline(filter)
	collectionName := aggr.C()
	_, span := dataStore.startTraceSpan(ctx, collectionName, "pipeFindOne", pipeline)
	defer span.End()
	result := dataStore.PipeFindOne(ctx, collectionName, pipeline)
	raw, err := result.Raw()
	if err != nil {
		return spanErrorHandler(fmt.Errorf("%w: %w", ErrReadFailed, err), span)
	}
	if unmarshaler, ok := any(&aggr).(bson.Unmarshaler); ok {
		return spanErrorHandler(unmarshaler.UnmarshalBSON(raw), span)
	}
	return spanErrorHandler(result.Decode(&aggr), span)
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
