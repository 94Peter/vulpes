package mgo_test

import (
	"testing"

	"github.com/94peter/vulpes/db/mgo"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestPipeFindOne(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		defer cleanDb()
		ctx := t.Context()
		// Arrange
		for i := range 10 {
			expectedUser := testUser{ID: bson.NewObjectID(), Name: "Peter", Age: i}
			savedUser, err := mgo.Save(ctx, &expectedUser)
			require.NoError(t, err)
			require.NotNil(t, savedUser)
		}

		// Act
		aggr := &testAggregate{
			CollectionName: "users",
			Pipeline:       []bson.D{{bson.E{Key: "$match", Value: bson.D{bson.E{Key: "name", Value: "Peter"}}}}},
		}
		result, err := mgo.PipeFind(ctx, aggr, nil, 20)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "Peter", result[0].Name)
		assert.Len(t, result, 10)
		for _, r := range result {
			assert.Equal(t, "Peter", r.Name)
		}
	})
}
