# Vulpes: A Go Microservice Toolkit

`vulpes` (Latin for *fox*) is a collection of cohesive, opinionated Go packages designed to accelerate the development of robust, production-ready, and maintainable microservices. It provides a solid foundation by handling common backend concerns, allowing developers to focus on business logic.

Built with a focus on gRPC and modern best practices, Vulpes offers sensible defaults and a clean, decoupled architecture.

## Core Philosophy

- **Productivity**: Get from idea to a running service quickly. Boilerplate for database connections, logging, gRPC servers, and more is handled for you.
- **Best Practices Built-in**: Encourages patterns like structured logging, dependency injection, and self-describing data models.
- **Decoupled and Extensible**: Each package is designed to be useful on its own, but they shine when used together. The architecture is modular and easy to extend.
- **Opinionated, Not Restrictive**: Provides strong defaults and patterns but allows for customization where it matters.

## Key Features

- **Rapid gRPC & RESTful API Development**: Easily set up a gRPC server that also serves a JSON RESTful gateway on the same port.
- **Type-Safe & Self-Describing Database Models**: An abstraction layer for MongoDB that encourages models to define their own schema and indexes.
- **High-Performance Structured Logging**: A simple, global logger built on Zap that provides environment-aware, structured output.
- **Flexible Data Serialization**: A generic `codec` package for serializing data structures, perfect for session data.
- **Built-in Authorization Hooks**: Includes a `relation` package designed to integrate with permission systems like Ory Keto.

## Package Overview

| Package | Description |
| :--- | :--- |
| **`log`** | A wrapper around Zap for high-performance, structured logging. Configurable for dev/prod environments. |
| **`errors`** | A simple utility for creating wrapped, traceable errors. |
| **`codec`** | A flexible serialization package (GOB, MessagePack) for encoding/decoding Go types to strings. |
| **`db/mgo`** | An abstraction layer for MongoDB that simplifies connection management and promotes self-describing models with automatic index creation. |
| **`validate`** | A helper for request validation, used by the gRPC interceptor. |
| **`ezgrpc`** | The core of the toolkit. Simplifies gRPC server and gateway setup, including interceptors for logging, metrics, validation, and session management. |
| **`relation`** | An interface for managing authorization tuples, designed for systems like Ory Keto. |

## Getting Started: A Complete Example

This example demonstrates how to build a simple `UserService` that creates a user and saves it to a database.

### 1. Define the Database Model (`models/user.go`)

First, define the `User` model using the `db/mgo` package patterns.

```go
package models

import (
	"github.com/94peter/vulpes/db/mgo"
	"github.com/94peter/vulpes/validate"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var userCollection = mgo.NewCollectDef("users", func() []mongo.IndexModel {
	return []mongo.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
	}
})

func init() {
	mgo.RegisterIndex(userCollection)
}

type User struct {
	mgo.Index   `bson:"-"`
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Name      string        `bson:"name" validate:"required"`
	Email     string        `bson:"email" validate:"required,email"`
	Password  string        `bson:"password" validate:"required"`
}

func (u *User) Validate() error {
	return validate.Struct(u)
}

func NewUser(name, email string) *User {
	return &User{
		Index: userCollection,
		ID:    bson.NewObjectID(),
		Name:  name,
		Email: email,
	}
}
```

### 2. Define the gRPC Service (`proto/user.proto`)

Define the service, request, and response messages.

```protobuf
syntax = "proto3";

package user;

option go_package = "path/to/your/user";

import "google/api/annotations.proto";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse) {
    option (google.api.http) = {
      post: "/v1/users",
      body: "*"
    };
  }
}

message CreateUserRequest {
  string name = 1;
  string email = 2;
}

message CreateUserResponse {
  string user_id = 1;
}
```

### 3. Implement the gRPC Service (`services/user.go`)

Implement the server logic and register it with `ezgrpc`.

```go
package services

import (
	"context"

	"github.com/94peter/vulpes/db/mgo"
	"github.com/94peter/vulpes/ezgrpc"
	"github.com/94peter/vulpes/log"
	"path/to/your/models"
	pb "path/to/your/user"
	"google.golang.org/grpc"
)

func init() {
	ezgrpc.InjectGrpcService(func(s *grpc.Server) {
		pb.RegisterUserServiceServer(s, &userService{})
	})
	ezgrpc.RegisterHandlerFromEndpoint(pb.RegisterUserServiceHandlerFromEndpoint)
}

type userService struct {
	pb.UnimplementedUserServiceServer
}

func (s *userService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	log.Info("Received CreateUser request", log.String("name", req.Name), log.String("email", req.Email))

	user := models.NewUser(req.Name, req.Email)

	if err := user.Validate(); err != nil {
		log.Error("Validation failed", log.Err(err))
		// The validation interceptor in ezgrpc will likely catch this first,
		// but it's good practice to validate here too.
		return nil, err
	}

	userCollection := mgo.GetCollection(user.C())
	_, err := userCollection.InsertOne(ctx, user)
	if err != nil {
		log.Error("Failed to insert user", log.Err(err))
		return nil, err
	}

	log.Info("Successfully created user", log.String("user_id", user.ID.Hex()))

	return &pb.CreateUserResponse{UserId: user.ID.Hex()}, nil
}
```

### 4. Tie It All Together (`main.go`)

Your main function is now incredibly simple. It just needs to initialize the Vulpes packages and run the server.

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/94peter/vulpes/db/mgo"
	"github.com/94peter/vulpes/ezgrpc"
	vulpeslog "github.com/94peter/vulpes/log"

	// Blank imports to trigger service and model registration
	_ "path/to/your/models"
	_ "path/to/your/services"
)

func main() {
	// 1. Configure logger (optional)
	vulpeslog.SetConfig(vulpeslog.WithDev(true), vulpeslog.WithLevel("debug"))

	// 2. Initialize MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := mgo.InitConnection(ctx, "myAppDB", mgo.WithURI("mongodb://localhost:27017")); err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer mgo.Close(ctx)

	// 3. Create database indexes if they don't exist
	if err := mgo.CreateIndexesIfNotExists(ctx); err != nil {
		log.Fatalf("Failed to create indexes: %v", err)
	}

	// 4. Initialize gRPC session store
	ezgrpc.InitSessionStore()

	// 5. Run the gRPC server and gateway
	vulpeslog.Info("Starting server on port 8080")
	if err := ezgrpc.RunGrpcGateway(context.Background(), 8080); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
```

## Installation

```bash
go get github.com/94peter/vulpes/...
```
