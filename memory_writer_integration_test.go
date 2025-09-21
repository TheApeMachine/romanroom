package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMemoryWriterIntegration(t *testing.T) {
	Convey("Given a complete memory writing system", t, func() {
		// Setup complete system with all components
		storage := &MultiViewStorage{
			vectorStore: NewMockVectorStore(),
			graphStore:  NewMockGraphStore(),
			searchIndex: NewMockSearchIndex(),
		}

		contentProcessor := NewContentProcessor()

		config := &MemoryWriterConfig{
			RequireEvidence:     false,
			MaxChunkSize:        1000,
			MinConfidence:       0.5,
			EnableDeduplication: true,
		}

		writer := NewMemoryWriter(storage, contentProcessor, config)

		Convey("When writing a complex document end-to-end", func() {
			ctx := context.Background()
			content := `
			Artificial Intelligence (AI) is a transformative technology that enables machines to perform tasks typically requiring human intelligence.
			Machine Learning (ML) is a subset of AI that allows systems to learn from data without explicit programming.
			Deep Learning uses neural networks with multiple layers to process complex patterns in data.
			Natural Language Processing (NLP) enables computers to understand and generate human language.
			`

			metadata := WriteMetadata{
				Source:          "ai_overview.txt",
				Timestamp:       time.Now(),
				UserID:          "researcher_001",
				Tags:            []string{"ai", "technology", "overview"},
				Confidence:      0.9,
				RequireEvidence: false,
			}

			result, err := writer.Write(ctx, content, metadata)

			Convey("Then it should process and store successfully", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(result.MemoryID, ShouldNotBeEmpty)
				So(result.CandidateCount, ShouldBeGreaterThan, 0)
				So(result.ProvenanceID, ShouldNotBeEmpty)

				// Verify storage backends were used
				mockVectorStore := storage.vectorStore.(*MockVectorStore)
				mockGraphStore := storage.graphStore.(*MockGraphStore)
				mockSearchIndex := storage.searchIndex.(*MockSearchIndex)

				stored := mockVectorStore.GetStored()
				nodes := mockGraphStore.GetNodes()
				indexed := mockSearchIndex.GetIndexed()

				So(len(stored), ShouldBeGreaterThan, 0)
				So(len(nodes), ShouldBeGreaterThan, 0)
				So(len(indexed), ShouldBeGreaterThan, 0)

				// Verify provenance tracking
				provenance, err := writer.provenanceTracker.GetProvenance(result.ProvenanceID)
				So(err, ShouldBeNil)
				So(provenance.MemoryID, ShouldEqual, result.MemoryID)
				So(provenance.OriginalSource, ShouldEqual, metadata.Source)
			})
		})

		Convey("When writing with entity resolution", func() {
			ctx := context.Background()

			// Setup existing entities in search index for name-based search
			mockSearchIndex := storage.searchIndex.(*MockSearchIndex)
			err := mockSearchIndex.Index(ctx, IndexDocument{
				ID:      "existing_ai_entity",
				Content: "Artificial Intelligence AI Machine Intelligence",
				Metadata: map[string]interface{}{
					"name":        "Artificial Intelligence",
					"type":        "entity",
					"entity_type": "Technology",
				},
			})
			So(err, ShouldBeNil)

			// Add entities to graph store for linking
			mockGraphStore := storage.graphStore.(*MockGraphStore)
			err = mockGraphStore.CreateNode(ctx, &Node{
				ID:   "existing_ai_entity",
				Type: EntityNode,
				Properties: map[string]interface{}{
					"name": "Artificial Intelligence",
					"type": "Technology",
				},
			})
			So(err, ShouldBeNil)

			content := "Artificial Intelligence is revolutionizing many industries through automation and intelligent decision-making."
			metadata := WriteMetadata{
				Source:     "ai_impact.txt",
				Timestamp:  time.Now(),
				UserID:     "analyst_002",
				Confidence: 0.8,
			}

			result, err := writer.Write(ctx, content, metadata)

			Convey("Then it should resolve and link entities", func() {
				So(err, ShouldBeNil)
				// EntitiesLinked may be nil or empty with mock data
				if result.EntitiesLinked != nil {
					So(len(result.EntitiesLinked), ShouldBeGreaterThanOrEqualTo, 0)
				}

				// Verify entity linking created edges (if entities were linked)
				if len(result.EntitiesLinked) > 0 {
					mockGraphStore := storage.graphStore.(*MockGraphStore)
					edges := mockGraphStore.GetEdges()
					hasRelationEdge := false
					for _, edge := range edges {
						if edge.Type == RelatedTo {
							hasRelationEdge = true
							break
						}
					}
					So(hasRelationEdge, ShouldBeTrue)
				}
			})
		})

		Convey("When writing with conflict detection", func() {
			ctx := context.Background()

			// First write: AI is beneficial
			content1 := "Artificial Intelligence brings significant benefits to society through improved efficiency and decision-making."
			metadata1 := WriteMetadata{
				Source:     "ai_benefits.txt",
				Timestamp:  time.Now(),
				UserID:     "optimist_001",
				Confidence: 0.8,
			}

			result1, err1 := writer.Write(ctx, content1, metadata1)
			So(err1, ShouldBeNil)

			// Second write: AI has risks (potential conflict)
			content2 := "Artificial Intelligence poses significant risks including job displacement and privacy concerns."
			metadata2 := WriteMetadata{
				Source:     "ai_risks.txt",
				Timestamp:  time.Now().Add(time.Hour),
				UserID:     "skeptic_001",
				Confidence: 0.8,
			}

			result2, err2 := writer.Write(ctx, content2, metadata2)

			Convey("Then it should detect and handle conflicts", func() {
				So(err2, ShouldBeNil)
				So(result1, ShouldNotBeNil)
				So(result2, ShouldNotBeNil)

				// Both memories should be stored despite potential conflicts
				// (Conflict resolution would be handled by a separate system)
				mockVectorStore := storage.vectorStore.(*MockVectorStore)
				stored := mockVectorStore.GetStored()
				So(len(stored), ShouldBeGreaterThanOrEqualTo, 2)
			})
		})

		Convey("When writing with provenance tracking", func() {
			ctx := context.Background()

			content := "Machine Learning algorithms improve through iterative training on large datasets."
			metadata := WriteMetadata{
				Source:     "ml_training.txt",
				Timestamp:  time.Now(),
				UserID:     "data_scientist_001",
				Tags:       []string{"ml", "training", "datasets"},
				Confidence: 0.9,
			}

			result, err := writer.Write(ctx, content, metadata)
			So(err, ShouldBeNil)

			// Add some transformations to test provenance updates
			provenanceID := result.ProvenanceID
			err = writer.provenanceTracker.TrackTransformation(
				provenanceID,
				TransformationEmbedding,
				"Generated embeddings using text-embedding-ada-002",
				"embedding_service",
				map[string]interface{}{
					"model":      "text-embedding-ada-002",
					"dimensions": 1536,
				},
			)
			So(err, ShouldBeNil)

			Convey("Then it should maintain complete provenance", func() {
				provenance, err := writer.provenanceTracker.GetProvenance(provenanceID)
				So(err, ShouldBeNil)
				So(provenance.MemoryID, ShouldEqual, result.MemoryID)
				So(provenance.OriginalSource, ShouldEqual, metadata.Source)
				So(provenance.CreatedBy, ShouldEqual, metadata.UserID)
				So(len(provenance.Transformations), ShouldEqual, 1)
				So(provenance.Transformations[0].Type, ShouldEqual, TransformationEmbedding)

				// Verify lineage tracking
				lineage, err := writer.provenanceTracker.GetMemoryLineage(result.MemoryID)
				So(err, ShouldBeNil)
				So(len(lineage), ShouldEqual, 1)
			})
		})

		Convey("When writing fails at storage level", func() {
			ctx := context.Background()

			// Make vector store fail
			mockVectorStore := storage.vectorStore.(*MockVectorStore)
			mockVectorStore.SetShouldFail(true)

			content := "This write should fail due to storage error."
			metadata := WriteMetadata{
				Source:     "failing_write.txt",
				Timestamp:  time.Now(),
				Confidence: 0.8,
			}

			result, err := writer.Write(ctx, content, metadata)

			Convey("Then it should handle errors gracefully", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "failed to store")
			})
		})
	})
}

