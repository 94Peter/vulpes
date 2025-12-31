package mgo

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// testUser is a simple struct used for testing purposes.
type testUser struct {
	ID   bson.ObjectID `bson:"_id,omitempty"`
	Name string
	Age  int
}

// Implement the DocInter interface for testUser.
func (*testUser) C() string                   { return "users" }
func (*testUser) Indexes() []mongo.IndexModel { return nil }
func (*testUser) Validate() error             { return nil }
func (u *testUser) GetId() any                { return u.ID }
func (u *testUser) SetId(id any)              { u.ID = id.(bson.ObjectID) }

func TestNewOnFindMock(t *testing.T) {
	// Arrange: Define the fake data we want the mock cursor to return.
	expectedUsers := []testUser{
		{Name: "Peter", Age: 30},
		{Name: "Alice", Age: 25},
	}

	fakeData := []any{expectedUsers[0], expectedUsers[1]}

	// Act: Call the function we want to test to get the mock implementation.
	onFindFunc := NewOnFindMock(fakeData...)

	// Call the generated function to get the mock cursor.
	cursor, err := onFindFunc(context.Background(), "users", nil)

	// Assert: Verify the results.
	require.NoError(t, err, "The mock function itself should not return an error")
	assert.NotNil(t, cursor, "The returned cursor should not be nil")

	// Try to decode the data from the cursor.
	var decodedUsers []testUser
	err = cursor.All(context.Background(), &decodedUsers)

	// Assert that decoding works and the data matches our original fake data.
	require.NoError(t, err, "cursor.All should decode without errors")
	assert.Equal(t, expectedUsers, decodedUsers, "The decoded data should match the fake data")
}

func TestNewOnFindOneMock(t *testing.T) {
	// Arrange
	expectedUser := testUser{Name: "Peter", Age: 30}

	// Act
	onFindOneFunc := NewOnFindOneMock(expectedUser)
	singleResult := onFindOneFunc(context.Background(), "users", nil)

	// Assert
	assert.NotNil(t, singleResult, "The returned SingleResult should not be nil")

	var decodedUser testUser
	err := singleResult.Decode(&decodedUser)

	require.NoError(t, err, "Decode should not return an error")
	assert.Equal(t, expectedUser, decodedUser, "The decoded user should match the expected user")
}

func TestNewErrOnFind(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")

	// Act
	onFindFunc := NewErrOnFind(expectedErr)
	cursor, err := onFindFunc(context.Background(), "users", nil)

	// Assert
	require.Nil(t, cursor, "Cursor should be nil on error")
	require.Error(t, err, "An error should be returned")
	assert.Equal(t, expectedErr, err, "The returned error should be the one we provided")
}

func TestNewErrOnFindOne(t *testing.T) {
	// Arrange
	expectedErr := mongo.ErrNoDocuments

	// Act
	onFindOneFunc := NewErrOnFindOne(expectedErr)
	singleResult := onFindOneFunc(context.Background(), "users", nil)

	// Assert
	var decodedUser testUser
	err := singleResult.Decode(&decodedUser)

	require.Error(t, err, "Decode should return an error")
	assert.ErrorIs(t, err, expectedErr, "The error should be mongo.ErrNoDocuments")
}

func TestNewOnSaveMock(t *testing.T) {
	// Arrange
	user := &testUser{Name: "Peter"}
	assert.True(t, user.ID.IsZero(), "Initial ID should be zero")

	// Act
	onSaveFunc := NewOnSaveMock()
	savedUser, err := onSaveFunc(context.Background(), user)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, savedUser)
	assert.False(t, savedUser.GetId().(bson.ObjectID).IsZero(), "Saved document ID should not be zero")
}

func TestNewOnPipeFindMock(t *testing.T) {
	// Arrange
	// 1. Define the expected results with the concrete type.
	expectedResults := []map[string]any{
		{"name": "Peter"},
	}

	// 2. Create a separate slice of type []any for the mock factory function.
	//    The factory needs this type for its variadic ...any parameter.
	fakeDataForMock := make([]any, len(expectedResults))
	for i, v := range expectedResults {
		fakeDataForMock[i] = v
	}

	// Act
	onPipeFindFunc := NewOnPipeFindMock(fakeDataForMock...)
	cursor, err := onPipeFindFunc(context.Background(), "users", nil)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, cursor)

	// Decode into a variable of the concrete type.
	var decodedResults []map[string]any
	err = cursor.All(context.Background(), &decodedResults)
	require.NoError(t, err)

	// Assert that the decoded results match the expected concrete type results.
	assert.Equal(t, expectedResults, decodedResults)
}

func TestNewOnBulkOperationMock(t *testing.T) {
	t.Run("Success Case", func(t *testing.T) {
		// Arrange
		expectedResult := &mongo.BulkWriteResult{InsertedCount: 1}
		onBulkFunc := NewOnBulkOperationMock(expectedResult, nil)

		// Act
		operator := onBulkFunc("users")
		// Check for chainability
		chainedOp := operator.InsertOne(&testUser{}).UpdateOne(nil, nil)
		result, err := chainedOp.Execute(context.Background())

		// Assert
		require.NoError(t, err)
		assert.Same(t, operator, chainedOp, "Chainable methods should return the same operator")
		assert.Equal(t, expectedResult, result)
	})

	t.Run("Error Case", func(t *testing.T) {
		// Arrange
		expectedErr := errors.New("bulk write failed")
		onBulkFunc := NewOnBulkOperationMock(nil, expectedErr)

		// Act
		operator := onBulkFunc("users")
		result, err := operator.Execute(context.Background())

		// Assert
		assert.Nil(t, result)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}
