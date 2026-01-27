package mgo_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/94peter/vulpes/db/mgo"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type SubStruct struct {
	Field1 string `bson:"field1"`
	Field2 int    `bson:"field2"`
	Field3 bool   `bson:"field3"`
}

type NestedLevel3 struct {
	Value bool `bson:"value"`
}

type NestedLevel2 struct {
	Value int           `bson:"value"`
	Next  *NestedLevel3 `bson:"next"`
}

type NestedLevel1 struct {
	Value string        `bson:"value"`
	Next  *NestedLevel2 `bson:"next"`
}

type ComplexStruct struct {
	ID       bson.ObjectID     `bson:"_id,omitempty"`
	Name     string            `bson:"name"`
	Age      int               `bson:"age"`
	Balance  float64           `bson:"balance"`
	Active   bool              `bson:"active"`
	Tags     []string          `bson:"tags"`
	Scores   []int             `bson:"scores"`
	Metadata map[string]string `bson:"metadata"`
}

func (c *ComplexStruct) UnmarshalBSON(data []byte) error {
	raw := bson.Raw(data)
	elements, err := raw.Elements()
	if err != nil {
		return err
	}

	for _, el := range elements {
		switch el.Key() {
		case "_id":
			c.ID = el.Value().ObjectID()
		case "name":
			c.Name = el.Value().StringValue()
		case "active":
			c.Active = el.Value().Boolean()
		case "age":
			c.Age = int(el.Value().Int32())
		case "tags":
			if arrRaw, ok := el.Value().ArrayOK(); ok {
				vals, err := arrRaw.Values()
				if err != nil {
					return err
				}

				// 2. 根據長度預分配或重置 Tags
				if cap(c.Tags) >= len(vals) {
					c.Tags = c.Tags[:0]
				} else {
					c.Tags = make([]string, 0, len(vals))
				}

				// 3. 現在可以直接使用 range 了！
				for _, v := range vals {
					// StringValue() 依然是分配字串的主要來源
					c.Tags = append(c.Tags, v.StringValue())
				}
			}
		case "scores":
			if arrRaw, ok := el.Value().ArrayOK(); ok {
				vals, err := arrRaw.Values()
				if err != nil {
					return err
				}

				// 2. 根據長度預分配或重置 Scores
				if cap(c.Scores) >= len(vals) {
					c.Scores = c.Scores[:0]
				} else {
					c.Scores = make([]int, 0, len(vals))
				}

				// 3. 現在可以直接使用 range 了！
				for _, v := range vals {
					c.Scores = append(c.Scores, int(v.Int32()))
				}
			}
		case "metadata":
			// 優化點：手動解析 Map
			if mRaw, ok := el.Value().DocumentOK(); ok {
				mElements, _ := mRaw.Elements()
				// 根據 BSON 元素數量預分配 Map 容量
				if c.Metadata == nil {
					c.Metadata = make(map[string]string, len(mElements))
				}
				for _, mEl := range mElements {
					c.Metadata[mEl.Key()] = mEl.Value().StringValue()
				}
			}
			// 	// 處理其他欄位...
		}
	}
	return nil
}

func (*ComplexStruct) Validate() error {
	return nil
}

func (c *ComplexStruct) GetId() any {
	return c.ID
}

func (c *ComplexStruct) SetId(id any) {
	c.ID = id.(bson.ObjectID)
}

func (*ComplexStruct) C() string {
	const collectionName = "complex_collection"
	return collectionName
}

func (*ComplexStruct) Indexes() []mongo.IndexModel {
	return nil
}

var setupOnce sync.Once

func BenchmarkPipeFindByPipeline(b *testing.B) {
	ctx := b.Context()
	setupOnce.Do(func() {
		cleanDb()
		// Prepare data
		count := 100
		for i := range count {
			doc := &ComplexStruct{
				ID:      bson.NewObjectID(),
				Name:    fmt.Sprintf("User %d", i),
				Age:     20 + (i % 50),
				Balance: 1000.50 * float64(i),
				Active:  i%2 == 0,
				Tags:    []string{"tag1", "tag2", "tag3", "tag4", "tag5"},
				Scores:  []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
				Metadata: map[string]string{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
				},
			}
			_, err := mgo.Save(ctx, doc)
			if err != nil {
				b.Fatalf("failed to save doc: %v", err)
			}
		}
	})
	collectionName := "complex_collection"
	_ = collectionName
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "active", Value: true}}}},
	}

	b.ResetTimer()

	for b.Loop() {
		results, err := mgo.PipeFindByPipeline[ComplexStruct](ctx, collectionName, pipeline, 30)
		if err != nil {
			b.Fatal(err)
		}
		if len(results) == 0 {
			b.Fatal("expected results, got 0")
		}
	}
}

// func BenchmarkDecodeComplex(b *testing.B) {
// 	// 1. 先準備好一份 BSON 數據（[]byte）
// 	i := 0
// 	mockLargeComplexStruct := &ComplexStruct{
// 		ID:      bson.NewObjectID(),
// 		Name:    fmt.Sprintf("User %d", i),
// 		Age:     20 + (i % 50),
// 		Balance: 1000.50 * float64(i),
// 		Active:  i%2 == 0,
// 		Tags:    []string{"tag1", "tag2", "tag3", "tag4", "tag5"},
// 		Scores:  []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
// 		Metadata: map[string]string{
// 			"key1": "value1",
// 			"key2": "value2",
// 			"key3": "value3",
// 		},
// 		SubDoc: SubStruct{
// 			Field1: "subfield",
// 			Field2: i,
// 			Field3: true,
// 		},
// 		SubDocs: []SubStruct{
// 			{Field1: "s1", Field2: 1, Field3: true},
// 			{Field1: "s2", Field2: 2, Field3: false},
// 			{Field1: "s3", Field2: 3, Field3: true},
// 		},
// 		DeepNest: &NestedLevel1{
// 			Value: "level1",
// 			Next: &NestedLevel2{
// 				Value: 2,
// 				Next: &NestedLevel3{
// 					Value: true,
// 				},
// 			},
// 		},
// 		CreatedAt: time.Now(),
// 	}

// 	data, _ := bson.Marshal(mockLargeComplexStruct)
// 	b.ResetTimer() // 重置時間，排除準備數據的時間

// 	for i := 0; i < b.N; i++ {
// 		var res ComplexStruct
// 		bson.Unmarshal(data, &res)
// 	}
// }
