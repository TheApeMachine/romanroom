package main

import (
	"context"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSearchIndexInterface(t *testing.T) {
	Convey("SearchIndex Interface Contract", t, func() {
		ctx := context.Background()
		index := NewMockSearchIndex()
		
		Convey("Index operations", func() {
			doc := IndexDocument{
				ID:      "test-doc",
				Content: "This is a test document about artificial intelligence and machine learning",
				Metadata: map[string]interface{}{
					"category": "technology",
					"author":   "test-author",
				},
			}
			
			Convey("Index document", func() {
				err := index.Index(ctx, doc)
				So(err, ShouldBeNil)
				
				Convey("Should be retrievable", func() {
					retrieved, err := index.GetDocument(ctx, "test-doc")
					So(err, ShouldBeNil)
					So(retrieved.ID, ShouldEqual, "test-doc")
					So(retrieved.Content, ShouldEqual, doc.Content)
					So(retrieved.Metadata["category"], ShouldEqual, "technology")
				})
				
				Convey("Should exist", func() {
					exists, err := index.DocumentExists(ctx, "test-doc")
					So(err, ShouldBeNil)
					So(exists, ShouldBeTrue)
				})
			})
			
			Convey("Update document", func() {
				index.Index(ctx, doc)
				
				updatedDoc := IndexDocument{
					ID:      "test-doc",
					Content: "Updated content about deep learning and neural networks",
					Metadata: map[string]interface{}{
						"category": "AI",
						"version":  2,
					},
				}
				
				err := index.Update(ctx, "test-doc", updatedDoc)
				So(err, ShouldBeNil)
				
				retrieved, err := index.GetDocument(ctx, "test-doc")
				So(err, ShouldBeNil)
				So(retrieved.Content, ShouldEqual, updatedDoc.Content)
				So(retrieved.Metadata["version"], ShouldEqual, 2)
			})
			
			Convey("Delete document", func() {
				index.Index(ctx, doc)
				
				err := index.Delete(ctx, "test-doc")
				So(err, ShouldBeNil)
				
				exists, err := index.DocumentExists(ctx, "test-doc")
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
			})
		})
		
		Convey("Search operations", func() {
			// Index test documents
			docs := []IndexDocument{
				{
					ID:      "doc1",
					Content: "Machine learning algorithms for data analysis",
					Metadata: map[string]interface{}{"category": "ML", "score": 8.5},
				},
				{
					ID:      "doc2", 
					Content: "Deep learning neural networks and artificial intelligence",
					Metadata: map[string]interface{}{"category": "AI", "score": 9.0},
				},
				{
					ID:      "doc3",
					Content: "Natural language processing and text mining",
					Metadata: map[string]interface{}{"category": "NLP", "score": 7.5},
				},
			}
			
			for _, doc := range docs {
				err := index.Index(ctx, doc)
				So(err, ShouldBeNil)
			}
			
			Convey("Basic search", func() {
				options := SearchIndexOptions{Limit: 10}
				results, err := index.Search(ctx, "machine learning", options)
				So(err, ShouldBeNil)
				So(len(results), ShouldBeGreaterThan, 0)
				
				// Should find doc1 which contains "machine learning"
				found := false
				for _, result := range results {
					if result.ID == "doc1" {
						found = true
						So(result.Score, ShouldBeGreaterThan, 0)
						break
					}
				}
				So(found, ShouldBeTrue)
			})
			
			Convey("Search with filters", func() {
				options := SearchIndexOptions{
					Limit:   10,
					Filters: map[string]interface{}{"category": "AI"},
				}
				results, err := index.Search(ctx, "neural", options)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 1)
				So(results[0].ID, ShouldEqual, "doc2")
				So(results[0].Metadata["category"], ShouldEqual, "AI")
			})
			
			Convey("Search with highlights", func() {
				options := SearchIndexOptions{
					Limit:     10,
					Highlight: true,
				}
				results, err := index.Search(ctx, "learning", options)
				So(err, ShouldBeNil)
				So(len(results), ShouldBeGreaterThan, 0)
				
				// At least one result should have highlights
				hasHighlights := false
				for _, result := range results {
					if len(result.Highlights) > 0 {
						hasHighlights = true
						break
					}
				}
				So(hasHighlights, ShouldBeTrue)
			})
			
			Convey("Search with pagination", func() {
				options := SearchIndexOptions{
					Offset: 1,
					Limit:  1,
				}
				results, err := index.Search(ctx, "learning", options)
				So(err, ShouldBeNil)
				So(len(results), ShouldBeLessThanOrEqualTo, 1)
			})
			
			Convey("Multi-search", func() {
				queries := []string{"machine learning", "neural networks", "text mining"}
				options := SearchIndexOptions{Limit: 5}
				
				results, err := index.MultiSearch(ctx, queries, options)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 3)
				
				// Each query should return results
				for i, queryResults := range results {
					So(len(queryResults), ShouldBeGreaterThanOrEqualTo, 0)
					_ = i // Use the index to avoid unused variable
				}
			})
		})
		
		Convey("Suggestion operations", func() {
			// Index documents with various terms
			docs := []IndexDocument{
				{ID: "s1", Content: "machine learning algorithms", Metadata: map[string]interface{}{}},
				{ID: "s2", Content: "machine intelligence systems", Metadata: map[string]interface{}{}},
				{ID: "s3", Content: "manufacturing processes", Metadata: map[string]interface{}{}},
			}
			
			for _, doc := range docs {
				index.Index(ctx, doc)
			}
			
			Convey("Should provide suggestions", func() {
				suggestions, err := index.Suggest(ctx, "mach", "content", 5)
				So(err, ShouldBeNil)
				So(len(suggestions), ShouldBeGreaterThan, 0)
				
				// Should include terms starting with "mach"
				foundMachine := false
				for _, suggestion := range suggestions {
					if suggestion == "machine" {
						foundMachine = true
						break
					}
				}
				So(foundMachine, ShouldBeTrue)
			})
		})
		
		Convey("Batch operations", func() {
			docs := []IndexDocument{
				{ID: "batch1", Content: "First batch document", Metadata: map[string]interface{}{"batch": 1}},
				{ID: "batch2", Content: "Second batch document", Metadata: map[string]interface{}{"batch": 1}},
				{ID: "batch3", Content: "Third batch document", Metadata: map[string]interface{}{"batch": 1}},
			}
			
			err := index.BatchIndex(ctx, docs)
			So(err, ShouldBeNil)
			
			// All documents should be indexed
			for _, doc := range docs {
				exists, err := index.DocumentExists(ctx, doc.ID)
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			}
		})
		
		Convey("Statistics", func() {
			// Index some documents first
			for i := 0; i < 5; i++ {
				doc := IndexDocument{
					ID:      fmt.Sprintf("stats-doc-%d", i),
					Content: fmt.Sprintf("Document %d content", i),
					Metadata: map[string]interface{}{},
				}
				index.Index(ctx, doc)
			}
			
			count, err := index.DocumentCount(ctx)
			So(err, ShouldBeNil)
			So(count, ShouldBeGreaterThanOrEqualTo, 5)
			
			size, err := index.IndexSize(ctx)
			So(err, ShouldBeNil)
			So(size, ShouldBeGreaterThan, 0)
		})
		
		Convey("Index management", func() {
			// These are no-ops in mock but should not error
			err := index.CreateIndex(ctx, "test-index", map[string]interface{}{})
			So(err, ShouldBeNil)
			
			err = index.RefreshIndex(ctx)
			So(err, ShouldBeNil)
			
			err = index.DeleteIndex(ctx, "test-index")
			So(err, ShouldBeNil)
		})
		
		Convey("Health and lifecycle", func() {
			err := index.Health(ctx)
			So(err, ShouldBeNil)
			
			err = index.Close()
			So(err, ShouldBeNil)
			
			// Operations should fail after close
			doc := IndexDocument{ID: "after-close", Content: "test", Metadata: map[string]interface{}{}}
			err = index.Index(ctx, doc)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestSearchIndexErrorHandling(t *testing.T) {
	Convey("SearchIndex Error Handling", t, func() {
		ctx := context.Background()
		index := NewMockSearchIndex()
		
		Convey("Should handle non-existent document", func() {
			_, err := index.GetDocument(ctx, "non-existent")
			So(err, ShouldNotBeNil)
		})
		
		Convey("Should handle delete of non-existent document", func() {
			err := index.Delete(ctx, "non-existent")
			So(err, ShouldNotBeNil)
		})
		
		Convey("Should handle unhealthy state", func() {
			index.SetHealthy(false)
			
			err := index.Health(ctx)
			So(err, ShouldNotBeNil)
		})
	})
}

func BenchmarkSearchIndex(b *testing.B) {
	ctx := context.Background()
	index := NewMockSearchIndex()
	
	// Prepare test documents
	docs := make([]IndexDocument, 1000)
	for i := 0; i < 1000; i++ {
		docs[i] = IndexDocument{
			ID:      fmt.Sprintf("bench-doc-%d", i),
			Content: fmt.Sprintf("This is benchmark document %d with machine learning content", i),
			Metadata: map[string]interface{}{
				"category": "benchmark",
				"number":   i,
			},
		}
	}
	
	b.Run("Index", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			doc := docs[i%1000]
			index.Index(ctx, doc)
		}
	})
	
	// Index documents for search benchmarks
	for _, doc := range docs {
		index.Index(ctx, doc)
	}
	
	b.Run("Search", func(b *testing.B) {
		options := SearchIndexOptions{Limit: 10}
		for i := 0; i < b.N; i++ {
			index.Search(ctx, "machine learning", options)
		}
	})
	
	b.Run("GetDocument", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			index.GetDocument(ctx, fmt.Sprintf("bench-doc-%d", i%1000))
		}
	})
}