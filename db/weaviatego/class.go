package weaviatego

import (
	"context"
	"fmt"

	"github.com/weaviate/weaviate/entities/models"
)

type ModelsClassBuilder interface {
	AddProperty(name, dataType, description string) ModelsClassBuilder
	Apply() *models.Class
}

type modelsClassBuilder struct {
	class *models.Class
}

func NewModelsClassBuilder(name, description string) ModelsClassBuilder {
	return &modelsClassBuilder{
		class: &models.Class{
			Class:       name,
			Description: description,
			VectorConfig: map[string]models.VectorConfig{
				"data_vector": {
					VectorIndexType: "hnsw",
					Vectorizer: map[string]any{
						"text2vec-weaviate": map[string]any{
							"model":      "Snowflake/snowflake-arctic-embed-m-v1.5",
							"dimensions": 256,
						},
					},
				},
			},
		},
	}
}

func (b *modelsClassBuilder) AddProperty(name, dataType, description string) ModelsClassBuilder {
	b.class.Properties = append(b.class.Properties, &models.Property{
		Name:        name,
		DataType:    []string{dataType},
		Description: description,
	})
	return b
}

func (b *modelsClassBuilder) Apply() *models.Class {
	return b.class
}

func (b *weaviateSdk) CreateClassIfNotExists(ctx context.Context, class *models.Class) error {
	if b.clt == nil {
		return fmt.Errorf("weaviate client is not initialized")
	}
	isExist, err := b.ClassExistenceChecker(ctx, class.Class)
	if err != nil {
		return err
	}
	if !isExist {
		err = b.ClassCreator(ctx, class)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sdk *weaviateSdk) ClassExistenceChecker(ctx context.Context, className string) (bool, error) {
	if sdk.clt == nil {
		return false, fmt.Errorf("weaviate client is not initialized")
	}
	return sdk.clt.Schema().ClassExistenceChecker().WithClassName(className).Do(ctx)
}

func (sdk *weaviateSdk) ClassCreator(ctx context.Context, class *models.Class) error {
	if sdk.clt == nil {
		return fmt.Errorf("weaviate client is not initialized")
	}
	return sdk.clt.Schema().ClassCreator().WithClass(class).Do(ctx)
}
