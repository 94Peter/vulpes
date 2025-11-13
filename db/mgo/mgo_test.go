package mgo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/94peter/vulpes/db/mgo"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// testUser is a simple struct used for testing purposes.
type testUser struct {
	ID   bson.ObjectID `bson:"_id,omitempty"`
	Name string
	Age  int
}

// Implement the DocInter interface for testUser.
func (u *testUser) C() string                   { return "users" }
func (u *testUser) Indexes() []mongo.IndexModel { return nil }
func (u *testUser) Validate() error             { return nil }
func (u *testUser) GetId() any                  { return u.ID }
func (u *testUser) SetId(id any)                { u.ID = id.(bson.ObjectID) }

func TestFind(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		expectedUsers := []any{
			testUser{ID: bson.NewObjectID(), Name: "Peter", Age: 30},
			testUser{ID: bson.NewObjectID(), Name: "Alice", Age: 25},
		}
		mockDB := &mgo.MockDatastore{
			OnFind: mgo.NewOnFindMock(expectedUsers...),
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		var result []*testUser
		result, err := mgo.Find(context.Background(), &testUser{}, nil)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, expectedUsers[0].(testUser).Name, result[0].Name)
		assert.Equal(t, expectedUsers[1].(testUser).Name, result[1].Name)
	})

	t.Run("Error from Datastore", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("datastore find failed")
		mockDB := &mgo.MockDatastore{
			OnFind: mgo.NewErrOnFind(expectedErr),
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		var result []*testUser
		result, err := mgo.Find(context.Background(), &testUser{}, nil)

		// Assert
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestFindOne(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		expectedUser := testUser{ID: bson.NewObjectID(), Name: "Peter"}
		mockDB := &mgo.MockDatastore{
			OnFindOne: mgo.NewOnFindOneMock(expectedUser),
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		var foundUser testUser
		err := mgo.FindOne(context.Background(), &foundUser, nil)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, foundUser)
	})

	t.Run("Error No Documents", func(t *testing.T) {
		// Arrange
		mockDB := &mgo.MockDatastore{
			OnFindOne: mgo.NewErrOnFindOne(mongo.ErrNoDocuments),
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		var foundUser testUser
		err := mgo.FindOne(context.Background(), &foundUser, nil)

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, mongo.ErrNoDocuments)
	})
}

func TestFindById(t *testing.T) {
	// Arrange
	userID := bson.NewObjectID()
	expectedUser := testUser{ID: userID, Name: "Peter"}

	mockDB := &mgo.MockDatastore{
		// We mock FindOne because FindById calls it internally.
		OnFindOne: func(ctx context.Context, collection string, filter any, opts ...options.Lister[options.FindOneOptions]) *mongo.SingleResult {
			// Assert that the filter passed by FindById is correct.
			filterMap := filter.(bson.M)
			assert.Equal(t, userID, filterMap["_id"])

			// Return the expected user.
			return mongo.NewSingleResultFromDocument(expectedUser, nil, nil)
		},
	}
	restore := mgo.SetDatastore(mockDB)
	defer restore()

	// Act: Call FindById with a user struct that has the ID we want to find.
	userToFind := &testUser{ID: userID}
	err := mgo.FindById(context.Background(), userToFind)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedUser.Name, userToFind.Name)
}