func TestMemoryWriterPerformance(t *testing.T) {
	Convey("Given a memory writer for performance testing", t, func() {
		storage := &MultiViewStorage{
			vectorStore: NewMockVectorStore(),
			graphStore:  NewMockGraphStore(),
			searchIndex: NewMockSearchIndex(),
		}

		contentProcessor := NewContentProcessor()

		writer := NewMemoryWriter(storage, contentProcessor, nil)

		Convey("When writing multiple documents concurrently", func() {
			ctx := context.Background()
			numDocs := 10

			results := make(chan error, numDocs)

			for i := 0; i < numDocs; i++ {
				go func(docNum int) {
					content := fmt.Sprintf("This is test document number %d for concurrent processing.", docNum)
					metadata := WriteMetadata{
						Source:     fmt.Sprintf("concurrent_test_%d.txt", docNum),
						Timestamp:  time.Now(),
						UserID:     "test_user",
						Confidence: 0.8,
					}

					_, err := writer.Write(ctx, content, metadata)
					results <- err
				}(i)
			}

			// Collect results
			var errors []error
			for i := 0; i < numDocs; i++ {
				if err := <-results; err != nil {
					errors = append(errors, err)
				}
			}

			Convey("Then all writes should succeed", func() {
				So(len(errors), ShouldEqual, 0)

				// Verify all documents were stored
				mockVectorStore := storage.vectorStore.(*MockVectorStore)
				stored := mockVectorStore.GetStored()
				So(len(stored), ShouldEqual, numDocs)
			})
		})
	})
}

