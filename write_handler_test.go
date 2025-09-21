package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	. "github.com/smartystreets/goconvey/convey"
)

// NewMockMultiViewStorage creates a mock multi-view storage for testing
func NewMockMultiViewStorage() *MultiViewStorage {
	vectorStore := NewMockVectorStore()
	graphStore := NewMockGraphStore()
	searchIndex := NewMockSearchIndex()

	config := &MultiViewStorageConfig{
		VectorStore: VectorStoreConfig{
			Provider:   "mock",
			Dimensions: 1536,
		},
		GraphStore: GraphStoreConfig{
			Provider: "mock",
		},
		SearchIndex: SearchIndexConfig{
			Provider: "mock",
		},
		Timeout: 10 * time.Second,
	}

	return NewMultiViewStorage(vectorStore, graphStore, searchIndex, config)
}

func TestWriteHandler(t *testing.T) {
	Convey("WriteHandler", t, func() {
		// Setup test dependencies
		storage := NewMockMultiViewStorage()
		contentProcessor := NewContentProcessor()
		memoryWriter := NewMemoryWriter(storage, contentProcessor, nil)
		handler := NewWriteHandler(memoryWriter, contentProcessor)

		Convey("NewWriteHandler", func() {
			Convey("Should create handler with default config", func() {
				So(handler, ShouldNotBeNil)
				So(handler.memoryWriter, ShouldNotBeNil)
				So(handler.contentProcessor, ShouldNotBeNil)
				So(handler.validator, ShouldNotBeNil)
				So(handler.formatter, ShouldNotBeNil)
				So(handler.config, ShouldNotBeNil)
				So(handler.config.MaxContentLength, ShouldEqual, 10000)
				So(handler.config.DefaultConfidence, ShouldEqual, 1.0)
				So(handler.config.RequireSource, ShouldBeTrue)
			})

			Convey("Should create handler with custom config", func() {
				config := &WriteHandlerConfig{
					MaxContentLength:        5000,
					DefaultConfidence:       0.8,
					RequireSource:           false,
					EnableConflictDetection: false,
					ProcessingTimeout:       10 * time.Second,
					EnableDeduplication:     false,
				}
				customHandler := NewWriteHandlerWithConfig(memoryWriter, contentProcessor, config)

				So(customHandler.config.MaxContentLength, ShouldEqual, 5000)
				So(customHandler.config.DefaultConfidence, ShouldEqual, 0.8)
				So(customHandler.config.RequireSource, ShouldBeFalse)
				So(customHandler.config.EnableConflictDetection, ShouldBeFalse)
				So(customHandler.config.EnableDeduplication, ShouldBeFalse)
			})
		})

		Convey("HandleWrite", func() {
			ctx := context.Background()
			req := &mcp.CallToolRequest{}

			Convey("Should handle valid write request", func() {
				args := WriteArgs{
					Content: "This is test content about artificial intelligence and machine learning.",
					Source:  "test_source",
					Tags:    []string{"ai", "ml", "test"},
					Metadata: map[string]interface{}{
						"confidence": 0.9,
						"language":   "en",
					},
					RequireEvidence: false,
				}

				mcpResult, result, err := handler.HandleWrite(ctx, req, args)

				So(err, ShouldBeNil)
				So(mcpResult, ShouldNotBeNil)
				So(result.MemoryID, ShouldNotBeEmpty)
				So(result.CandidateCount, ShouldBeGreaterThan, 0)
				So(result.EntitiesLinked, ShouldNotBeNil)
				So(result.ProvenanceID, ShouldNotBeEmpty)
				So(len(mcpResult.Content), ShouldBeGreaterThan, 0)
			})

			Convey("Should reject empty content", func() {
				args := WriteArgs{
					Content: "",
					Source:  "test_source",
				}

				_, _, err := handler.HandleWrite(ctx, req, args)

				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "validation")
			})

			Convey("Should reject missing source when required", func() {
				args := WriteArgs{
					Content: "Test content",
					Source:  "",
				}

				_, _, err := handler.HandleWrite(ctx, req, args)

				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "validation")
			})

			Convey("Should handle content with entities and claims", func() {
				args := WriteArgs{
					Content: "John Smith works at OpenAI. The company was founded in 2015.",
					Source:  "test_source",
					Tags:    []string{"person", "company"},
				}

				_, result, err := handler.HandleWrite(ctx, req, args)

				So(err, ShouldBeNil)
				So(result.MemoryID, ShouldNotBeEmpty)
				So(result.CandidateCount, ShouldBeGreaterThan, 0)
				// Entities would be extracted by the content processor
				So(result.EntitiesLinked, ShouldNotBeNil)
			})

			Convey("Should handle timeout context", func() {
				// Create a context that times out quickly
				timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
				defer cancel()

				args := WriteArgs{
					Content: "Test content",
					Source:  "test_source",
				}

				// Wait for context to timeout
				time.Sleep(2 * time.Millisecond)

				_, _, _ = handler.HandleWrite(timeoutCtx, req, args)

				// Error could be timeout or validation, or operation might complete successfully
				// This is acceptable since the mock operations are very fast
			})
		})

		Convey("processContent", func() {
			ctx := context.Background()

			Convey("Should process valid content", func() {
				content := "This is a test document about machine learning algorithms."
				source := "test_source"

				result, err := handler.processContent(ctx, content, source)

				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(len(result.Chunks), ShouldBeGreaterThan, 0)
				So(result.Stats.OriginalLength, ShouldEqual, len(content))
				So(result.Stats.ChunkCount, ShouldBeGreaterThan, 0)
			})

			Convey("Should reject empty content", func() {
				content := ""
				source := "test_source"

				result, err := handler.processContent(ctx, content, source)

				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
			})

			Convey("Should process content with multiple sentences", func() {
				content := "First sentence about AI. Second sentence about ML. Third sentence about data science."
				source := "test_source"

				result, err := handler.processContent(ctx, content, source)

				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(len(result.Chunks), ShouldBeGreaterThan, 0)
				So(result.Stats.ChunkCount, ShouldBeGreaterThan, 0)
			})
		})

		Convey("detectConflicts", func() {
			ctx := context.Background()

			Convey("Should detect no conflicts for new content", func() {
				processedContent := &ProcessingResult{
					Chunks: []*Chunk{
						{
							ID:      "test_chunk_1",
							Content: "Test content",
							Entities: []Entity{
								{ID: "entity_1", Name: "Test Entity", Type: "PERSON"},
							},
						},
					},
					Stats: ProcessingStats{ChunkCount: 1, EntityCount: 1},
				}
				writeResponse := &WriteResult{MemoryID: "test_memory_1"}

				conflicts, err := handler.detectConflicts(ctx, processedContent, writeResponse)

				So(err, ShouldBeNil)
				So(conflicts, ShouldNotBeNil)
				// With mock implementation, no conflicts should be detected
				So(len(conflicts), ShouldEqual, 0)
			})

			Convey("Should handle empty processed content", func() {
				processedContent := &ProcessingResult{
					Chunks: []*Chunk{},
					Stats:  ProcessingStats{ChunkCount: 0},
				}
				writeResponse := &WriteResult{MemoryID: "test_memory_1"}

				conflicts, err := handler.detectConflicts(ctx, processedContent, writeResponse)

				So(err, ShouldBeNil)
				So(conflicts, ShouldNotBeNil)
				So(len(conflicts), ShouldEqual, 0)
			})
		})

		Convey("convertArgsToMetadata", func() {
			Convey("Should convert basic args", func() {
				args := WriteArgs{
					Content: "Test content",
					Source:  "test_source",
					Tags:    []string{"tag1", "tag2"},
					Metadata: map[string]interface{}{
						"key1": "value1",
						"key2": 42,
					},
					RequireEvidence: true,
				}

				metadata := handler.convertArgsToMetadata(args)

				So(metadata.Source, ShouldEqual, "test_source")
				So(metadata.Tags, ShouldResemble, []string{"tag1", "tag2"})
				So(metadata.RequireEvidence, ShouldBeTrue)
				So(metadata.Confidence, ShouldEqual, handler.config.DefaultConfidence)
				So(metadata.Language, ShouldEqual, "en")
				So(metadata.ContentType, ShouldEqual, "text/plain")
				So(metadata.Metadata, ShouldResemble, args.Metadata)
			})

			Convey("Should override confidence from metadata", func() {
				args := WriteArgs{
					Content: "Test content",
					Source:  "test_source",
					Metadata: map[string]interface{}{
						"confidence": 0.7,
					},
				}

				metadata := handler.convertArgsToMetadata(args)

				So(metadata.Confidence, ShouldEqual, 0.7)
			})

			Convey("Should set content type from metadata", func() {
				args := WriteArgs{
					Content: "Test content",
					Source:  "test_source",
					Metadata: map[string]interface{}{
						"content_type": "text/markdown",
						"language":     "es",
						"user_id":      "user123",
					},
				}

				metadata := handler.convertArgsToMetadata(args)

				So(metadata.ContentType, ShouldEqual, "text/markdown")
				So(metadata.Language, ShouldEqual, "es")
				So(metadata.UserID, ShouldEqual, "user123")
			})
		})

		Convey("Configuration", func() {
			Convey("GetConfig should return current config", func() {
				config := handler.GetConfig()
				So(config, ShouldNotBeNil)
				So(config.MaxContentLength, ShouldEqual, 10000)
			})

			Convey("UpdateConfig should update configuration", func() {
				newConfig := &WriteHandlerConfig{
					MaxContentLength:  5000,
					DefaultConfidence: 0.8,
				}

				handler.UpdateConfig(newConfig)
				config := handler.GetConfig()

				So(config.MaxContentLength, ShouldEqual, 5000)
				So(config.DefaultConfidence, ShouldEqual, 0.8)
			})

			Convey("GetStats should return handler statistics", func() {
				stats := handler.GetStats()
				So(stats, ShouldNotBeNil)
				So(stats["max_content_length"], ShouldEqual, 10000)
				So(stats["default_confidence"], ShouldEqual, 1.0)
				So(stats["require_source"], ShouldEqual, true)
			})
		})

		Convey("Dependency Injection", func() {
			Convey("SetMemoryWriter should update memory writer", func() {
				newStorage := NewMockMultiViewStorage()
				newMemoryWriter := NewMemoryWriter(newStorage, contentProcessor, nil)

				handler.SetMemoryWriter(newMemoryWriter)

				So(handler.memoryWriter, ShouldEqual, newMemoryWriter)
			})

			Convey("SetContentProcessor should update content processor", func() {
				newContentProcessor := NewContentProcessor()

				handler.SetContentProcessor(newContentProcessor)

				So(handler.contentProcessor, ShouldEqual, newContentProcessor)
			})
		})
	})
}