func TestUpdateOne(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		filter := bson.D{{Key: "name", Value: "old_name"}}
		update := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: "new_name"}}}}
		expectedModifiedCount := int64(1)

		mockDB := &mgo.MockDatastore{
			OnUpdateOne: func(ctx context.Context, collection string, f bson.D, u bson.D) (int64, error) {
				assert.Equal(t, "users", collection)
				assert.Equal(t, filter, f)
				assert.Equal(t, update, u)
				return expectedModifiedCount, nil
			},
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		modifiedCount, err := mgo.UpdateOne(context.Background(), &testUser{}, filter, update)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedModifiedCount, modifiedCount)
	})

	t.Run("Error from Datastore", func(t *testing.T) {
		// Arrange
		filter := bson.D{{Key: "name", Value: "old_name"}}
		update := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: "new_name"}}}}
		expectedErr := errors.New("datastore update failed")

		mockDB := &mgo.MockDatastore{
			OnUpdateOne: func(ctx context.Context, collection string, f bson.D, u bson.D) (int64, error) {
				return 0, expectedErr
			},
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		modifiedCount, err := mgo.UpdateOne(context.Background(), &testUser{}, filter, update)

		// Assert
		assert.Zero(t, modifiedCount)
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestUpdateById(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		userID := bson.NewObjectID()
		user := &testUser{ID: userID}
		update := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: "new_name"}}}}
		expectedModifiedCount := int64(1)

		mockDB := &mgo.MockDatastore{
			OnUpdateOne: func(ctx context.Context, collection string, f bson.D, u bson.D) (int64, error) {
				assert.Equal(t, "users", collection)
				assert.Equal(t, bson.D{{Key: "_id", Value: userID}}, f)
				assert.Equal(t, update, u)
				return expectedModifiedCount, nil
			},
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		modifiedCount, err := mgo.UpdateById(context.Background(), user, update)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedModifiedCount, modifiedCount)
	})

	t.Run("Error from Datastore", func(t *testing.T) {
		// Arrange
		userID := bson.NewObjectID()
		user := &testUser{ID: userID}
		update := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: "new_name"}}}}
		expectedErr := errors.New("datastore update by id failed")

		mockDB := &mgo.MockDatastore{
			OnUpdateOne: func(ctx context.Context, collection string, f bson.D, u bson.D) (int64, error) {
				return 0, expectedErr
			},
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		modifiedCount, err := mgo.UpdateById(context.Background(), user, update)

		// Assert
		assert.Zero(t, modifiedCount)
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})
}

// testUserWithValidationError is a test struct that always returns a validation error.
type testUserWithValidationError struct {
	testUser
}

func (u *testUserWithValidationError) Validate() error {
	return errors.New("validation failed for test user")
}

