package mgo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/94peter/vulpes/db/mgo"
)

// func TestPipeFindOne(t *testing.T) {
// 	mgo.InitConnection(context.Background(), "test_db", mgo.WithURI("mongodb://mongodb.dev.orb.local:27017"))
// 	defer mgo.Close(context.Background())
// 	t.Run("Success", func(t *testing.T) {
// 		// Arrange
// 		expectedUser := testUser{ID: bson.NewObjectID(), Name: "Peter"}
// 		savedUser, err := mgo.Save(context.Background(), &expectedUser)
// 		assert.NoError(t, err)
// 		assert.NotNil(t, savedUser)

// 		// Act
// 		aggr := &testAggregate{
// 			CollectionName: "users",
// 			Pipeline:       []bson.D{{{Key: "$match", Value: bson.D{{Key: "name", Value: "Peter"}}}}},
// 		}
// 		err = mgo.PipeFindOne(context.Background(), aggr, nil)
// 		fmt.Println(aggr.Name, aggr.CollectionName)

// 		foundUsers, err := mgo.PipeFind(context.Background(), aggr, nil)
// 		fmt.Println(foundUsers, len(foundUsers))

// 		// Assert
// 		assert.NoError(t, err)
// 		assert.Equal(t, expectedUser.Name, aggr.Name)
// 		assert.False(t, true)
// 	})
// }

func TestUpdateMany(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		filter := bson.D{{Key: "age", Value: bson.D{{Key: "$gt", Value: 25}}}}
		update := bson.D{{Key: "$set", Value: bson.D{{Key: "Name", Value: "Over 25"}}}}
		expectedModifiedCount := int64(2)

		mockDB := &mgo.MockDatastore{
			OnUpdateMany: func(_ context.Context, collection string, f bson.D, u bson.D) (int64, error) {
				assert.Equal(t, "users", collection)
				assert.Equal(t, filter, f)
				assert.Equal(t, update, u)
				return expectedModifiedCount, nil
			},
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		modifiedCount, err := mgo.UpdateMany(context.Background(), &testUser{}, filter, update)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expectedModifiedCount, modifiedCount)
	})

	t.Run("Error from Datastore", func(t *testing.T) {
		// Arrange
		filter := bson.D{{Key: "age", Value: bson.D{{Key: "$gt", Value: 25}}}}
		update := bson.D{{Key: "$set", Value: bson.D{{Key: "Name", Value: "Over 25"}}}}
		expectedErr := errors.New("datastore update many failed")

		mockDB := &mgo.MockDatastore{
			OnUpdateMany: func(_ context.Context, _ string, _, _ bson.D) (int64, error) {
				return 0, expectedErr
			},
		}
		restore := mgo.SetDatastore(mockDB)
		defer restore()

		// Act
		modifiedCount, err := mgo.UpdateMany(context.Background(), &testUser{}, filter, update)

		// Assert
		assert.Zero(t, modifiedCount)
		require.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})
}
