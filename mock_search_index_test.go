package main

import (
	"context"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMockSearchIndex(t *testing.T) {
	Convey("MockSearchIndex Implementation", t, func() {
		ctx := context.Background()
		index := NewMockSearchIndex()
		
		Convey("Should implement SearchIndex interface", func() {
			var _ SearchIndex = index
		})
		
		Convey("Document lifecycle operations", func() {
			doc := IndexDocument{
				ID:      "test-doc",
				Content: "This is a test document about machine learning and artificial intelligence",
				Metadata: map[string]interface{}{
					"category": "technology",
					"author":   "test-author",
					"score":    8.5,
				},
			}
			
			Convey("Index and retrieve document", func() {
				err := index.Index(ctx, doc)
				So(err, ShouldBeNil)
				
				retrieved, err := index.GetDocument(ctx, "test-doc")
				So(err, ShouldBeNil)
				So(retrieved.ID, ShouldEqual, "test-doc")
				So(retrieved.Content, ShouldEqual, doc.Content)
				So(retrieved.Metadata["category"], ShouldEqual, "technology")
				So(retrieved.Metadata["score"], ShouldEqual, 8.5)
				
				exists, err := index.DocumentExists(ctx, "test-doc")
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
			
			Convey("Update existing document", func() {
				index.Index(ctx, doc)
				
				updatedDoc := IndexDocument{
					ID:      "test-doc",
					Content: "Updated content about deep learning and neural networks",
					Metadata: map[string]interface{}{
						"category": "AI",
						"version":  2,
						"score":    9.0,
					},
				}
				
				err := index.Update(ctx, "test-doc", updatedDoc)
				So(err, ShouldBeNil)
				
				retrieved, err := index.GetDocument(ctx, "test-doc")
				So(err, ShouldBeNil)
				So(retrieved.Content, ShouldEqual, updatedDoc.Content)
				So(retrieved.Metadata["category"], ShouldEqual, "AI")
				So(retrieved.Metadata["version"], ShouldEqual, 2)
			})
			
			Convey("Delete document", func() {
				index.Index(ctx, doc)
				
				err := index.Delete(ctx, "test-doc")
				So(err, ShouldBeNil)
				
				exists, err := index.DocumentExists(ctx, "test-doc")
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
				
				_, err = index.GetDocument(ctx, "test-doc")
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("Search functionality", func() {
			// Index test documents
			docs := []IndexDocument{
				{
					ID:      "doc1",
					Content: "Machine learning algorithms for data analysis and pattern recognition",
					Metadata: map[string]interface{}{"category": "ML", "score": 8.5},
				},
				{
					ID:      "doc2",
					Content: "Deep learning neural networks and artificial intelligence systems",
					Metadata: map[string]interface{}{"category": "AI", "score": 9.0},
				},
				{
					ID:      "doc3",
					Content: "Natural language processing and computational linguistics",
					Metadata: map[string]interface{}{"category": "NLP", "score": 7.5},
				},
				{
					ID:      "doc4",
					Content: "Computer vision and image recognition using machine learning",
					Metadata: map[string]interface{}{"category": "CV", "score": 8.0},
				},
			}
			
			for _, doc := range docs {
				err := index.Index(ctx, doc)
				So(err, ShouldBeNil)
			}
			
			Convey("Basic text search", func() {
				options := SearchIndexOptions{Limit: 10}
				results, err := index.Search(ctx, "machine learning", options)
				So(err, ShouldBeNil)
				So(len(results), ShouldBeGreaterThan, 0)
				
				// Should find documents containing "machine" and "learning"
				foundDoc1 := false
				foundDoc4 := false
				for _, result := range results {
					So(result.Score, ShouldBeGreaterThan, 0)
					if result.ID == "doc1" {
						foundDoc1 = true
					}
					if result.ID == "doc4" {
						foundDoc4 = true
					}
				}
				So(foundDoc1, ShouldBeTrue)
				So(foundDoc4, ShouldBeTrue)
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
						// Highlights should contain the search term
						for _, highlight := range result.Highlights {
							So(highlight, ShouldContainSubstring, "learning")
						}
						break
					}
				}
				So(hasHighlights, ShouldBeTrue)
			})
			
			Convey("Search with pagination", func() {
				options := SearchIndexOptions{
					Offset: 1,
					Limit:  2,
				}
				results, err := index.Search(ctx, "learning", options)
				So(err, ShouldBeNil)
				So(len(results), ShouldBeLessThanOrEqualTo, 2)
			})
			
			Convey("Search ordering by score", func() {
				options := SearchIndexOptions{Limit: 10}
				results, err := index.Search(ctx, "machine learning", options)
				So(err, ShouldBeNil)
				So(len(results), ShouldBeGreaterThan, 1)
				
				// Results should be ordered by score (descending)
				for i := 1; i < len(results); i++ {
					So(results[i-1].Score, ShouldBeGreaterThanOrEqualTo, results[i].Score)
				}
			})
			
			Convey("Multi-search", func() {
				queries := []string{"machine learning", "neural networks", "language processing"}
				options := SearchIndexOptions{Limit: 5}
				
				results, err := index.MultiSearch(ctx, queries, options)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 3)
				
				// Each query should return some results
				for i, queryResults := range results {
					So(len(queryResults), ShouldBeGreaterThanOrEqualTo, 0)
					_ = i // Use the index to avoid unused variable
				}
			})
			
			Convey("Empty search results", func() {
				options := SearchIndexOptions{Limit: 10}
				results, err := index.Search(ctx, "nonexistent term xyz", options)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 0)
			})
		})
		
		Convey("Tokenization and indexing", func() {
			doc := IndexDocument{
				ID:      "tokenize-test",
				Content: "Hello, World! This is a test with punctuation: semicolons; and numbers 123.",
				Metadata: map[string]interface{}{},
			}
			
			index.Index(ctx, doc)
			
			Convey("Should find words ignoring punctuation", func() {
				options := SearchIndexOptions{Limit: 10}
				
				// Should find "hello" despite capitalization and punctuation
				results, err := index.Search(ctx, "hello", options)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 1)
				So(results[0].ID, ShouldEqual, "tokenize-test")
				
				// Should find "world" despite punctuation
				results, err = index.Search(ctx, "world", options)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 1)
				
				// Should find numbers
				results, err = index.Search(ctx, "123", options)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 1)
			})
		})
		
		Convey("Suggestion functionality", func() {
			// Index documents with various terms
			docs := []IndexDocument{
				{ID: "s1", Content: "machine learning algorithms", Metadata: map[string]interface{}{}},
				{ID: "s2", Content: "machine intelligence systems", Metadata: map[string]interface{}{}},
				{ID: "s3", Content: "manufacturing processes", Metadata: map[string]interface{}{}},
				{ID: "s4", Content: "mathematical models", Metadata: map[string]interface{}{}},
			}
			
			for _, doc := range docs {
				index.Index(ctx, doc)
			}
			
			Convey("Should provide relevant suggestions", func() {
				suggestions, err := index.Suggest(ctx, "mach", "content", 5)
				So(err, ShouldBeNil)
				So(len(suggestions), ShouldBeGreaterThan, 0)
				
				// Should include terms starting with "mach"
				foundMachine := false
				
				for _, suggestion := range suggestions {
					if suggestion == "machine" {
						foundMachine = true
					}
				}
				
				So(foundMachine, ShouldBeTrue)
			})
			
			Convey("Should limit suggestions", func() {
				suggestions, err := index.Suggest(ctx, "ma", "content", 2)
				So(err, ShouldBeNil)
				So(len(suggestions), ShouldBeLessThanOrEqualTo, 2)
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
			
			// All documents should be indexed and searchable
			for _, doc := range docs {
				exists, err := index.DocumentExists(ctx, doc.ID)
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
				
				retrieved, err := index.GetDocument(ctx, doc.ID)
				So(err, ShouldBeNil)
				So(retrieved.Content, ShouldEqual, doc.Content)
				So(retrieved.Metadata["batch"], ShouldEqual, 1)
			}
			
			// Should be findable via search
			options := SearchIndexOptions{Limit: 10}
			results, err := index.Search(ctx, "batch", options)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 3)
		})
		
		Convey("Statistics", func() {
			// Index some documents
			for i := 0; i < 5; i++ {
				doc := IndexDocument{
					ID:      fmt.Sprintf("stats-doc-%d", i),
					Content: fmt.Sprintf("Document %d content for statistics testing", i),
					Metadata: map[string]interface{}{"number": i},
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
		
		Convey("Index management operations", func() {
			// These are no-ops in mock but should not error
			err := index.CreateIndex(ctx, "test-index", map[string]interface{}{
				"mappings": map[string]interface{}{
					"properties": map[string]interface{}{
						"content": map[string]interface{}{"type": "text"},
					},
				},
			})
			So(err, ShouldBeNil)
			
			err = index.RefreshIndex(ctx)
			So(err, ShouldBeNil)
			
			err = index.DeleteIndex(ctx, "test-index")
			So(err, ShouldBeNil)
		})
		
		Convey("Health status management", func() {
			// Initially healthy
			err := index.Health(ctx)
			So(err, ShouldBeNil)
			
			// Set unhealthy
			index.SetHealthy(false)
			err = index.Health(ctx)
			So(err, ShouldNotBeNil)
			
			// Set healthy again
			index.SetHealthy(true)
			err = index.Health(ctx)
			So(err, ShouldBeNil)
		})
		
		Convey("Close functionality", func() {
			err := index.Close()
			So(err, ShouldBeNil)
			
			// All operations should fail after close
			doc := IndexDocument{ID: "after-close", Content: "test", Metadata: map[string]interface{}{}}
			err = index.Index(ctx, doc)
			So(err, ShouldNotBeNil)
			
			_, err = index.Search(ctx, "test", SearchIndexOptions{})
			So(err, ShouldNotBeNil)
			
			_, err = index.GetDocument(ctx, "any-id")
			So(err, ShouldNotBeNil)
			
			err = index.Health(ctx)
			So(err, ShouldNotBeNil)
		})
		
		Convey("Edge cases and error handling", func() {
			Convey("Empty content", func() {
				doc := IndexDocument{
					ID:       "empty-content",
					Content:  "",
					Metadata: map[string]interface{}{"empty": true},
				}
				
				err := index.Index(ctx, doc)
				So(err, ShouldBeNil)
				
				retrieved, err := index.GetDocument(ctx, "empty-content")
				So(err, ShouldBeNil)
				So(retrieved.Content, ShouldEqual, "")
			})
			
			Convey("Special characters in content", func() {
				doc := IndexDocument{
					ID:      "special-chars",
					Content: "Content with émojis 🚀 and spëcial chàracters!",
					Metadata: map[string]interface{}{},
				}
				
				err := index.Index(ctx, doc)
				So(err, ShouldBeNil)
				
				// Should still be searchable
				options := SearchIndexOptions{Limit: 10}
				results, err := index.Search(ctx, "content", options)
				So(err, ShouldBeNil)
				So(len(results), ShouldBeGreaterThan, 0)
			})
			
			Convey("Non-existent document operations", func() {
				_, err := index.GetDocument(ctx, "non-existent")
				So(err, ShouldNotBeNil)
				
				err = index.Delete(ctx, "non-existent")
				So(err, ShouldNotBeNil)
				
				exists, err := index.DocumentExists(ctx, "non-existent")
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
			})
		})
	})
}

func BenchmarkMockSearchIndex(b *testing.B) {
	ctx := context.Background()
	index := NewMockSearchIndex()
	
	// Prepare test documents
	docs := make([]IndexDocument, 1000)
	for i := 0; i < 1000; i++ {
		docs[i] = IndexDocument{
			ID:      fmt.Sprintf("bench-doc-%d", i),
			Content: fmt.Sprintf("This is benchmark document %d with machine learning and artificial intelligence content", i),
			Metadata: map[string]interface{}{
				"category": "benchmark",
				"number":   i,
				"score":    float64(i) / 100.0,
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
	
	b.Run("Tokenize", func(b *testing.B) {
		text := "This is a sample text with various words for tokenization benchmarking"
		
		for i := 0; i < b.N; i++ {
			index.tokenize(text)
		}
	})
}