func TestSave(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		user := &testUser{Name: "Peter"}
		assert.True(t, user.ID.IsZero(), "Initial ID should be zero")

		mockDB := &mgo.MockDatastore{
			OnSave: mgo.NewOnSaveMock(),
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		savedUser, err := mgo.Save(context.Background(), user)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, savedUser)
		assert.False(t, savedUser.GetId().(bson.ObjectID).IsZero(), "Saved document ID should not be zero")
		assert.Equal(t, user.Name, savedUser.Name, "Saved user name should match original")
	})

	t.Run("Error from Datastore", func(t *testing.T) {
		// Arrange
		user := &testUser{Name: "Peter"}
		expectedErr := errors.New("datastore save failed")

		mockDB := &mgo.MockDatastore{
			OnSave: func(ctx context.Context, doc mgo.DocInter) (mgo.DocInter, error) {
				return nil, expectedErr
			},
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		savedUser, err := mgo.Save(context.Background(), user)

		// Assert
		assert.Nil(t, savedUser)
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("Validation Failed", func(t *testing.T) {
		// Arrange
		user := &testUserWithValidationError{testUser: testUser{Name: "Invalid"}}
		mockDB := &mgo.MockDatastore{
			// OnSave should not be called if validation fails, but we set it just in case.
			OnSave: mgo.NewOnSaveMock(),
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		savedUser, err := mgo.Save(context.Background(), user)

		// Assert
		assert.Nil(t, savedUser)
		assert.Error(t, err)
		assert.ErrorIs(t, err, mgo.ErrInvalidDocument)
		assert.Contains(t, err.Error(), "validation failed for test user")
	})

	t.Run("Document is Nil", func(t *testing.T) {
		// Arrange
		var user *testUser = nil // Explicitly nil pointer
		mockDB := &mgo.MockDatastore{
			// OnSave should not be called if doc is nil
			OnSave: mgo.NewOnSaveMock(),
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		savedUser, err := mgo.Save(context.Background(), user)

		// Assert
		assert.Nil(t, savedUser)
		assert.Error(t, err)
		assert.ErrorIs(t, err, mgo.ErrInvalidDocument)
		assert.Contains(t, err.Error(), "document cannot be nil")
	})
}

// testAggregate is a simple struct used for testing PipeFind.
type testAggregate struct {
	CollectionName string
	Pipeline       mongo.Pipeline
	Name           string
}

// Implement the MgoAggregate interface for testAggregate.
func (t *testAggregate) GetPipeline(q bson.M) mongo.Pipeline { return t.Pipeline }
func (t *testAggregate) C() string                           { return t.CollectionName }
func (t *testAggregate) Indexes() []mongo.IndexModel         { return nil }

func TestPipeFind(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		expectedUsers := []testUser{
			{ID: bson.NewObjectID(), Name: "Peter", Age: 30},
		}
		fakeData := []any{expectedUsers[0]}

		mockDB := &mgo.MockDatastore{
			OnPipeFind: mgo.NewOnPipeFindMock(fakeData...),
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		aggr := &testAggregate{
			CollectionName: "users",
			Pipeline:       []bson.D{{{Key: "$match", Value: bson.D{{Key: "name", Value: "Peter"}}}}},
		}

		// Act
		var result []*testAggregate
		result, err := mgo.PipeFind(context.Background(), aggr, nil) // filter is nil for PipeFind

		// Assert
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, expectedUsers[0].Name, result[0].Name)
	})

	t.Run("Error from Datastore", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("datastore pipefind failed")
		mockDB := &mgo.MockDatastore{
			OnPipeFind: func(ctx context.Context, collection string, pipeline mongo.Pipeline) (*mongo.Cursor, error) {
				return nil, expectedErr
			},
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		aggr := &testAggregate{
			CollectionName: "users",
			Pipeline:       []bson.D{{{Key: "$match", Value: bson.D{{Key: "name", Value: "Peter"}}}}},
		}

		// Act
		var result []*testAggregate
		result, err := mgo.PipeFind(context.Background(), aggr, nil)

		// Assert
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestDeleteOne(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		filter := bson.D{{Key: "name", Value: "Peter"}}
		expectedDeletedCount := int64(1)

		mockDB := &mgo.MockDatastore{
			OnDeleteOne: func(ctx context.Context, collection string, f bson.D) (int64, error) {
				assert.Equal(t, "users", collection)
				assert.Equal(t, filter, f)
				return expectedDeletedCount, nil
			},
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		deletedCount, err := mgo.DeleteOne(context.Background(), &testUser{}, filter)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedDeletedCount, deletedCount)
	})

	t.Run("Error from Datastore", func(t *testing.T) {
		// Arrange
		filter := bson.D{{Key: "name", Value: "Peter"}}
		expectedErr := errors.New("datastore delete failed")

		mockDB := &mgo.MockDatastore{
			OnDeleteOne: func(ctx context.Context, collection string, f bson.D) (int64, error) {
				return 0, expectedErr
			},
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		deletedCount, err := mgo.DeleteOne(context.Background(), &testUser{}, filter)

		// Assert
		assert.Zero(t, deletedCount)
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestDeleteById(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		userID := bson.NewObjectID()
		user := &testUser{ID: userID}
		expectedDeletedCount := int64(1)

		mockDB := &mgo.MockDatastore{
			OnDeleteOne: func(ctx context.Context, collection string, f bson.D) (int64, error) {
				assert.Equal(t, "users", collection)
				assert.Equal(t, bson.D{{Key: "_id", Value: userID}}, f)
				return expectedDeletedCount, nil
			},
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		deletedCount, err := mgo.DeleteById(context.Background(), user)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedDeletedCount, deletedCount)
	})

	t.Run("Error from Datastore", func(t *testing.T) {
		// Arrange
		userID := bson.NewObjectID()
		user := &testUser{ID: userID}
		expectedErr := errors.New("datastore delete by id failed")

		mockDB := &mgo.MockDatastore{
			OnDeleteOne: func(ctx context.Context, collection string, f bson.D) (int64, error) {
				return 0, expectedErr
			},
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		deletedCount, err := mgo.DeleteById(context.Background(), user)

		// Assert
		assert.Zero(t, deletedCount)
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestDeleteMany(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		filter := bson.D{{Key: "age", Value: 30}}
		expectedDeletedCount := int64(2)

		mockDB := &mgo.MockDatastore{
			OnDeleteMany: func(ctx context.Context, collection string, f bson.D) (int64, error) {
				assert.Equal(t, "users", collection)
				assert.Equal(t, filter, f)
				return expectedDeletedCount, nil
			},
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		deletedCount, err := mgo.DeleteMany(context.Background(), &testUser{}, filter)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedDeletedCount, deletedCount)
	})

	t.Run("Error from Datastore", func(t *testing.T) {
		// Arrange
		filter := bson.D{{Key: "age", Value: 30}}
		expectedErr := errors.New("datastore delete many failed")

		mockDB := &mgo.MockDatastore{
			OnDeleteMany: func(ctx context.Context, collection string, f bson.D) (int64, error) {
				return 0, expectedErr
			},
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		deletedCount, err := mgo.DeleteMany(context.Background(), &testUser{}, filter)

		// Assert
		assert.Zero(t, deletedCount)
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})
}
