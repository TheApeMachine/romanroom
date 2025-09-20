package main

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFileVectorStore(t *testing.T) {
	Convey("Given a FileVectorStore", t, func() {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "vectors.json")
		store := NewFileVectorStore(filePath)
		ctx := context.Background()
		
		Convey("When storing a vector", func() {
			embedding := []float32{0.1, 0.2, 0.3}
			metadata := map[string]interface{}{"type": "test"}
			
			err := store.Store(ctx, "test1", embedding, metadata)
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
			
			Convey("And the vector should be retrievable", func() {
				result, err := store.GetByID(ctx, "test1")
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(result.ID, ShouldEqual, "test1")
				So(result.Embedding, ShouldResemble, embedding)
				So(result.Metadata["type"], ShouldEqual, "test")
			})
		})
		
		Convey("When searching for similar vectors", func() {
			// Store test vectors
			vectors := []struct {
				id        string
				embedding []float32
				metadata  map[string]interface{}
			}{
				{"vec1", []float32{1.0, 0.0, 0.0}, map[string]interface{}{"category": "A"}},
				{"vec2", []float32{0.0, 1.0, 0.0}, map[string]interface{}{"category": "B"}},
				{"vec3", []float32{0.9, 0.1, 0.0}, map[string]interface{}{"category": "A"}},
			}
			
			for _, v := range vectors {
				err := store.Store(ctx, v.id, v.embedding, v.metadata)
				So(err, ShouldBeNil)
			}
			
			query := []float32{1.0, 0.0, 0.0}
			results, err := store.Search(ctx, query, 2, nil)
			
			Convey("Then it should return similar vectors", func() {
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 2)
				So(results[0].ID, ShouldEqual, "vec1") // Most similar
				So(results[0].Score, ShouldBeGreaterThan, results[1].Score)
			})
			
			Convey("And filtering should work", func() {
				filters := map[string]interface{}{"category": "A"}
				results, err := store.Search(ctx, query, 10, filters)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 2) // Only category A vectors
				for _, result := range results {
					So(result.Metadata["category"], ShouldEqual, "A")
				}
			})
		})
		
		Convey("When batch storing vectors", func() {
			items := []VectorStoreItem{
				{"batch1", []float32{0.1, 0.2}, map[string]interface{}{"batch": true}},
				{"batch2", []float32{0.3, 0.4}, map[string]interface{}{"batch": true}},
			}
			
			err := store.BatchStore(ctx, items)
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
			
			Convey("And all vectors should be stored", func() {
				count, err := store.Count(ctx)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 2)
			})
		})
		
		Convey("When updating vector metadata", func() {
			embedding := []float32{0.5, 0.5}
			originalMetadata := map[string]interface{}{"version": 1}
			updatedMetadata := map[string]interface{}{"version": 2, "updated": true}
			
			err := store.Store(ctx, "update_test", embedding, originalMetadata)
			So(err, ShouldBeNil)
			
			err = store.Update(ctx, "update_test", updatedMetadata)
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
			
			Convey("And the metadata should be updated", func() {
				result, err := store.GetByID(ctx, "update_test")
				So(err, ShouldBeNil)
				So(result.Metadata["version"], ShouldEqual, 2)
				So(result.Metadata["updated"], ShouldEqual, true)
			})
		})
		
		Convey("When deleting a vector", func() {
			embedding := []float32{0.7, 0.8}
			err := store.Store(ctx, "delete_test", embedding, nil)
			So(err, ShouldBeNil)
			
			err = store.Delete(ctx, "delete_test")
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
			
			Convey("And the vector should not be found", func() {
				_, err := store.GetByID(ctx, "delete_test")
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("When checking health", func() {
			err := store.Health(ctx)
			
			Convey("Then it should be healthy", func() {
				So(err, ShouldBeNil)
			})
		})
		
		Convey("When closing the store", func() {
			err := store.Close()
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
			
			Convey("And subsequent operations should fail", func() {
				err := store.Store(ctx, "test", []float32{1.0}, nil)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "closed")
			})
		})
		
		Convey("When loading and saving", func() {
			// Store some data
			embedding := []float32{0.1, 0.2, 0.3}
			metadata := map[string]interface{}{"persistent": true}
			err := store.Store(ctx, "persist_test", embedding, metadata)
			So(err, ShouldBeNil)
			
			// Save explicitly
			err = store.Save()
			So(err, ShouldBeNil)
			
			// Create new store instance and load
			newStore := NewFileVectorStore(filePath)
			err = newStore.Load()
			
			Convey("Then data should persist", func() {
				So(err, ShouldBeNil)
				
				result, err := newStore.GetByID(ctx, "persist_test")
				So(err, ShouldBeNil)
				So(result.ID, ShouldEqual, "persist_test")
				So(result.Embedding, ShouldResemble, embedding)
				So(result.Metadata["persistent"], ShouldEqual, true)
			})
		})
	})
}

func TestFileVectorStoreEdgeCases(t *testing.T) {
	Convey("Given a FileVectorStore with edge cases", t, func() {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "vectors_edge.json")
		store := NewFileVectorStore(filePath)
		ctx := context.Background()
		
		Convey("When getting a non-existent vector", func() {
			result, err := store.GetByID(ctx, "nonexistent")
			
			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
			})
		})
		
		Convey("When deleting a non-existent vector", func() {
			err := store.Delete(ctx, "nonexistent")
			
			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("When updating a non-existent vector", func() {
			err := store.Update(ctx, "nonexistent", map[string]interface{}{"test": true})
			
			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("When searching with empty query", func() {
			results, err := store.Search(ctx, []float32{}, 10, nil)
			
			Convey("Then it should return empty results", func() {
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 0)
			})
		})
		
		Convey("When loading from non-existent file", func() {
			nonExistentStore := NewFileVectorStore(filepath.Join(tempDir, "nonexistent.json"))
			err := nonExistentStore.Load()
			
			Convey("Then it should succeed with empty store", func() {
				So(err, ShouldBeNil)
				count, err := nonExistentStore.Count(ctx)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 0)
			})
		})
		
		Convey("When storing with nil metadata", func() {
			err := store.Store(ctx, "nil_meta", []float32{1.0}, nil)
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
				
				result, err := store.GetByID(ctx, "nil_meta")
				So(err, ShouldBeNil)
				So(result.Metadata, ShouldNotBeNil)
			})
		})
	})
}

func BenchmarkFileVectorStore(b *testing.B) {
	tempDir := b.TempDir()
	filePath := filepath.Join(tempDir, "bench_vectors.json")
	store := NewFileVectorStore(filePath)
	ctx := context.Background()
	
	// Prepare test data
	embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	metadata := map[string]interface{}{"benchmark": true}
	
	b.Run("Store", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			id := fmt.Sprintf("bench_%d", i)
			store.Store(ctx, id, embedding, metadata)
		}
	})
	
	// Store some vectors for search benchmark
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("search_bench_%d", i)
		store.Store(ctx, id, embedding, metadata)
	}
	
	b.Run("Search", func(b *testing.B) {
		query := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
		for i := 0; i < b.N; i++ {
			store.Search(ctx, query, 10, nil)
		}
	})
	
	b.Run("GetByID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			id := fmt.Sprintf("search_bench_%d", i%1000)
			store.GetByID(ctx, id)
		}
	})
}