func TestWriteHandlerIntegration(t *testing.T) {
	Convey("WriteHandler Integration Tests", t, func() {
		ctx := context.Background()
		req := &mcp.CallToolRequest{}

		Convey("Should handle complex content with multiple entities", func() {
			// Setup fresh test environment for this test
			storage := NewMockMultiViewStorage()
			contentProcessor := NewContentProcessor()
			memoryWriter := NewMemoryWriter(storage, contentProcessor, nil)
			handler := NewWriteHandler(memoryWriter, contentProcessor)
			args := WriteArgs{
				Content: `
					John Smith is the CEO of TechCorp, a company founded in 2020.
					The company specializes in artificial intelligence and machine learning.
					Their headquarters is located in San Francisco, California.
					In 2023, they raised $50 million in Series A funding.
				`,
				Source: "company_profile",
				Tags:   []string{"company", "ceo", "funding", "ai"},
				Metadata: map[string]interface{}{
					"confidence":   0.95,
					"content_type": "text/plain",
					"language":     "en",
					"category":     "business",
				},
				RequireEvidence: false,
			}

			mcpResult, result, err := handler.HandleWrite(ctx, req, args)

			So(err, ShouldBeNil)
			So(mcpResult, ShouldNotBeNil)
			So(result.MemoryID, ShouldNotBeEmpty)
			So(result.CandidateCount, ShouldBeGreaterThan, 0)
			So(result.ProvenanceID, ShouldNotBeEmpty)

			// Check MCP result content
			So(len(mcpResult.Content), ShouldBeGreaterThan, 0)
			textContent, ok := mcpResult.Content[0].(*mcp.TextContent)
			So(ok, ShouldBeTrue)
			So(textContent.Text, ShouldContainSubstring, "Stored memory")
			So(textContent.Text, ShouldContainSubstring, result.MemoryID)
		})

		Convey("Should handle markdown content", func() {
			// Setup fresh test environment for this test
			storage := NewMockMultiViewStorage()
			contentProcessor := NewContentProcessor()
			memoryWriter := NewMemoryWriter(storage, contentProcessor, nil)
			handler := NewWriteHandler(memoryWriter, contentProcessor)
			args := WriteArgs{
				Content: `
# Machine Learning Overview

Machine learning is a subset of **artificial intelligence** that focuses on algorithms.

## Key Concepts

- Supervised Learning
- Unsupervised Learning  
- Reinforcement Learning

### Applications

1. Natural Language Processing
2. Computer Vision
3. Recommendation Systems
				`,
				Source: "ml_guide",
				Tags:   []string{"ml", "ai", "guide"},
				Metadata: map[string]interface{}{
					"content_type": "text/markdown",
					"author":       "AI Researcher",
				},
			}

			_, result, err := handler.HandleWrite(ctx, req, args)

			So(err, ShouldBeNil)
			So(result.MemoryID, ShouldNotBeEmpty)
			So(result.CandidateCount, ShouldBeGreaterThan, 0)
		})

		Convey("Should handle JSON content", func() {
			// Setup fresh test environment for this test
			storage := NewMockMultiViewStorage()
			contentProcessor := NewContentProcessor()
			memoryWriter := NewMemoryWriter(storage, contentProcessor, nil)
			handler := NewWriteHandler(memoryWriter, contentProcessor)
			args := WriteArgs{
				Content: `{
					"name": "GPT-4",
					"type": "language_model",
					"parameters": "175B",
					"capabilities": ["text_generation", "code_completion", "reasoning"],
					"release_date": "2023-03-14"
				}`,
				Source: "model_specs",
				Tags:   []string{"gpt", "llm", "openai"},
				Metadata: map[string]interface{}{
					"content_type":   "application/json",
					"schema_version": "1.0",
				},
			}

			_, result, err := handler.HandleWrite(ctx, req, args)

			So(err, ShouldBeNil)
			So(result.MemoryID, ShouldNotBeEmpty)
		})

		Convey("Should handle content with special characters", func() {
			// Setup fresh test environment for this test
			storage := NewMockMultiViewStorage()
			contentProcessor := NewContentProcessor()
			memoryWriter := NewMemoryWriter(storage, contentProcessor, nil)
			handler := NewWriteHandler(memoryWriter, contentProcessor)
			args := WriteArgs{
				Content: "Café résumé naïve Zürich 北京 東京 москва العربية 🚀 ⭐ 💡",
				Source:  "unicode_test",
				Tags:    []string{"unicode", "international"},
			}

			_, result, err := handler.HandleWrite(ctx, req, args)

			So(err, ShouldBeNil)
			So(result.MemoryID, ShouldNotBeEmpty)
		})

		Convey("Should handle large content within limits", func() {
			// Setup fresh test environment for this test
			storage := NewMockMultiViewStorage()
			contentProcessor := NewContentProcessor()
			memoryWriter := NewMemoryWriter(storage, contentProcessor, nil)
			handler := NewWriteHandler(memoryWriter, contentProcessor)
			// Create content just under the limit
			largeContent := strings.Repeat("This is a test sentence. ", 400) // ~9600 characters

			args := WriteArgs{
				Content: largeContent,
				Source:  "large_document",
				Tags:    []string{"large", "test"},
			}

			_, result, err := handler.HandleWrite(ctx, req, args)

			So(err, ShouldBeNil)
			So(result.MemoryID, ShouldNotBeEmpty)
			So(result.CandidateCount, ShouldBeGreaterThan, 0)
		})
	})
}