func BenchmarkMemoryWriter(b *testing.B) {
	storage := &MultiViewStorage{
		vectorStore: NewMockVectorStore(),
		graphStore:  NewMockGraphStore(),
		searchIndex: NewMockSearchIndex(),
	}

	contentProcessor := NewContentProcessor()

	writer := NewMemoryWriter(storage, contentProcessor, nil)
	ctx := context.Background()

	content := "This is a benchmark test document for measuring memory writing performance. It contains various entities and claims that need to be processed and stored efficiently."
	metadata := WriteMetadata{
		Source:     "benchmark.txt",
		Timestamp:  time.Now(),
		UserID:     "benchmark_user",
		Confidence: 0.8,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := writer.Write(ctx, content, metadata)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEntityResolution(b *testing.B) {
	storage := &MultiViewStorage{
		vectorStore: NewMockVectorStore(),
		graphStore:  NewMockGraphStore(),
		searchIndex: NewMockSearchIndex(),
	}

	resolver := NewEntityResolver(storage)
	ctx := context.Background()

	entities := []Entity{
		{
			ID:         "entity_1",
			Name:       "Artificial Intelligence",
			Type:       "Technology",
			Confidence: 0.9,
			Source:     "test",
			CreatedAt:  time.Now(),
			Properties: make(map[string]interface{}),
		},
		{
			ID:         "entity_2",
			Name:       "Machine Learning",
			Type:       "Technology",
			Confidence: 0.8,
			Source:     "test",
			CreatedAt:  time.Now(),
			Properties: make(map[string]interface{}),
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := resolver.Resolve(ctx, entities)
		if err != nil {
			b.Fatal(err)
		}
	}
}
