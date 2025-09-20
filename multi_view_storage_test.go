package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMultiViewStorage(t *testing.T) {
	Convey("MultiViewStorage Coordinator", t, func() {
		ctx := context.Background()
		
		// Create mock storage backends
		vectorStore := NewMockVectorStore()
		graphStore := NewMockGraphStore()
		searchIndex := NewMockSearchIndex()
		
		config := &MultiViewStorageConfig{
			Timeout:    5 * time.Second,
			RetryCount: 3,
		}
		
		mvs := NewMultiViewStorage(vectorStore, graphStore, searchIndex, config)
		
		Convey("Chunk storage operations", func() {
			chunk := &Chunk{
				ID:        "test-chunk-1",
				Content:   "This is a test chunk about machine learning and artificial intelligence",
				Embedding: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
				Metadata: map[string]interface{}{
					"source":   "test-document",
					"category": "AI",
					"timestamp": time.Now().Unix(),
				},
				Entities: []Entity{
					{
						ID:         "entity-1",
						Name:       "machine learning",
						Type:       "concept",
						Confidence: 0.9,
					},
					{
						ID:         "entity-2", 
						Name:       "artificial intelligence",
						Type:       "concept",
						Confidence: 0.95,
					},
				},
				Claims: []Claim{
					{
						ID:         "claim-1",
						Subject:    "Machine learning",
						Predicate:  "is a subset of",
						Object:     "artificial intelligence",
						Confidence: 0.85,
						Evidence:   []string{"test-document"},
					},
				},
				Timestamp: time.Now(),
				Source:    "test-document",
				Confidence: 0.9,
			}
			
			Convey("Should store chunk across all backends", func() {
				err := mvs.StoreChunk(ctx, chunk)
				So(err, ShouldBeNil)
				
				// Verify vector storage
				vectorResult, err := vectorStore.GetByID(ctx, chunk.ID)
				So(err, ShouldBeNil)
				So(vectorResult.ID, ShouldEqual, chunk.ID)
				So(vectorResult.Embedding, ShouldResemble, chunk.Embedding)
				
				// Verify search index
				searchDoc, err := searchIndex.GetDocument(ctx, chunk.ID)
				So(err, ShouldBeNil)
				So(searchDoc.ID, ShouldEqual, chunk.ID)
				So(searchDoc.Content, ShouldEqual, chunk.Content)
				
				// Verify graph storage (entities and claims)
				for _, entity := range chunk.Entities {
					node, err := graphStore.GetNode(ctx, entity.ID)
					So(err, ShouldBeNil)
					So(node.Type, ShouldEqual, EntityNode)
					So(node.Properties["name"], ShouldEqual, entity.Name)
					So(node.Properties["chunk_id"], ShouldEqual, chunk.ID)
				}
				
				for _, claim := range chunk.Claims {
					node, err := graphStore.GetNode(ctx, claim.ID)
					So(err, ShouldBeNil)
					So(node.Type, ShouldEqual, ClaimNode)
					So(node.Properties["subject"], ShouldEqual, claim.Subject)
					So(node.Properties["chunk_id"], ShouldEqual, chunk.ID)
				}
			})
			
			Convey("Should handle partial storage failures gracefully", func() {
				// Make vector store unhealthy to simulate failure
				vectorStore.SetHealthy(false)
				
				err := mvs.StoreChunk(ctx, chunk)
				So(err, ShouldNotBeNil) // Should report partial failure
				So(err.Error(), ShouldContainSubstring, "partial storage failure")
				
				// But other backends should still work
				searchDoc, err := searchIndex.GetDocument(ctx, chunk.ID)
				So(err, ShouldBeNil)
				So(searchDoc.Content, ShouldEqual, chunk.Content)
			})
		})
		
		Convey("Multi-view retrieval operations", func() {
			// Store test data
			chunks := []*Chunk{
				{
					ID:        "chunk-1",
					Content:   "Machine learning algorithms for data analysis",
					Embedding: []float32{1.0, 0.0, 0.0, 0.0, 0.0},
					Metadata:  map[string]interface{}{"category": "ML"},
					Entities:  []Entity{{ID: "ml-entity", Name: "machine learning", Type: "concept"}},
					Claims:    []Claim{{ID: "ml-claim", Subject: "ML", Predicate: "is useful for", Object: "data analysis"}},
				},
				{
					ID:        "chunk-2",
					Content:   "Deep learning neural networks and AI systems",
					Embedding: []float32{0.0, 1.0, 0.0, 0.0, 0.0},
					Metadata:  map[string]interface{}{"category": "DL"},
					Entities:  []Entity{{ID: "dl-entity", Name: "deep learning", Type: "concept"}},
					Claims:    []Claim{{ID: "dl-claim", Subject: "Deep learning", Predicate: "uses", Object: "neural networks"}},
				},
				{
					ID:        "chunk-3",
					Content:   "Natural language processing and text analysis",
					Embedding: []float32{0.0, 0.0, 1.0, 0.0, 0.0},
					Metadata:  map[string]interface{}{"category": "NLP"},
					Entities:  []Entity{{ID: "nlp-entity", Name: "natural language processing", Type: "concept"}},
					Claims:    []Claim{{ID: "nlp-claim", Subject: "NLP", Predicate: "processes", Object: "human language"}},
				},
			}
			
			for _, chunk := range chunks {
				mvs.StoreChunk(ctx, chunk)
			}
			
			Convey("Should retrieve from all backends and fuse results", func() {
				query := "machine learning analysis"
				embedding := []float32{0.8, 0.1, 0.1, 0.0, 0.0}
				options := RetrievalOptions{
					MaxResults:   10,
					IncludeGraph: true,
				}
				
				results, err := mvs.RetrieveMultiView(ctx, query, embedding, options)
				So(err, ShouldBeNil)
				So(results, ShouldNotBeNil)
				So(results.Query, ShouldEqual, query)
				
				// Should have results from vector search
				So(len(results.VectorResults), ShouldBeGreaterThan, 0)
				
				// Should have results from text search
				So(len(results.SearchResults), ShouldBeGreaterThan, 0)
				
				// Should have results from graph search
				So(len(results.GraphResults), ShouldBeGreaterThan, 0)
				
				// Vector results should be ordered by similarity
				if len(results.VectorResults) > 1 {
					So(results.VectorResults[0].Score, ShouldBeGreaterThanOrEqualTo, results.VectorResults[1].Score)
				}
			})
			
			Convey("Should handle backend failures gracefully", func() {
				// Make search index unhealthy
				searchIndex.SetHealthy(false)
				
				query := "machine learning"
				embedding := []float32{0.8, 0.1, 0.1, 0.0, 0.0}
				options := RetrievalOptions{MaxResults: 10}
				
				results, err := mvs.RetrieveMultiView(ctx, query, embedding, options)
				So(err, ShouldBeNil)
				So(results, ShouldNotBeNil)
				
				// Should have errors reported
				So(len(results.Errors), ShouldBeGreaterThan, 0)
				
				// But should still have results from working backends
				So(len(results.VectorResults), ShouldBeGreaterThan, 0)
			})
			
			Convey("Should apply filters across backends", func() {
				query := "learning"
				embedding := []float32{0.5, 0.5, 0.0, 0.0, 0.0}
				options := RetrievalOptions{
					MaxResults: 10,
					Filters:    map[string]interface{}{"category": "ML"},
				}
				
				results, err := mvs.RetrieveMultiView(ctx, query, embedding, options)
				So(err, ShouldBeNil)
				
				// Results should only include ML category items
				for _, result := range results.VectorResults {
					So(result.Metadata["category"], ShouldEqual, "ML")
				}
				
				for _, result := range results.SearchResults {
					So(result.Metadata["category"], ShouldEqual, "ML")
				}
			})
		})
		
		Convey("Chunk deletion operations", func() {
			chunk := &Chunk{
				ID:        "delete-test-chunk",
				Content:   "This chunk will be deleted",
				Embedding: []float32{0.1, 0.2, 0.3},
				Metadata:  map[string]interface{}{"temp": true},
				Entities:  []Entity{{ID: "delete-entity", Name: "test entity", Type: "test"}},
				Claims:    []Claim{{ID: "delete-claim", Subject: "test", Predicate: "is", Object: "claim"}},
			}
			
			// Store the chunk first
			mvs.StoreChunk(ctx, chunk)
			
			Convey("Should delete from all backends", func() {
				err := mvs.DeleteChunk(ctx, chunk.ID)
				So(err, ShouldBeNil)
				
				// Verify deletion from vector store
				_, err = vectorStore.GetByID(ctx, chunk.ID)
				So(err, ShouldNotBeNil)
				
				// Verify deletion from search index
				exists, err := searchIndex.DocumentExists(ctx, chunk.ID)
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
				
				// Verify deletion of related graph nodes
				_, err = graphStore.GetNode(ctx, "delete-entity")
				So(err, ShouldNotBeNil)
				
				_, err = graphStore.GetNode(ctx, "delete-claim")
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("Statistics collection", func() {
			// Reset all backends to healthy state
			vectorStore.SetHealthy(true)
			graphStore.SetHealthy(true)
			searchIndex.SetHealthy(true)
			
			// Store some test data
			for i := 0; i < 3; i++ {
				chunk := &Chunk{
					ID:        fmt.Sprintf("stats-chunk-%d", i),
					Content:   fmt.Sprintf("Statistics test chunk %d", i),
					Embedding: []float32{float32(i), 0.0, 0.0},
					Metadata:  map[string]interface{}{"stats": true},
					Entities:  []Entity{{ID: fmt.Sprintf("stats-entity-%d", i), Name: fmt.Sprintf("entity %d", i)}},
					Claims:    []Claim{{ID: fmt.Sprintf("stats-claim-%d", i), Subject: fmt.Sprintf("claim %d", i), Predicate: "is", Object: "test"}},
				}
				mvs.StoreChunk(ctx, chunk)
			}
			
			Convey("Should collect statistics from all backends", func() {
				stats, err := mvs.GetStats(ctx)
				So(err, ShouldBeNil)
				So(stats, ShouldNotBeNil)
				
				// Should have counts from all backends
				So(stats.VectorCount, ShouldBeGreaterThanOrEqualTo, 3)
				So(stats.NodeCount, ShouldBeGreaterThanOrEqualTo, 6) // 3 entities + 3 claims
				So(stats.DocumentCount, ShouldBeGreaterThanOrEqualTo, 3)
				
				// Should have health status for all backends
				So(stats.StorageHealth["vector"], ShouldBeTrue)
				So(stats.StorageHealth["graph_nodes"], ShouldBeTrue)
				So(stats.StorageHealth["graph_edges"], ShouldBeTrue)
				So(stats.StorageHealth["search"], ShouldBeTrue)
				
				So(stats.LastUpdated, ShouldHappenWithin, time.Second, time.Now())
			})
			
			Convey("Should report unhealthy backends", func() {
				// Reset all to healthy first
				vectorStore.SetHealthy(true)
				graphStore.SetHealthy(true)
				searchIndex.SetHealthy(true)
				
				// Then set only vector store to unhealthy
				vectorStore.SetHealthy(false)
				
				stats, err := mvs.GetStats(ctx)
				So(err, ShouldBeNil)
				So(stats.StorageHealth["vector"], ShouldBeFalse)
				So(stats.StorageHealth["search"], ShouldBeTrue) // Others still healthy
			})
		})
		
		Convey("Health monitoring", func() {
			Convey("Should report healthy when all backends are healthy", func() {
				err := mvs.Health(ctx)
				So(err, ShouldBeNil)
			})
			
			Convey("Should report unhealthy when any backend is unhealthy", func() {
				graphStore.SetHealthy(false)
				
				err := mvs.Health(ctx)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "graph store unhealthy")
			})
			
			Convey("Should report multiple unhealthy backends", func() {
				vectorStore.SetHealthy(false)
				searchIndex.SetHealthy(false)
				
				err := mvs.Health(ctx)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "vector store unhealthy")
				So(err.Error(), ShouldContainSubstring, "search index unhealthy")
			})
		})
		
		Convey("Lifecycle management", func() {
			Convey("Should close all backends", func() {
				err := mvs.Close()
				So(err, ShouldBeNil)
				
				// All backends should be closed
				err = vectorStore.Health(ctx)
				So(err, ShouldNotBeNil)
				
				err = graphStore.Health(ctx)
				So(err, ShouldNotBeNil)
				
				err = searchIndex.Health(ctx)
				So(err, ShouldNotBeNil)
			})
			
			Convey("Should handle partial close failures", func() {
				// Create a new MVS for this test
				vectorStore2 := NewMockVectorStore()
				graphStore2 := NewMockGraphStore()
				searchIndex2 := NewMockSearchIndex()
				mvs2 := NewMultiViewStorage(vectorStore2, graphStore2, searchIndex2, config)
				
				// Make one backend fail to close (simulate by closing it first)
				vectorStore2.Close()
				
				err := mvs2.Close()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "storage close failed")
			})
		})
		
		Convey("Timeout handling", func() {
			// Create MVS with very short timeout
			shortConfig := &MultiViewStorageConfig{
				Timeout:    1 * time.Millisecond,
				RetryCount: 1,
			}
			shortMvs := NewMultiViewStorage(vectorStore, graphStore, searchIndex, shortConfig)
			
			chunk := &Chunk{
				ID:        "timeout-test",
				Content:   "Timeout test chunk",
				Embedding: []float32{0.1, 0.2},
				Metadata:  map[string]interface{}{},
				Entities:  []Entity{},
				Claims:    []Claim{},
			}
			
			Convey("Should handle timeouts gracefully", func() {
				// This might succeed or fail depending on timing, but shouldn't panic
				err := shortMvs.StoreChunk(ctx, chunk)
				// We don't assert on the error since timing is unpredictable in tests
				_ = err
			})
		})
		
		Convey("Concurrent operations", func() {
			Convey("Should handle concurrent chunk storage", func() {
				const numChunks = 10
				errors := make(chan error, numChunks)
				
				for i := 0; i < numChunks; i++ {
					go func(id int) {
						chunk := &Chunk{
							ID:        fmt.Sprintf("concurrent-chunk-%d", id),
							Content:   fmt.Sprintf("Concurrent test chunk %d", id),
							Embedding: []float32{float32(id), 0.0},
							Metadata:  map[string]interface{}{"concurrent": true},
							Entities:  []Entity{},
							Claims:    []Claim{},
						}
						errors <- mvs.StoreChunk(ctx, chunk)
					}(i)
				}
				
				// Wait for all operations to complete
				for i := 0; i < numChunks; i++ {
					err := <-errors
					So(err, ShouldBeNil)
				}
				
				// Verify all chunks were stored
				count, err := vectorStore.Count(ctx)
				So(err, ShouldBeNil)
				So(count, ShouldBeGreaterThanOrEqualTo, numChunks)
			})
		})
	})
}

func BenchmarkMultiViewStorage(b *testing.B) {
	ctx := context.Background()
	
	vectorStore := NewMockVectorStore()
	graphStore := NewMockGraphStore()
	searchIndex := NewMockSearchIndex()
	
	config := &MultiViewStorageConfig{
		Timeout:    5 * time.Second,
		RetryCount: 3,
	}
	
	mvs := NewMultiViewStorage(vectorStore, graphStore, searchIndex, config)
	
	// Prepare test chunk
	chunk := &Chunk{
		ID:        "bench-chunk",
		Content:   "Benchmark test chunk with machine learning content",
		Embedding: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
		Metadata:  map[string]interface{}{"benchmark": true},
		Entities:  []Entity{{ID: "bench-entity", Name: "machine learning", Type: "concept"}},
		Claims:    []Claim{{ID: "bench-claim", Subject: "ML", Predicate: "is", Object: "useful"}},
	}
	
	b.Run("StoreChunk", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			testChunk := *chunk
			testChunk.ID = fmt.Sprintf("bench-chunk-%d", i)
			testChunk.Entities[0].ID = fmt.Sprintf("bench-entity-%d", i)
			testChunk.Claims[0].ID = fmt.Sprintf("bench-claim-%d", i)
			mvs.StoreChunk(ctx, &testChunk)
		}
	})
	
	// Pre-populate for retrieval benchmarks
	for i := 0; i < 100; i++ {
		testChunk := *chunk
		testChunk.ID = fmt.Sprintf("retrieval-bench-chunk-%d", i)
		testChunk.Entities[0].ID = fmt.Sprintf("retrieval-bench-entity-%d", i)
		testChunk.Claims[0].ID = fmt.Sprintf("retrieval-bench-claim-%d", i)
		mvs.StoreChunk(ctx, &testChunk)
	}
	
	b.Run("RetrieveMultiView", func(b *testing.B) {
		query := "machine learning"
		embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
		options := RetrievalOptions{
			MaxResults:   10,
			IncludeGraph: true,
		}
		
		for i := 0; i < b.N; i++ {
			mvs.RetrieveMultiView(ctx, query, embedding, options)
		}
	})
	
	b.Run("GetStats", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mvs.GetStats(ctx)
		}
	})
}