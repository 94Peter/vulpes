package mgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.opentelemetry.io/otel/attribute"
)

// DeleteOne deletes a single document matching the filter.
// doc: An instance of the document type, used to determine the collection.
func DeleteOne[T DocInter](ctx context.Context, doc T, filter bson.D) (int64, error) {
	if dataStore == nil {
		return 0, ErrNotConnected
	}
	data, _ := json.Marshal(filter)
	collectionName := doc.C()
	_, span := dataStore.startTraceSpan(ctx,
		fmt.Sprintf("mongo.deleteOne.%s", collectionName),
		attribute.String("db.collection", collectionName),
		attribute.String("db.operation", "deleteOne"),
		attribute.String("db.statement", string(data)),
	)
	defer span.End()
	affected, err := dataStore.DeleteOne(ctx, doc.C(), filter)
	if err != nil {
		return 0, spanErrorHandler(err, span)
	}
	span.SetAttributes(attribute.Int64("db.affected_number_of_documents", affected))
	return affected, spanErrorHandler(nil, span)
}

// DeleteMany deletes all documents matching the filter.
// doc: An instance of the document type, used to determine the collection.
func DeleteMany[T DocInter](ctx context.Context, doc T, filter bson.D) (int64, error) {
	if dataStore == nil {
		return 0, ErrNotConnected
	}
	data, _ := json.Marshal(filter)
	collectionName := doc.C()
	_, span := dataStore.startTraceSpan(ctx,
		fmt.Sprintf("mongo.deleteMany.%s", collectionName),
		attribute.String("db.collection", collectionName),
		attribute.String("db.operation", "deleteMany"),
		attribute.String("db.statement", string(data)),
	)
	defer span.End()
	affected, err := dataStore.DeleteMany(ctx, doc.C(), filter)
	if err != nil {
		return 0, spanErrorHandler(err, span)
	}
	span.SetAttributes(attribute.Int64("db.affected_number_of_documents", affected))
	return affected, spanErrorHandler(nil, span)
}

// DeleteById deletes a single document identified by the _id of the provided document instance.
// doc: An instance of the document, from which the _id is extracted for the filter.
func DeleteById[T DocInter](ctx context.Context, doc T) (int64, error) {
	if dataStore == nil {
		return 0, ErrNotConnected
	}
	return dataStore.DeleteOne(ctx, doc.C(), bson.D{{Key: "_id", Value: doc.GetId()}})
}

func (m *mongoStore) DeleteMany(ctx context.Context, collection string, filter bson.D) (int64, error) {
	result, err := m.getCollection(collection).DeleteMany(ctx, filter)
	if err != nil {
		return 0, errors.Join(ErrWriteFailed, err)
	}
	return result.DeletedCount, nil
}

func (m *mongoStore) DeleteOne(ctx context.Context, collection string, filter bson.D) (int64, error) {
	result, err := m.getCollection(collection).DeleteOne(ctx, filter)
	if err != nil {
		return 0, errors.Join(ErrWriteFailed, err)
	}
	return result.DeletedCount, nil
}
