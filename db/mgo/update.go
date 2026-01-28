package mgo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/otel/attribute"
)

// UpdateById updates a single document identified by the _id field of the provided document instance.
// It offers a flexible way to apply any update operator ($set, $inc, etc.).
//
// doc: An instance of the document, from which the _id is extracted for the filter.
//
//	It is also used to determine the target collection.
//
// update: The update document, e.g., bson.D{{"$set", bson.D{{"field", "value"}}}}.
func UpdateById[T DocInter](ctx context.Context, doc T, update bson.D) (int64, error) {
	return UpdateOne(ctx, doc, bson.D{bson.E{Key: "_id", Value: doc.GetId()}}, update)
}

// UpdateOne updates the first document that matches a given filter.
// This is a generic and flexible update function.
//
// doc: An instance of the document type, used to determine the collection.
// filter: The filter to select the document to update.
// update: The update document, e.g., bson.D{{"$set", bson.D{{"field", "value"}}}}.
func UpdateOne[T DocInter](ctx context.Context, doc T, filter bson.D, update bson.D) (int64, error) {
	if dataStore == nil {
		return 0, ErrNotConnected
	}
	collectionName := doc.C()
	_, span := dataStore.startTraceSpan(ctx, collectionName, "updateOne", filter)
	defer span.End()
	affected, err := dataStore.UpdateOne(ctx, doc.C(), filter, update)
	if err != nil {
		return 0, spanErrorHandler(err, span)
	}
	span.SetAttributes(attribute.Int64("db.affected_number_of_documents", affected))
	return affected, spanErrorHandler(nil, span)
}

func UpdateMany[T DocInter](ctx context.Context, doc T, filter bson.D, update bson.D) (int64, error) {
	if dataStore == nil {
		return 0, ErrNotConnected
	}
	return dataStore.UpdateMany(ctx, doc.C(), filter, update)
}

func (m *mongoStore) UpdateOne(
	ctx context.Context, collection string, filter bson.D, update bson.D,
	opts ...options.Lister[options.UpdateOneOptions],
) (int64, error) {
	result, err := m.getCollection(collection).UpdateOne(ctx, filter, update, opts...)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrWriteFailed, err)
	}
	return result.ModifiedCount, nil
}

// UpdateMany updates all documents that match a given filter.
func (m *mongoStore) UpdateMany(ctx context.Context, collection string, filter bson.D, update bson.D) (int64, error) {
	result, err := m.getCollection(collection).UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrWriteFailed, err)
	}
	return result.ModifiedCount, nil
}
