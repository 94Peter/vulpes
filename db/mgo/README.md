# MongoDB Abstraction Layer (db/mgo)

`db/mgo` is a high-level Go package designed to simplify interactions with MongoDB. It acts as a thoughtful wrapper around the official [MongoDB Go Driver](https://github.com/mongodb/mongo-go-driver), providing a streamlined and opinionated approach to common database operations.

## Core Philosophy

The primary goal of this package is to promote a consistent, maintainable, and self-describing data access layer. It achieves this through:

- **Singleton Connection Management**: Ensures a single, efficiently managed database connection for the entire application.
- **Standardized Document Interface (`DocInter`)**: Mandates that all data models define their own collection name, validation rules, and database indexes.
- **Automated Schema Management**: Automatically creates collections and indexes based on model definitions, reducing manual setup and preventing configuration drift.
- **Decoupled Schema Definition**: Encourages separating the collection's schema (name, indexes) from the model's data structure, promoting cleaner code.

## Key Features

- **Singleton Connection**: Uses `sync.Once` to guarantee a single, thread-safe database connection.
- **Functional Options**: Provides a clean, flexible API for configuring the database connection and for constructing model instances.
- **Self-Describing Models**: The `DocInter` and `Index` interfaces encourage models to be self-contained and aware of their database schema.
- **Automatic Index Creation**: On application startup, automatically creates necessary indexes for collections that don't yet exist.
- **Fluent Bulk Operations**: Provides a `BulkOperation` builder for safely and efficiently executing multiple `insert`, `update`, or `delete` operations in a single request.

## How to Use

This guide demonstrates the recommended pattern for defining a model, its schema, and performing database operations.

### 1. Define Your Model and Schema

Create your data model and its corresponding schema definition. The schema should define the collection name and indexes. Use an `init()` function to automatically register this schema.

The model itself should embed the `mgo.Index` interface, which will be fulfilled by injecting the schema definition during instantiation.

**`models/user.go`**
```go
package models

import (
	"github.com/94peter/vulpes/db/mgo"
	"github.com/94peter/vulpes/validate"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// userCollection defines the schema for the "users" collection.
// It uses NewCollectDef to create a reusable, self-contained schema definition.
var userCollection = mgo.NewCollectDef("users", func() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}
})

// init automatically registers the User model's indexes when the package is imported.
func init() {
	mgo.RegisterIndex(userCollection)
}

// User represents the data structure for a user document.
type User struct {
	mgo.Index   `bson:"-"` // Embed the Index interface, ignored by BSON marshalling.
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Name      string        `bson:"name" validate:"required"
	Email     string        `bson:"email" validate:"required,email"
}

// Validate implements the validation logic for a User.
func (u *User) Validate() error {
	return validate.Struct(u)
}

// userOption defines the functional option type for creating a User.
type userOption func(*User)

// NewUser is a constructor that creates a new User instance with the given options.
// It injects the collection schema and sets default values.
func NewUser(opts ...userOption) *User {
	u := &User{
		Index: userCollection, // Inject the collection definition.
		ID:    bson.NewObjectID(),
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

// WithUserName is a functional option to set the user's name.
func WithUserName(name string) userOption {
	return func(u *User) {
		u.Name = name
	}
}

// WithUserEmail is a functional option to set the user's email.
func WithUserEmail(email string) userOption {
	return func(u *User) {
		u.Email = email
	}
}
```

### 2. Initialize and Use in `main.go`

In your application's entry point, initialize the database connection. Use the model's constructor (`NewUser`) to create new instances safely.

**`main.go`**
```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/94peter/vulpes/db/mgo"
	// Blank import to trigger the init() function in the models package.
	_ "path/to/your/models"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Initialize the singleton MongoDB connection.
	err := mgo.InitConnection(
		ctx,
		"myAppDatabase", // The name of your database
		mgo.WithURI("mongodb://localhost:27017"),
		mgo.WithMaxPoolSize(20),
	)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mgo.Close(ctx);

	fmt.Println("Successfully connected to MongoDB.")

	// 2. Sync indexes for all registered models.
	if err := mgo.SyncIndexes(ctx); err != nil {
		log.Fatalf("Failed to sync indexes: %v", err)
	}
	fmt.Println("Indexes are up to date.")

	// 3. Example: Performing a bulk operation.
	bulkOp := mgo.NewBulkOperation("users")
	bulkOp.InsertOne(models.NewUser(
		models.WithUserName("Peter"),
		models.WithUserEmail("peter@example.com"),
	))
	bulkOp.InsertOne(models.NewUser(
		models.WithUserName("Alice"),
		models.WithUserEmail("alice@example.com"),
	))
	// You can also chain other operations like Update or Delete
	// bulkOp.UpdateById(someId, bson.D{{"$set", bson.M{{"name": "New Name"}}}})

	result, err := bulkOp.Execute(ctx)
	if err != nil {
		log.Fatalf("Bulk operation failed: %v", err)
	}
	fmt.Printf("Successfully inserted %d documents.\n", result.InsertedCount)
}
