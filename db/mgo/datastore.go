package mgo

import (
	"context"
	"encoding/json"
	"io"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Datastore defines the interface for all database operations.
// It allows for mocking the entire package for testing purposes.
type Datastore interface {
	Save(ctx context.Context, doc DocInter) (DocInter, error)
	CountDocument(
		ctx context.Context, collectionName string, filter any,
	) (int64, error)
	Find(
		ctx context.Context, collection string, filter any,
		opts ...options.Lister[options.FindOptions],
	) (*mongo.Cursor, error)
	FindOne(
		ctx context.Context, collection string, filter any,
		opts ...options.Lister[options.FindOneOptions],
	) *mongo.SingleResult
	UpdateOne(
		ctx context.Context, collection string, filter bson.D, update bson.D,
		opts ...options.Lister[options.UpdateOneOptions],
	) (int64, error)
	UpdateMany(ctx context.Context, collection string, filter bson.D, update bson.D) (int64, error)
	DeleteOne(ctx context.Context, collection string, filter bson.D) (int64, error)
	DeleteMany(ctx context.Context, collection string, filter bson.D) (int64, error)
	ReplaceOne(
		ctx context.Context, collection string, filter any, replacement any,
		opts ...options.Lister[options.ReplaceOptions],
	) (*mongo.UpdateResult, error)

	PipeFind(
		ctx context.Context, collection string, pipeline mongo.Pipeline,
	) (*mongo.Cursor, error)
	PipeFindOne(ctx context.Context, collection string, pipeline mongo.Pipeline) *mongo.SingleResult

	Distinct(
		ctx context.Context, collectionName string, field string, filter any,
		opts ...options.Lister[options.DistinctOptions],
	) ([]bson.RawValue, error)

	Import(
		ctx context.Context, collectionName string, reader io.Reader,
	) error

	NewBulkOperation(cname string) BulkOperator
	getCollection(name string) *mongo.Collection
	getDatabase() *mongo.Database
	close(ctx context.Context) error
	getClient() *mongo.Client
	startTraceSpan(
		ctx context.Context,
		collectionName string,
		operation string,
		statement any,
	) (context.Context, trace.Span)
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
	db     *mongo.Database
	tracer trace.Tracer
	isNoop bool
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

const dbSystem = "mongodb"

func (m *mongoStore) startTraceSpan(
	ctx context.Context, collectionName string, operation string, statement any,
) (context.Context, trace.Span) {
	if m.isNoop {
		// 直接返回一個 no-op 的 Span，或者根據你的需求返回 ctx, nil
		// 注意：如果返回 nil，後續呼叫 span.End() 時要加檢查
		return ctx, trace.SpanFromContext(ctx)
	}
	name := "mongo." + operation + "." + collectionName
	ctx, span := m.tracer.Start(ctx, name, trace.WithSpanKind(trace.SpanKindClient))
	span.SetAttributes(
		attribute.String("db.system", dbSystem),
		attribute.String("db.collection", collectionName),
		attribute.String("db.operation", operation),
	)
	if span.IsRecording() && statement != nil {
		data, _ := json.Marshal(statement)
		span.SetAttributes(attribute.String("db.statement", string(data)))
	}
	return ctx, span
}

func spanErrorHandler(err error, span trace.Span) error {
	if span == nil {
		return err
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "ok")
	}
	return err
}
