package mgo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Datastore defines the interface for all database operations.
// It allows for mocking the entire package for testing purposes.
type Datastore interface {
	Save(ctx context.Context, doc DocInter) (DocInter, error)
	Find(ctx context.Context, collection string, filter any, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, error)
	FindOne(ctx context.Context, collection string, filter any, opts ...options.Lister[options.FindOneOptions]) *mongo.SingleResult
	UpdateOne(ctx context.Context, collection string, filter bson.D, update bson.D, opts ...options.Lister[options.UpdateOneOptions]) (int64, error)
	UpdateMany(ctx context.Context, collection string, filter bson.D, update bson.D) (int64, error)
	DeleteOne(ctx context.Context, collection string, filter bson.D) (int64, error)
	DeleteMany(ctx context.Context, collection string, filter bson.D) (int64, error)
	ReplaceOne(ctx context.Context, collection string, filter any, replacement any, opts ...options.Lister[options.ReplaceOptions]) (*mongo.UpdateResult, error)

	PipeFind(ctx context.Context, collection string, pipeline mongo.Pipeline) (*mongo.Cursor, error)
	PipeFindOne(ctx context.Context, collection string, pipeline mongo.Pipeline) *mongo.SingleResult

	NewBulkOperation(cname string) BulkOperator
	getCollection(name string) *mongo.Collection
	getDatabase() *mongo.Database
	close(ctx context.Context) error
	getClient() *mongo.Client
}

// BulkOperator defines the interface for the fluent bulk operation builder.
type BulkOperator interface {
	InsertOne(doc DocInter) BulkOperator
	UpdateOne(filter any, update any) BulkOperator
	UpdateById(id any, update any) BulkOperator
	DeleteOne(filter any) BulkOperator
	DeleteById(id any) BulkOperator

	Execute(ctx context.Context) (*mongo.BulkWriteResult, error)
}

var dataStore Datastore

type mongoStore struct {
	db *mongo.Database
}

func (m *mongoStore) getCollection(name string) *mongo.Collection {
	return m.db.Collection(name)
}

func (m *mongoStore) close(ctx context.Context) error {
	return m.db.Client().Disconnect(ctx)
}

func (m *mongoStore) getDatabase() *mongo.Database {
	return m.db
}

func (m *mongoStore) getClient() *mongo.Client {
	return m.db.Client()
}
