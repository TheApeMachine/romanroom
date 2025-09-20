package main

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFileSearchIndex(t *testing.T) {
	Convey("Given a FileSearchIndex", t, func() {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "search_index.json")
		index := NewFileSearchIndex(filePath)
		ctx := context.Background()
		
		Convey("When indexing a document", func() {
			doc := IndexDocument{
				ID:      "doc1",
				Content: "This is a test document about artificial intelligence",
				Metadata: map[string]interface{}{
					"category": "AI",
					"author":   "test",
				},
			}
			
			err := index.Index(ctx, doc)
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
			
			Convey("And the document should be retrievable", func() {
				retrieved, err := index.GetDocument(ctx, "doc1")
				So(err, ShouldBeNil)
				So(retrieved, ShouldNotBeNil)
				So(retrieved.ID, ShouldEqual, "doc1")
				So(retrieved.Content, ShouldEqual, doc.Content)
				So(retrieved.Metadata["category"], ShouldEqual, "AI")
			})
			
			Convey("And document existence check should work", func() {
				exists, err := index.DocumentExists(ctx, "doc1")
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
				
				exists, err = index.DocumentExists(ctx, "nonexistent")
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
			})
		})
		
		Convey("When searching for documents", func() {
			// Index test documents
			docs := []IndexDocument{
				{
					ID:      "doc1",
					Content: "artificial intelligence machine learning",
					Metadata: map[string]interface{}{"category": "AI"},
				},
				{
					ID:      "doc2",
					Content: "natural language processing text analysis",
					Metadata: map[string]interface{}{"category": "NLP"},
				},
				{
					ID:      "doc3",
					Content: "machine learning algorithms and artificial neural networks",
					Metadata: map[string]interface{}{"category": "AI"},
				},
			}
			
			for _, doc := range docs {
				err := index.Index(ctx, doc)
				So(err, ShouldBeNil)
			}
			
			options := SearchIndexOptions{
				Limit:     10,
				Highlight: true,
			}
			
			results, err := index.Search(ctx, "machine learning", options)
			
			Convey("Then it should return relevant documents", func() {
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 2) // doc1 and doc3 contain "machine learning"
				
				// Results should be sorted by score (descending)
				So(results[0].Score, ShouldBeGreaterThanOrEqualTo, results[1].Score)
				
				// Check that highlights are included
				for _, result := range results {
					So(len(result.Highlights), ShouldBeGreaterThan, 0)
				}
			})
			
			Convey("And filtering should work", func() {
				options.Filters = map[string]interface{}{"category": "AI"}
				results, err := index.Search(ctx, "artificial", options)
				So(err, ShouldBeNil)
				
				for _, result := range results {
					So(result.Metadata["category"], ShouldEqual, "AI")
				}
			})
			
			Convey("And pagination should work", func() {
				options.Offset = 1
				options.Limit = 1
				options.Filters = nil
				
				results, err := index.Search(ctx, "machine learning", options)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 1)
			})
			
			Convey("And sorting should work", func() {
				options.SortBy = "id"
				options.SortOrder = "asc"
				options.Offset = 0
				options.Limit = 10
				
				results, err := index.Search(ctx, "artificial", options)
				So(err, ShouldBeNil)
				So(len(results), ShouldBeGreaterThan, 1)
				
				// Should be sorted by ID ascending
				for i := 1; i < len(results); i++ {
					So(results[i-1].ID, ShouldBeLessThan, results[i].ID)
				}
			})
		})
		
		Convey("When batch indexing documents", func() {
			docs := []IndexDocument{
				{ID: "batch1", Content: "first batch document", Metadata: map[string]interface{}{"batch": 1}},
				{ID: "batch2", Content: "second batch document", Metadata: map[string]interface{}{"batch": 1}},
				{ID: "batch3", Content: "third batch document", Metadata: map[string]interface{}{"batch": 1}},
			}
			
			err := index.BatchIndex(ctx, docs)
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
			
			Convey("And all documents should be indexed", func() {
				count, err := index.DocumentCount(ctx)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 3)
				
				for _, doc := range docs {
					retrieved, err := index.GetDocument(ctx, doc.ID)
					So(err, ShouldBeNil)
					So(retrieved.Content, ShouldEqual, doc.Content)
				}
			})
		})
		
		Convey("When updating a document", func() {
			originalDoc := IndexDocument{
				ID:      "update_test",
				Content: "original content",
				Metadata: map[string]interface{}{"version": 1},
			}
			
			err := index.Index(ctx, originalDoc)
			So(err, ShouldBeNil)
			
			updatedDoc := IndexDocument{
				ID:      "update_test",
				Content: "updated content with new keywords",
				Metadata: map[string]interface{}{"version": 2},
			}
			
			err = index.Update(ctx, "update_test", updatedDoc)
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
			
			Convey("And the document should be updated", func() {
				retrieved, err := index.GetDocument(ctx, "update_test")
				So(err, ShouldBeNil)
				So(retrieved.Content, ShouldEqual, "updated content with new keywords")
				So(retrieved.Metadata["version"], ShouldEqual, 2)
			})
			
			Convey("And search should reflect the update", func() {
				// Should find with new keywords
				results, err := index.Search(ctx, "keywords", SearchIndexOptions{})
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 1)
				So(results[0].ID, ShouldEqual, "update_test")
				
				// Should not find with old keywords
				results, err = index.Search(ctx, "original", SearchIndexOptions{})
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 0)
			})
		})
		
		Convey("When deleting a document", func() {
			doc := IndexDocument{
				ID:      "delete_test",
				Content: "document to be deleted",
				Metadata: map[string]interface{}{"temp": true},
			}
			
			err := index.Index(ctx, doc)
			So(err, ShouldBeNil)
			
			err = index.Delete(ctx, "delete_test")
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
			
			Convey("And the document should not be found", func() {
				_, err := index.GetDocument(ctx, "delete_test")
				So(err, ShouldNotBeNil)
				
				exists, err := index.DocumentExists(ctx, "delete_test")
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
			})
			
			Convey("And search should not return the deleted document", func() {
				results, err := index.Search(ctx, "deleted", SearchIndexOptions{})
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 0)
			})
		})
		
		Convey("When getting suggestions", func() {
			// Index documents with various terms
			docs := []IndexDocument{
				{ID: "s1", Content: "machine learning algorithms", Metadata: nil},
				{ID: "s2", Content: "machine intelligence systems", Metadata: nil},
				{ID: "s3", Content: "natural language processing", Metadata: nil},
			}
			
			for _, doc := range docs {
				err := index.Index(ctx, doc)
				So(err, ShouldBeNil)
			}
			
			suggestions, err := index.Suggest(ctx, "mac", "", 5)
			
			Convey("Then it should return matching suggestions", func() {
				So(err, ShouldBeNil)
				So(len(suggestions), ShouldBeGreaterThan, 0)
				
				// Should contain "machine"
				So(suggestions, ShouldContain, "machine")
			})
		})
		
		Convey("When performing multi-search", func() {
			// Index test documents
			docs := []IndexDocument{
				{ID: "m1", Content: "artificial intelligence research", Metadata: nil},
				{ID: "m2", Content: "machine learning models", Metadata: nil},
				{ID: "m3", Content: "natural language understanding", Metadata: nil},
			}
			
			for _, doc := range docs {
				err := index.Index(ctx, doc)
				So(err, ShouldBeNil)
			}
			
			queries := []string{"artificial", "machine", "natural"}
			results, err := index.MultiSearch(ctx, queries, SearchIndexOptions{})
			
			Convey("Then it should return results for each query", func() {
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 3)
				
				// Each query should have at least one result
				for i, queryResults := range results {
					So(len(queryResults), ShouldBeGreaterThan, 0)
					// Verify the result contains the query term
					query := queries[i]
					found := false
					for _, result := range queryResults {
						if containsSubstring(result.Content, query) {
							found = true
							break
						}
					}
					So(found, ShouldBeTrue)
				}
			})
		})
		
		Convey("When getting document count and index size", func() {
			// Index some documents
			for i := 0; i < 5; i++ {
				doc := IndexDocument{
					ID:      fmt.Sprintf("count_%d", i),
					Content: fmt.Sprintf("Document number %d with some content", i),
					Metadata: map[string]interface{}{"number": i},
				}
				err := index.Index(ctx, doc)
				So(err, ShouldBeNil)
			}
			
			count, err := index.DocumentCount(ctx)
			
			Convey("Then document count should be correct", func() {
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 5)
			})
			
			size, err := index.IndexSize(ctx)
			
			Convey("And index size should be positive", func() {
				So(err, ShouldBeNil)
				So(size, ShouldBeGreaterThan, 0)
			})
		})
		
		Convey("When checking health", func() {
			err := index.Health(ctx)
			
			Convey("Then it should be healthy", func() {
				So(err, ShouldBeNil)
			})
		})
		
		Convey("When closing the index", func() {
			err := index.Close()
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
			
			Convey("And subsequent operations should fail", func() {
				doc := IndexDocument{ID: "test", Content: "test", Metadata: nil}
				err := index.Index(ctx, doc)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "closed")
			})
		})
		
		Convey("When loading and saving", func() {
			// Index some data
			doc := IndexDocument{
				ID:      "persist_test",
				Content: "persistent document content",
				Metadata: map[string]interface{}{"persistent": true},
			}
			
			err := index.Index(ctx, doc)
			So(err, ShouldBeNil)
			
			// Save explicitly
			err = index.Save()
			So(err, ShouldBeNil)
			
			// Create new index instance and load
			newIndex := NewFileSearchIndex(filePath)
			err = newIndex.Load()
			
			Convey("Then data should persist", func() {
				So(err, ShouldBeNil)
				
				retrieved, err := newIndex.GetDocument(ctx, "persist_test")
				So(err, ShouldBeNil)
				So(retrieved.ID, ShouldEqual, "persist_test")
				So(retrieved.Content, ShouldEqual, "persistent document content")
				So(retrieved.Metadata["persistent"], ShouldEqual, true)
				
				// Search should also work
				results, err := newIndex.Search(ctx, "persistent", SearchIndexOptions{})
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 1)
				So(results[0].ID, ShouldEqual, "persist_test")
			})
		})
	})
}

