// Package mgo provides a high-level abstraction layer over the official MongoDB Go driver,
// simplifying connection management, document operations, and schema definitions.
package mgo

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Index defines the interface for a collection's schema, including its name and index specifications.
// This allows for the automatic creation and registration of indexes.
type Index interface {
	// C returns the name of the MongoDB collection.
	C() string
	// Indexes returns a slice of mongo.IndexModel, defining the indexes for the collection.
	Indexes() []mongo.IndexModel
}

// indexes holds all registered Index definitions for the application.
var indexes = []Index{}

// collectDef is a concrete implementation of the Index interface.
// It is used internally to store registered index information.
type collectDef struct {
	collectionName string
	indexes        []mongo.IndexModel
}

// C returns the collection name.
func (c *collectDef) C() string {
	return c.collectionName
}

// Indexes returns the defined index models.
func (c *collectDef) Indexes() []mongo.IndexModel {
	return c.indexes
}

// NewCollectDef creates a new collection definition that satisfies the Index interface.
// This is a helper function to simplify the creation of index definitions.
func NewCollectDef(name string, f func() []mongo.IndexModel) Index {
	return &collectDef{
		collectionName: name,
		indexes:        f(),
	}
}

// RegisterIndex adds a new Index definition to the global registry.
// This is typically called from the init() function of a model package.
func RegisterIndex(index Index) {
	indexes = append(indexes, index)
}

// SyncIndexes ensures that all registered indexes are present in the database.
// It iterates through all registered models and applies their index definitions.
// The underlying MongoDB driver's CreateMany command is idempotent: it will only
// create indexes that do not already exist and will not change existing ones.
// This is a safe and effective way to keep code-defined schemas and the database in sync.
func SyncIndexes(ctx context.Context) error {
	if dataStore == nil {
		return ErrNotConnected
	}

	// Iterate over all programmatically registered index definitions.
	for _, index := range indexes {
		// Skip if there are no indexes to create for this model.
		if len(index.Indexes()) == 0 {
			continue
		}

		// Get the index view for the collection.
		indexView := dataStore.getCollection(index.C()).Indexes()

		// Create the defined indexes. This command is idempotent.
		_, err := indexView.CreateMany(ctx, index.Indexes())
		if err != nil {
			// Wrap the error for better context.
			return fmt.Errorf(
				"failed to create indexes for collection '%s': %w",
				index.C(), errors.Join(ErrCreateIndexFailed, err),
			)
		}
	}

	return nil
}
