// Package mgo provides a high-level abstraction layer over the official MongoDB Go driver,
// simplifying connection management, document operations, and schema definitions.
package mgo

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

var once sync.Once

// Option defines a function signature for configuring the MongoDB client.
// This follows the functional options pattern, allowing for flexible and clear configuration.
type Option func(*options.ClientOptions)

// WithURI sets the MongoDB connection URI.
func WithURI(uri string) Option {
	return func(o *options.ClientOptions) {
		o.ApplyURI(uri)
	}
}

// WithMaxPoolSize specifies the maximum number of connections allowed in the connection pool.
func WithMaxPoolSize(size uint64) Option {
	return func(o *options.ClientOptions) {
		o.SetMaxPoolSize(size)
	}
}

// WithMinPoolSize specifies the minimum number of connections to maintain in the connection pool.
func WithMinPoolSize(size uint64) Option {
	return func(o *options.ClientOptions) {
		o.SetMinPoolSize(size)
	}
}

// WithMaxConnIdleTime sets the maximum duration that a connection can remain idle in the pool.
func WithMaxConnIdleTime(d time.Duration) Option {
	return func(o *options.ClientOptions) {
		o.SetMaxConnIdleTime(d)
	}
}

var isConnected bool

// InitConnection establishes a connection to the MongoDB server using a singleton pattern.
// It is safe to call this function multiple times; the connection will only be initialized once.
//
// ctx: A context for the connection process.
// dbName: The name of the database to connect to.
// opts: A variadic set of Option functions for configuration.
func InitConnection(ctx context.Context, dbName string, opts ...Option) error {
	var err error
	once.Do(func() {
		var client *mongo.Client
		clientOpts := options.Client()
		// Default to reading from secondary nodes if available, improving read performance.
		clientOpts.SetReadPreference(readpref.SecondaryPreferred())

		// Apply all user-provided configuration options.
		for _, o := range opts {
			o(clientOpts)
		}

		// Establish the connection to the server.
		client, err = mongo.Connect(clientOpts)
		if err != nil {
			err = errors.Join(ErrConnectionFailed, err)
			return
		}

		// Ping the primary node to verify that the connection is alive.
		err = client.Ping(ctx, readpref.Primary())
		if err != nil {
			err = errors.Join(ErrPingFailed, err)
			return
		}

		dataStore = &mongoStore{
			db: client.Database(dbName),
		}
		isConnected = true
	})

	return err
}

// Close gracefully disconnects the client from the MongoDB server.
// It should be called at the end of the application's lifecycle, for example, using defer in main.
func Close(ctx context.Context) error {
	if dataStore != nil {
		return dataStore.close(ctx)
	}
	return nil
}

func IsConnected() bool {
	return isConnected
}

func IsCollectionExist(ctx context.Context, collectionName string) (bool, error) {
	result, err := dataStore.getDatabase().ListCollectionNames(ctx, bson.D{{Key: "name", Value: collectionName}})
	if err != nil {
		return false, err
	}
	return len(result) > 0, nil
}
