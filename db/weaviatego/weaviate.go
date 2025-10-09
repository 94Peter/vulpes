package weaviatego

import (
	"context"
	"fmt"

	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/auth"
	"github.com/weaviate/weaviate/entities/models"
)

type SDK interface {
	ClassExistenceChecker(ctx context.Context, className string) (bool, error)
	ClassCreator(ctx context.Context, class *models.Class) error
	CreateClassIfNotExists(ctx context.Context, class *models.Class) error
	CreateData(ctx context.Context, data Data) error
	CreateOrUpdateData(ctx context.Context, data Data) error
	querySDK
}

var sdk SDK
var allClass []*models.Class

type weaviateSdk struct {
	clt *weaviate.Client
}

func AddModelsClass(class *models.Class) {
	allClass = append(allClass, class)
}

func InitClient(ctx context.Context, host, apiKey string) (SDK, error) {
	if sdk != nil {
		return sdk, nil
	}
	cfg := weaviate.Config{
		Host:       host,
		Scheme:     "https",
		AuthConfig: auth.ApiKey{Value: apiKey},
	}
	var err error
	clt, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create weaviate client: %w", err)
	}
	// Check the connection
	live, err := clt.Misc().LiveChecker().Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to check weaviate live: %w", err)
	}

	if !live {
		return nil, fmt.Errorf("weaviate is not live")
	}
	sdk = &weaviateSdk{
		clt: clt,
	}
	for _, class := range allClass {
		err = sdk.CreateClassIfNotExists(ctx, class)
		if err != nil {
			return nil, err
		}
	}
	return sdk, nil
}
