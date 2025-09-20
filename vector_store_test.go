package main

import (
	"context"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestVectorStoreInterface(t *testing.T) {
	Convey("VectorStore Interface Contract", t, func() {
		ctx := context.Background()
		store := NewMockVectorStore()
		
		Convey("Store operation", func() {
			embedding := []float32{0.1, 0.2, 0.3, 0.4}
			metadata := map[string]interface{}{
				"content": "test content",
				"source":  "test",
			}
			
			err := store.Store(ctx, "test-id", embedding, metadata)
			So(err, ShouldBeNil)
			
			Convey("Should be retrievable by ID", func() {
				result, err := store.GetByID(ctx, "test-id")
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(result.ID, ShouldEqual, "test-id")
				So(result.Embedding, ShouldResemble, embedding)
				So(result.Metadata["content"], ShouldEqual, "test content")
			})
		})
		
		Convey("Search operation", func() {
			// Store test vectors
			vectors := []struct {
				id        string
				embedding []float32
				metadata  map[string]interface{}
			}{
				{"vec1", []float32{1.0, 0.0, 0.0}, map[string]interface{}{"category": "A"}},
				{"vec2", []float32{0.0, 1.0, 0.0}, map[string]interface{}{"category": "B"}},
				{"vec3", []float32{0.0, 0.0, 1.0}, map[string]interface{}{"category": "A"}},
			}
			
			for _, v := range vectors {
				err := store.Store(ctx, v.id, v.embedding, v.metadata)
				So(err, ShouldBeNil)
			}
			
			Convey("Should return similar vectors", func() {
				query := []float32{0.9, 0.1, 0.0}
				results, err := store.Search(ctx, query, 2, nil)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 2)
				So(results[0].ID, ShouldEqual, "vec1") // Most similar
			})
			
			Convey("Should apply filters", func() {
				query := []float32{0.5, 0.5, 0.5}
				filters := map[string]interface{}{"category": "A"}
				results, err := store.Search(ctx, query, 10, filters)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 2) // Only vec1 and vec3
				for _, result := range results {
					So(result.Metadata["category"], ShouldEqual, "A")
				}
			})
		})
		
		Convey("Update operation", func() {
			embedding := []float32{0.1, 0.2, 0.3}
			metadata := map[string]interface{}{"version": 1}
			
			err := store.Store(ctx, "update-test", embedding, metadata)
			So(err, ShouldBeNil)
			
			newMetadata := map[string]interface{}{"version": 2, "updated": true}
			err = store.Update(ctx, "update-test", newMetadata)
			So(err, ShouldBeNil)
			
			result, err := store.GetByID(ctx, "update-test")
			So(err, ShouldBeNil)
			So(result.Metadata["version"], ShouldEqual, 2)
			So(result.Metadata["updated"], ShouldEqual, true)
		})
		
		Convey("Delete operation", func() {
			embedding := []float32{0.1, 0.2, 0.3}
			metadata := map[string]interface{}{"temp": true}
			
			err := store.Store(ctx, "delete-test", embedding, metadata)
			So(err, ShouldBeNil)
			
			err = store.Delete(ctx, "delete-test")
			So(err, ShouldBeNil)
			
			_, err = store.GetByID(ctx, "delete-test")
			So(err, ShouldNotBeNil)
		})
		
		Convey("Batch operations", func() {
			vectors := []VectorStoreItem{
				{"batch1", []float32{1.0, 0.0}, map[string]interface{}{"batch": true}},
				{"batch2", []float32{0.0, 1.0}, map[string]interface{}{"batch": true}},
			}
			
			err := store.BatchStore(ctx, vectors)
			So(err, ShouldBeNil)
			
			for _, v := range vectors {
				result, err := store.GetByID(ctx, v.ID)
				So(err, ShouldBeNil)
				So(result.Metadata["batch"], ShouldEqual, true)
			}
		})
		
		Convey("Count operation", func() {
			initialCount, err := store.Count(ctx)
			So(err, ShouldBeNil)
			
			err = store.Store(ctx, "count-test", []float32{1.0}, map[string]interface{}{})
			So(err, ShouldBeNil)
			
			newCount, err := store.Count(ctx)
			So(err, ShouldBeNil)
			So(newCount, ShouldEqual, initialCount+1)
		})
		
		Convey("Health check", func() {
			err := store.Health(ctx)
			So(err, ShouldBeNil)
		})
		
		Convey("Close operation", func() {
			err := store.Close()
			So(err, ShouldBeNil)
			
			// Operations should fail after close
			err = store.Store(ctx, "after-close", []float32{1.0}, map[string]interface{}{})
			So(err, ShouldNotBeNil)
		})
	})
}

func TestVectorStoreErrorHandling(t *testing.T) {
	Convey("VectorStore Error Handling", t, func() {
		ctx := context.Background()
		store := NewMockVectorStore()
		
		Convey("Should handle non-existent vector", func() {
			_, err := store.GetByID(ctx, "non-existent")
			So(err, ShouldNotBeNil)
		})
		
		Convey("Should handle update of non-existent vector", func() {
			err := store.Update(ctx, "non-existent", map[string]interface{}{})
			So(err, ShouldNotBeNil)
		})
		
		Convey("Should handle delete of non-existent vector", func() {
			err := store.Delete(ctx, "non-existent")
			So(err, ShouldBeNil) // Delete is idempotent
		})
		
		Convey("Should handle unhealthy state", func() {
			store.SetHealthy(false)
			
			err := store.Health(ctx)
			So(err, ShouldNotBeNil)
		})
	})
}

func BenchmarkVectorStore(b *testing.B) {
	ctx := context.Background()
	store := NewMockVectorStore()
	
	// Prepare test data
	embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	metadata := map[string]interface{}{"benchmark": true}
	
	b.Run("Store", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			store.Store(ctx, fmt.Sprintf("bench-%d", i), embedding, metadata)
		}
	})
	
	// Store some vectors for search benchmarks
	for i := 0; i < 1000; i++ {
		store.Store(ctx, fmt.Sprintf("search-bench-%d", i), embedding, metadata)
	}
	
	b.Run("Search", func(b *testing.B) {
		query := []float32{0.2, 0.3, 0.4, 0.5, 0.6}
		for i := 0; i < b.N; i++ {
			store.Search(ctx, query, 10, nil)
		}
	})
	
	b.Run("GetByID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			store.GetByID(ctx, fmt.Sprintf("search-bench-%d", i%1000))
		}
	})
}