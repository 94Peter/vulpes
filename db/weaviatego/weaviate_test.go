package weaviatego

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestInitClient(t *testing.T) {
	// test invalid host
	t.Run("InvalidHost", func(t *testing.T) {
		_, err := InitClient(context.Background(), "invalid-host", "test")
		assert.Error(t, err)
	})
}

type TestData struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (d *TestData) ClassName() string {
	return "Test"
}

func (d *TestData) ID() uuid.UUID {
	return NewUUIDFromString(d.Name)
}

func TestGenerateUUIDFromString(t *testing.T) {
	t.Run("GenerateUUIDFromString", func(t *testing.T) {
		uuid := NewUUIDFromString("test")
		uuid2 := NewUUIDFromString("test")
		uuid3 := NewUUIDFromString("test2")
		assert.NotEqual(t, uuid, uuid3)
		assert.Equal(t, uuid, uuid2)
	})
}

func TestInitClientAndSave(t *testing.T) {
	t.Run("InvalidHost", func(t *testing.T) {
		testClass := NewModelsClassBuilder("Test", "This is just test calss").AddProperty("name", "string", "The name of the test").AddProperty("age", "int", "The age of the test").Apply()
		AddModelsClass(testClass)
		sdk, err := InitClient(context.Background(), "bqlmpphnta62iw0p8zdcjw.c0.asia-southeast1.gcp.weaviate.cloud", "bWtaUzRraVRacS9hVlBWdl8zdnhjMWl5bmpqNkEva1FPSjV0Vm9TcHhnak1qSWtZb2pmNDF3M0pBQko0PV92MjAw")
		if err != nil {
			t.Fatal(err)
		}
		sdk.FindRelated(context.Background(), "Curse", "畢氏定理")
		assert.False(t, true)
		// sdk.CreateOrUpdateData(context.Background(), &TestData{Name: "test", Age: 10})
	})
}
