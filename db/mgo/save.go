package mgo

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.opentelemetry.io/otel/attribute"
)

func Save[T DocInter](ctx context.Context, doc T) (T, error) {
	var zero T
	if dataStore == nil {
		return zero, ErrNotConnected
	}
	collectionName := doc.C()
	_, span := dataStore.startTraceSpan(ctx, collectionName, "save", nil)
	defer span.End()
	newDoc, err := dataStore.Save(ctx, doc)
	if err != nil {
		return zero, spanErrorHandler(fmt.Errorf("%w: %w", ErrWriteFailed, err), span)
	}
	result, ok := newDoc.(T)
	if !ok {
		return zero, spanErrorHandler(fmt.Errorf("%w: failed to cast to %T", ErrWriteFailed, doc), span)
	}
	if oid, ok := result.GetId().(bson.ObjectID); ok {
		span.SetAttributes(attribute.String("db.inserted_id", oid.Hex()))
	}
	return result, spanErrorHandler(nil, span)
}

func (m *mongoStore) Save(ctx context.Context, doc DocInter) (DocInter, error) {
	// 1. Restore the nil check for robustness.
	if v := reflect.ValueOf(doc); v.Kind() == reflect.Ptr && v.IsNil() {
		return nil, fmt.Errorf("%w: %w", ErrInvalidDocument, errors.New("document cannot be nil"))
	}

	// 2. Restore the validation check.
	if err := doc.Validate(); err != nil {
		return doc, fmt.Errorf("%w: %w", ErrInvalidDocument, err)
	}

	// 3. Perform the database operation.
	c := m.getCollection(doc.C())
	result, err := c.InsertOne(ctx, doc)
	if err != nil {
		return doc, fmt.Errorf("%w: %w", ErrWriteFailed, err)
	}
	doc.SetId(result.InsertedID)
	return doc, nil
}
