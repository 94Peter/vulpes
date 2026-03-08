package validate

import (
	"sync"

	validator "github.com/go-playground/validator/v10"
)

var (
	v    *validator.Validate
	once sync.Once
)

// Get returns a global singleton validator instance.
func Get() *validator.Validate {
	once.Do(func() {
		v = validator.New()
	})
	return v
}

// Struct validates a struct and returns an error if validation fails.
func Struct(s any) error {
	return Get().Struct(s)
}
