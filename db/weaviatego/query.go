package weaviatego

import (
	"context"
	"fmt"

	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
)

const (
	defaultBiasAlpha  = 0.2
	defaultQueryLimit = 3
)

type querySDK interface {
	Query(ctx context.Context, className string) error
	FindRelated(ctx context.Context, className string, relatedText string) error
}

func (sdk *weaviateSdk) FindRelated(_ context.Context, className string, relatedText string) error {
	concepts := []string{relatedText}

	// 2. 執行 NearText 向量相似度搜尋
	result, err := sdk.clt.GraphQL().Get().
		WithClassName(className). // 假設課程資料 class 名為 Course
		WithFields(
			// 選擇您想返回的課程欄位
			graphql.Field{Name: "course_id"},
			graphql.Field{Name: "course_name"},
			graphql.Field{Name: "knowledge_coverage"},
			graphql.Field{Name: "_additional", Fields: []graphql.Field{
				{Name: "distance"}, // 取得向量距離，距離越小越相似
			}},
		).
		WithNearText(sdk.clt.GraphQL().NearTextArgBuilder().
			WithConcepts(concepts),
		).
		WithLimit(defaultQueryLimit). // 只返回最相關的前 3 個課程
		Do(context.Background())

	if err != nil {
		return err
	}
	if len(result.Errors) > 0 {
		fmt.Println(result.Errors[0].Message)
	}
	fmt.Println(result.Data)
	return nil
}

func (sdk *weaviateSdk) Query(ctx context.Context, className string) error {
	// weaknessPoints := []string{"畢氏定理", "JHC02"}
	result, err := sdk.clt.GraphQL().Get().WithClassName(className).
		WithFields(graphql.Field{Name: "course_name"}, graphql.Field{Name: "knowledge_coverage"}).
		WithHybrid(
			sdk.clt.GraphQL().HybridArgumentBuilder().
				WithAlpha(defaultBiasAlpha).
				WithQuery("文言文 代數").WithProperties([]string{"course_name", "knowledge_coverage"}),
		).
		WithLimit(defaultQueryLimit).Do(ctx)
	if err != nil {
		return err
	}
	if len(result.Errors) > 0 {
		fmt.Println(result.Errors[0].Message)
	}
	fmt.Println(result.Data)

	// fmt.Println(result.Data, err)
	return nil
}
