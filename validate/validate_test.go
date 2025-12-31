package validate

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStruct is a helper struct for testing the validation rules.
// It uses standard `validate` tags.
type testStruct struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"gte=0,lte=130"`
}

func TestStruct(t *testing.T) {
	t.Run("ValidStruct", func(t *testing.T) {
		// This struct has all required fields and valid data.
		valid := &testStruct{
			Name:  "John Doe",
			Email: "john.doe@example.com",
			Age:   30,
		}

		err := Struct(valid)
		// We expect no error for a valid struct.
		assert.NoError(t, err)
	})

	t.Run("InvalidStruct_MissingRequiredField", func(t *testing.T) {
		// This struct is missing the required 'Name' field.
		invalid := &testStruct{
			Email: "jane.doe@example.com",
			Age:   25,
		}

		err := Struct(invalid)
		// We expect an error because a required field is missing.
		require.Error(t, err)

		// Optionally, we can assert that the error is of the expected type.
		var validationErrors validator.ValidationErrors
		ok := errors.As(err, &validationErrors)
		assert.True(t, ok, "error should be of type validator.ValidationErrors")
	})

	t.Run("InvalidStruct_InvalidEmail", func(t *testing.T) {
		// This struct has an invalid email format.
		invalid := &testStruct{
			Name:  "Jane Doe",
			Email: "not-an-email",
			Age:   28,
		}

		err := Struct(invalid)
		// We expect an error because the email format is invalid.
		require.Error(t, err)
	})

	t.Run("NilValue", func(t *testing.T) {
		// Nil value should return an error.
		err := Struct(nil)
		require.Error(t, err)
	})
}
