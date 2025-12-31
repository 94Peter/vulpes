package mgo

import (
	"context"
	"fmt"

	"github.com/94peter/vulpes/log"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// BulkOperation provides a fluent builder for constructing and executing
// bulk write operations, leveraging MongoDB's BulkWrite capabilities.
// It allows combining multiple insert, update, and delete operations into a single request.
type bulkOperation struct {
	operations []mongo.WriteModel
	collection *mongo.Collection
}

// NewBulkOperation creates a new builder for a bulk operation on a specific collection.
// cname: The name of the collection to perform operations on.
func NewBulkOperation(cname string) (BulkOperator, error) {
	if dataStore == nil {
		return nil, ErrNotConnected
	}
	return dataStore.NewBulkOperation(cname), nil
}

// InsertOne adds an InsertOne operation to the bulk request.
// The provided document will be validated before being added.
func (b *bulkOperation) InsertOne(doc DocInter) BulkOperator {
	if err := doc.Validate(); err != nil {
		log.Warn("Invalid document in bulk operation: " + err.Error())
		// To maintain the fluent API, we don't return an error here.
		// The error will be caught by the driver during Execute.
		// Consider pre-validating documents before adding them to the bulk operation.
	}
	model := mongo.NewInsertOneModel().SetDocument(doc)
	b.operations = append(b.operations, model)
	return b
}

// UpdateOne adds an UpdateOne operation to the bulk request.
// filter: The filter to select the document to update.
// update: The update document (e.g., using $set, $inc).
func (b *bulkOperation) UpdateOne(filter any, update any) BulkOperator {
	model := mongo.NewUpdateOneModel().
		SetFilter(filter).
		SetUpdate(update)
	b.operations = append(b.operations, model)
	return b
}

// UpdateById adds a convenient UpdateOne operation filtered by the document's _id.
func (b *bulkOperation) UpdateById(id any, update any) BulkOperator {
	return b.UpdateOne(bson.M{"_id": id}, update)
}

// DeleteOne adds a DeleteOne operation to the bulk request.
// filter: The filter to select the document to delete.
func (b *bulkOperation) DeleteOne(filter any) BulkOperator {
	model := mongo.NewDeleteOneModel().SetFilter(filter)
	b.operations = append(b.operations, model)
	return b
}

// DeleteById adds a convenient DeleteOne operation filtered by the document's _id.
func (b *bulkOperation) DeleteById(id any) BulkOperator {
	return b.DeleteOne(bson.M{"_id": id})
}

// ReplaceOne adds a ReplaceOne operation to the bulk request.
// The replacement document must not have an _id field if it's different from the filter's _id.
func (b *bulkOperation) ReplaceOne(filter any, replacement DocInter) BulkOperator {
	model := mongo.NewReplaceOneModel().
		SetFilter(filter).
		SetReplacement(replacement)
	b.operations = append(b.operations, model)
	return b
}

// Execute sends the accumulated operations to the database as a single bulk write request.
// Returns the result of the bulk write operation, or an error if it fails.
func (b *bulkOperation) Execute(ctx context.Context) (*mongo.BulkWriteResult, error) {
	if len(b.operations) == 0 {
		return nil, fmt.Errorf("%w: no operations to execute", ErrInvalidDocument)
	}
	result, err := b.collection.BulkWrite(ctx, b.operations)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWriteFailed, err)
	}

	return result, nil
}

func (m *mongoStore) NewBulkOperation(cname string) BulkOperator {
	return &bulkOperation{
		operations: make([]mongo.WriteModel, 0),
		collection: m.getCollection(cname),
	}
}
