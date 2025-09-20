package main

import (
	"context"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMockVectorStore(t *testing.T) {
	Convey("MockVectorStore Implementation", t, func() {
		ctx := context.Background()
		store := NewMockVectorStore()
		
		Convey("Should implement VectorStore interface", func() {
			var _ VectorStore = store
		})
		
		Convey("Store and retrieve operations", func() {
			embedding := []float32{0.1, 0.2, 0.3, 0.4}
			metadata := map[string]interface{}{
				"content": "test content",
				"source":  "test",
			}
			
			err := store.Store(ctx, "test-vector", embedding, metadata)
			So(err, ShouldBeNil)
			
			result, err := store.GetByID(ctx, "test-vector")
			So(err, ShouldBeNil)
			So(result.ID, ShouldEqual, "test-vector")
			So(result.Embedding, ShouldResemble, embedding)
			So(result.Metadata["content"], ShouldEqual, "test content")
			So(result.Score, ShouldEqual, 1.0) // Perfect match for GetByID
		})
		
		Convey("Cosine similarity calculation", func() {
			// Store vectors with known similarities
			store.Store(ctx, "vec1", []float32{1.0, 0.0, 0.0}, map[string]interface{}{})
			store.Store(ctx, "vec2", []float32{0.0, 1.0, 0.0}, map[string]interface{}{})
			store.Store(ctx, "vec3", []float32{1.0, 0.0, 0.0}, map[string]interface{}{}) // Same as vec1
			
			// Query with vector similar to vec1 and vec3
			query := []float32{0.9, 0.1, 0.0}
			results, err := store.Search(ctx, query, 3, nil)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 3)
			
			// Results should be ordered by similarity (vec1/vec3 should be more similar than vec2)
			So(results[0].Score, ShouldBeGreaterThan, results[2].Score)
		})
		
		Convey("Filter functionality", func() {
			// Store vectors with different metadata
			store.Store(ctx, "cat1", []float32{1.0, 0.0}, map[string]interface{}{"category": "A"})
			store.Store(ctx, "cat2", []float32{0.0, 1.0}, map[string]interface{}{"category": "B"})
			store.Store(ctx, "cat3", []float32{0.5, 0.5}, map[string]interface{}{"category": "A"})
			
			// Search with filter
			filters := map[string]interface{}{"category": "A"}
			results, err := store.Search(ctx, []float32{0.8, 0.2}, 10, filters)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 2) // Only cat1 and cat3
			
			for _, result := range results {
				So(result.Metadata["category"], ShouldEqual, "A")
			}
		})
		
		Convey("Update functionality", func() {
			embedding := []float32{0.1, 0.2, 0.3}
			metadata := map[string]interface{}{"version": 1}
			
			store.Store(ctx, "update-test", embedding, metadata)
			
			newMetadata := map[string]interface{}{"version": 2, "updated": true}
			err := store.Update(ctx, "update-test", newMetadata)
			So(err, ShouldBeNil)
			
			result, err := store.GetByID(ctx, "update-test")
			So(err, ShouldBeNil)
			So(result.Metadata["version"], ShouldEqual, 2)
			So(result.Metadata["updated"], ShouldEqual, true)
			So(result.Embedding, ShouldResemble, embedding) // Embedding unchanged
		})
		
		Convey("Batch operations", func() {
			vectors := []VectorStoreItem{
				{"batch1", []float32{1.0, 0.0}, map[string]interface{}{"batch": true}},
				{"batch2", []float32{0.0, 1.0}, map[string]interface{}{"batch": true}},
				{"batch3", []float32{0.5, 0.5}, map[string]interface{}{"batch": true}},
			}
			
			err := store.BatchStore(ctx, vectors)
			So(err, ShouldBeNil)
			
			// Verify all vectors were stored
			for _, v := range vectors {
				result, err := store.GetByID(ctx, v.ID)
				So(err, ShouldBeNil)
				So(result.Embedding, ShouldResemble, v.Embedding)
				So(result.Metadata["batch"], ShouldEqual, true)
			}
		})
		
		Convey("Delete functionality", func() {
			store.Store(ctx, "delete-me", []float32{1.0}, map[string]interface{}{})
			
			// Verify it exists
			_, err := store.GetByID(ctx, "delete-me")
			So(err, ShouldBeNil)
			
			// Delete it
			err = store.Delete(ctx, "delete-me")
			So(err, ShouldBeNil)
			
			// Verify it's gone
			_, err = store.GetByID(ctx, "delete-me")
			So(err, ShouldNotBeNil)
		})
		
		Convey("Count functionality", func() {
			initialCount, err := store.Count(ctx)
			So(err, ShouldBeNil)
			
			// Add some vectors
			for i := 0; i < 5; i++ {
				store.Store(ctx, fmt.Sprintf("count-test-%d", i), []float32{float32(i)}, map[string]interface{}{})
			}
			
			newCount, err := store.Count(ctx)
			So(err, ShouldBeNil)
			So(newCount, ShouldEqual, initialCount+5)
		})
		
		Convey("Health status management", func() {
			// Initially healthy
			err := store.Health(ctx)
			So(err, ShouldBeNil)
			
			// Set unhealthy
			store.SetHealthy(false)
			err = store.Health(ctx)
			So(err, ShouldNotBeNil)
			
			// Set healthy again
			store.SetHealthy(true)
			err = store.Health(ctx)
			So(err, ShouldBeNil)
		})
		
		Convey("Close functionality", func() {
			err := store.Close()
			So(err, ShouldBeNil)
			
			// All operations should fail after close
			err = store.Store(ctx, "after-close", []float32{1.0}, map[string]interface{}{})
			So(err, ShouldNotBeNil)
			
			_, err = store.Search(ctx, []float32{1.0}, 1, nil)
			So(err, ShouldNotBeNil)
			
			_, err = store.GetByID(ctx, "any-id")
			So(err, ShouldNotBeNil)
			
			err = store.Delete(ctx, "any-id")
			So(err, ShouldNotBeNil)
			
			_, err = store.Count(ctx)
			So(err, ShouldNotBeNil)
			
			err = store.Health(ctx)
			So(err, ShouldNotBeNil)
		})
		
		Convey("Edge cases", func() {
			Convey("Empty embedding vectors", func() {
				err := store.Store(ctx, "empty", []float32{}, map[string]interface{}{})
				So(err, ShouldBeNil)
				
				results, err := store.Search(ctx, []float32{}, 1, nil)
				So(err, ShouldBeNil)
				So(len(results), ShouldBeGreaterThanOrEqualTo, 0)
			})
			
			Convey("Zero vectors", func() {
				store.Store(ctx, "zero1", []float32{0.0, 0.0, 0.0}, map[string]interface{}{})
				store.Store(ctx, "zero2", []float32{0.0, 0.0, 0.0}, map[string]interface{}{})
				
				results, err := store.Search(ctx, []float32{0.0, 0.0, 0.0}, 2, nil)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 2)
				// Cosine similarity with zero vectors should be 0
				for _, result := range results {
					So(result.Score, ShouldEqual, 0.0)
				}
			})
			
			Convey("Mismatched vector dimensions", func() {
				store.Store(ctx, "dim3", []float32{1.0, 0.0, 0.0}, map[string]interface{}{})
				
				// Search with different dimension should return 0 similarity
				results, err := store.Search(ctx, []float32{1.0, 0.0}, 1, nil)
				So(err, ShouldBeNil)
				if len(results) > 0 {
					So(results[0].Score, ShouldEqual, 0.0)
				}
			})
		})
	})
}

func BenchmarkMockVectorStore(b *testing.B) {
	ctx := context.Background()
	store := NewMockVectorStore()
	
	// Prepare test data
	embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	metadata := map[string]interface{}{"benchmark": true}
	
	// Pre-populate for search benchmarks
	for i := 0; i < 1000; i++ {
		store.Store(ctx, fmt.Sprintf("bench-%d", i), embedding, metadata)
	}
	
	b.Run("Store", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			store.Store(ctx, fmt.Sprintf("store-bench-%d", i), embedding, metadata)
		}
	})
	
	b.Run("Search", func(b *testing.B) {
		query := []float32{0.2, 0.3, 0.4, 0.5, 0.6}
		for i := 0; i < b.N; i++ {
			store.Search(ctx, query, 10, nil)
		}
	})
	
	b.Run("GetByID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			store.GetByID(ctx, fmt.Sprintf("bench-%d", i%1000))
		}
	})
	
	b.Run("CosineSimilarity", func(b *testing.B) {
		vec1 := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
		vec2 := []float32{0.2, 0.3, 0.4, 0.5, 0.6}
		
		for i := 0; i < b.N; i++ {
			store.cosineSimilarity(vec1, vec2)
		}
	})
}