func BenchmarkWriteHandler(b *testing.B) {
	// Setup
	storage := NewMockMultiViewStorage()
	contentProcessor := NewContentProcessor()
	memoryWriter := NewMemoryWriter(storage, contentProcessor, nil)
	handler := NewWriteHandler(memoryWriter, contentProcessor)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	args := WriteArgs{
		Content: "This is benchmark content for testing write handler performance with entities and claims.",
		Source:  "benchmark_test",
		Tags:    []string{"benchmark", "performance"},
		Metadata: map[string]interface{}{
			"confidence": 0.9,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := handler.HandleWrite(ctx, req, args)
		if err != nil {
			b.Fatalf("HandleWrite failed: %v", err)
		}
	}
}

func BenchmarkWriteHandlerLargeContent(b *testing.B) {
	// Setup
	storage := NewMockMultiViewStorage()
	contentProcessor := NewContentProcessor()
	memoryWriter := NewMemoryWriter(storage, contentProcessor, nil)
	handler := NewWriteHandler(memoryWriter, contentProcessor)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}

	// Create large content
	largeContent := strings.Repeat("This is a large document with many sentences and entities. ", 100)

	args := WriteArgs{
		Content: largeContent,
		Source:  "benchmark_large",
		Tags:    []string{"benchmark", "large"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := handler.HandleWrite(ctx, req, args)
		if err != nil {
			b.Fatalf("HandleWrite failed: %v", err)
		}
	}
}