func TestFileSearchIndexEdgeCases(t *testing.T) {
	Convey("Given a FileSearchIndex with edge cases", t, func() {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "search_index_edge.json")
		index := NewFileSearchIndex(filePath)
		ctx := context.Background()
		
		Convey("When getting a non-existent document", func() {
			_, err := index.GetDocument(ctx, "nonexistent")
			
			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("When deleting a non-existent document", func() {
			err := index.Delete(ctx, "nonexistent")
			
			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("When searching with empty query", func() {
			results, err := index.Search(ctx, "", SearchIndexOptions{})
			
			Convey("Then it should return empty results", func() {
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 0)
			})
		})
		
		Convey("When searching with only whitespace", func() {
			results, err := index.Search(ctx, "   ", SearchIndexOptions{})
			
			Convey("Then it should return empty results", func() {
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 0)
			})
		})
		
		Convey("When loading from non-existent file", func() {
			nonExistentIndex := NewFileSearchIndex(filepath.Join(tempDir, "nonexistent.json"))
			err := nonExistentIndex.Load()
			
			Convey("Then it should succeed with empty index", func() {
				So(err, ShouldBeNil)
				count, err := nonExistentIndex.DocumentCount(ctx)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 0)
			})
		})
		
		Convey("When indexing document with empty content", func() {
			doc := IndexDocument{
				ID:      "empty",
				Content: "",
				Metadata: map[string]interface{}{"empty": true},
			}
			
			err := index.Index(ctx, doc)
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
				
				retrieved, err := index.GetDocument(ctx, "empty")
				So(err, ShouldBeNil)
				So(retrieved.Content, ShouldEqual, "")
			})
		})
		
		Convey("When searching with pagination beyond results", func() {
			// Index one document
			doc := IndexDocument{ID: "single", Content: "single document", Metadata: nil}
			err := index.Index(ctx, doc)
			So(err, ShouldBeNil)
			
			// Search with offset beyond results
			options := SearchIndexOptions{Offset: 10, Limit: 5}
			results, err := index.Search(ctx, "single", options)
			
			Convey("Then it should return empty results", func() {
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 0)
			})
		})
	})
}

