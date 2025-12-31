// Package mgo provides a high-level abstraction layer over the official MongoDB Go driver,
// simplifying connection management, document operations, and schema definitions.
package mgo

import "reflect"

type DocInterOpt func(d DocInter)

func WithId(id any) DocInterOpt {
	return func(d DocInter) {
		d.SetId(id)
	}
}

// DocInter defines a standard interface for all database documents.
// By embedding the Index interface, it requires all documents to specify
// their collection name, index definitions, and validation logic.
// This promotes a consistent and self-describing data model.
type DocInter interface {
	Index
	// Validate performs business logic validation on the document's fields.
	Validate() error
	GetId() any
	SetId(any)
}

type DocWithInitIndex interface {
	InitIndex()
}

func NewEmptyModel[T DocWithInitIndex]() T {
	var t T

	val := reflect.New(reflect.TypeOf(t)).Elem()

	// If the type is a pointer, we need to get the underlying element.
	if val.Kind() == reflect.Ptr {
		val.Set(reflect.New(val.Type().Elem()))
	} else {
		panic("NewEmptyModel: type is not a pointer")
	}

	// Type-assert the reflected value back to the generic type T.
	model, ok := val.Interface().(T)
	if !ok {
		panic("type assert fail")
	}

	// Now you can safely call the method.
	model.InitIndex()

	return model
}

type DocWithId interface {
	DocWithInitIndex
	NewId()
	DocInter
}

func NewModelWithId[T DocWithId](opts ...DocInterOpt) T {
	model := NewEmptyModel[T]()
	for _, opt := range opts {
		opt(model)
	}
	if model.GetId() == nil {
		model.NewId()
	}
	return model
}