func BenchmarkFileSearchIndex(b *testing.B) {
	tempDir := b.TempDir()
	filePath := filepath.Join(tempDir, "bench_search_index.json")
	index := NewFileSearchIndex(filePath)
	ctx := context.Background()
	
	// Prepare test data
	content := "This is a sample document with various keywords for benchmarking search performance"
	metadata := map[string]interface{}{"benchmark": true}
	
	b.Run("Index", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			doc := IndexDocument{
				ID:       fmt.Sprintf("bench_%d", i),
				Content:  fmt.Sprintf("%s %d", content, i),
				Metadata: metadata,
			}
			index.Index(ctx, doc)
		}
	})
	
	// Index some documents for search benchmark
	for i := 0; i < 1000; i++ {
		doc := IndexDocument{
			ID:       fmt.Sprintf("search_bench_%d", i),
			Content:  fmt.Sprintf("%s %d", content, i),
			Metadata: metadata,
		}
		index.Index(ctx, doc)
	}
	
	b.Run("Search", func(b *testing.B) {
		options := SearchIndexOptions{Limit: 10}
		for i := 0; i < b.N; i++ {
			index.Search(ctx, "sample document", options)
		}
	})
	
	b.Run("GetDocument", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			id := fmt.Sprintf("search_bench_%d", i%1000)
			index.GetDocument(ctx, id)
		}
	})
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsSubstring(text, substr string) bool {
	return len(text) >= len(substr) && 
		   (text == substr || 
		    len(text) > len(substr) && 
		    (text[:len(substr)] == substr || 
		     text[len(text)-len(substr):] == substr || 
		     containsMiddle(text, substr)))
}

func containsMiddle(text, substr string) bool {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}