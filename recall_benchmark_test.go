package main

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func BenchmarkRecallHandler(b *testing.B) {
	// Setup
	queryProcessor := NewQueryProcessor(nil)
	resultFuser := NewResultFuser()
	handler := NewRecallHandler(queryProcessor, resultFuser)
	
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	
	args := RecallArgs{
		Query:      "benchmark test query",
		MaxResults: 10,
		TimeBudget: 5000,
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, _, err := handler.HandleRecall(ctx, req, args)
		if err != nil {
			b.Fatalf("HandleRecall failed: %v", err)
		}
	}
}

func BenchmarkRecallHandlerConcurrent(b *testing.B) {
	// Setup
	queryProcessor := NewQueryProcessor(nil)
	resultFuser := NewResultFuser()
	handler := NewRecallHandler(queryProcessor, resultFuser)
	
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	
	args := RecallArgs{
		Query:      "concurrent benchmark test query",
		MaxResults: 10,
		TimeBudget: 5000,
	}

	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, err := handler.HandleRecall(ctx, req, args)
			if err != nil {
				b.Fatalf("HandleRecall failed: %v", err)
			}
		}
	})
}

func BenchmarkRecallHandlerVariousQuerySizes(b *testing.B) {
	queryProcessor := NewQueryProcessor(nil)
	resultFuser := NewResultFuser()
	handler := NewRecallHandler(queryProcessor, resultFuser)
	
	ctx := context.Background()
	req := &mcp.CallToolRequest{}

	// Test different query sizes
	querySizes := []struct {
		name  string
		query string
	}{
		{"Short", "AI"},
		{"Medium", "artificial intelligence machine learning"},
		{"Long", "artificial intelligence machine learning deep neural networks natural language processing computer vision"},
		{"VeryLong", "artificial intelligence machine learning deep neural networks natural language processing computer vision reinforcement learning generative adversarial networks transformer architectures attention mechanisms"},
	}

	for _, qs := range querySizes {
		b.Run(qs.name, func(b *testing.B) {
			args := RecallArgs{
				Query:      qs.query,
				MaxResults: 10,
				TimeBudget: 5000,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := handler.HandleRecall(ctx, req, args)
				if err != nil {
					b.Fatalf("HandleRecall failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkRecallHandlerVariousResultCounts(b *testing.B) {
	queryProcessor := NewQueryProcessor(nil)
	resultFuser := NewResultFuser()
	handler := NewRecallHandler(queryProcessor, resultFuser)
	
	ctx := context.Background()
	req := &mcp.CallToolRequest{}

	// Test different result counts
	resultCounts := []int{1, 5, 10, 25, 50, 100}

	for _, count := range resultCounts {
		b.Run("Results"+string(rune('0'+count/100))+string(rune('0'+(count%100)/10))+string(rune('0'+count%10)), func(b *testing.B) {
			args := RecallArgs{
				Query:      "benchmark test query",
				MaxResults: count,
				TimeBudget: 10000, // Longer timeout for larger result sets
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := handler.HandleRecall(ctx, req, args)
				if err != nil {
					b.Fatalf("HandleRecall failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkRecallHandlerWithFilters(b *testing.B) {
	queryProcessor := NewQueryProcessor(nil)
	resultFuser := NewResultFuser()
	handler := NewRecallHandler(queryProcessor, resultFuser)
	
	ctx := context.Background()
	req := &mcp.CallToolRequest{}

	// Test with various filter complexities
	filterTests := []struct {
		name    string
		filters map[string]interface{}
	}{
		{
			"NoFilters",
			nil,
		},
		{
			"SimpleFilter",
			map[string]interface{}{
				"source": "test",
			},
		},
		{
			"MultipleFilters",
			map[string]interface{}{
				"source":     "test",
				"confidence": 0.8,
				"type":       "research",
			},
		},
		{
			"ComplexFilters",
			map[string]interface{}{
				"source":     "research_papers",
				"confidence": 0.7,
				"type":       "academic",
				"date":       "2024",
				"tag":        "machine_learning",
			},
		},
	}

	for _, ft := range filterTests {
		b.Run(ft.name, func(b *testing.B) {
			args := RecallArgs{
				Query:      "benchmark test query with filters",
				MaxResults: 10,
				TimeBudget: 5000,
				Filters:    ft.filters,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := handler.HandleRecall(ctx, req, args)
				if err != nil {
					b.Fatalf("HandleRecall failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkRecallValidator(b *testing.B) {
	validator := NewRecallArgsValidator()
	
	args := RecallArgs{
		Query:      "validation benchmark test query",
		MaxResults: 10,
		TimeBudget: 5000,
		Filters:    map[string]interface{}{"source": "test"},
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err := validator.Validate(args)
		if err != nil {
			b.Fatalf("Validate failed: %v", err)
		}
	}
}

func BenchmarkRecallValidatorDetailed(b *testing.B) {
	validator := NewRecallArgsValidator()
	
	args := RecallArgs{
		Query:      "detailed validation benchmark test query",
		MaxResults: 10,
		TimeBudget: 5000,
		Filters:    map[string]interface{}{"source": "test", "confidence": 0.8},
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		result := validator.ValidateDetailed(args)
		if !result.Valid {
			b.Fatalf("ValidateDetailed failed: %v", result.Errors)
		}
	}
}

func BenchmarkRecallValidatorSanitize(b *testing.B) {
	validator := NewRecallArgsValidator()
	
	args := RecallArgs{
		Query:      "  sanitization benchmark test query  ",
		MaxResults: 10,
		TimeBudget: 5000,
		Filters:    map[string]interface{}{"source": "test"},
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := validator.SanitizeInput(args)
		if err != nil {
			b.Fatalf("SanitizeInput failed: %v", err)
		}
	}
}

func BenchmarkRecallFormatter(b *testing.B) {
	formatter := NewRecallResponseFormatter()
	
	response := &RecallResponse{
		Evidence: []Evidence{
			{
				Content:     "Benchmark test evidence content",
				Source:      "test_source",
				Confidence:  0.8,
				WhySelected: "High relevance score",
				RelationMap: map[string]string{"related_to": "entity1"},
				Provenance: ProvenanceInfo{
					Source:    "test_source",
					Timestamp: "2024-01-01T00:00:00Z",
					Version:   "1.0",
				},
			},
		},
		RetrievalStats: RetrievalStats{
			QueryTime:     100,
			VectorResults: 1,
			FusionScore:   0.8,
		},
	}
	
	args := RecallArgs{Query: "benchmark test query"}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := formatter.Format(response, args)
		if err != nil {
			b.Fatalf("Format failed: %v", err)
		}
	}
}

func BenchmarkRecallFormatterLargeEvidence(b *testing.B) {
	formatter := NewRecallResponseFormatter()
	
	// Create a response with many evidence items
	evidence := make([]Evidence, 100)
	for i := range evidence {
		evidence[i] = Evidence{
			Content:     "Benchmark test evidence content number " + string(rune('0'+i%10)),
			Source:      "test_source_" + string(rune('0'+i%10)),
			Confidence:  0.8 - float64(i%10)*0.05,
			WhySelected: "Relevance score for item " + string(rune('0'+i%10)),
			RelationMap: map[string]string{
				"related_to": "entity" + string(rune('0'+i%10)),
				"type":       "test",
			},
			Provenance: ProvenanceInfo{
				Source:    "test_source_" + string(rune('0'+i%10)),
				Timestamp: "2024-01-01T00:00:00Z",
				Version:   "1.0",
			},
		}
	}
	
	response := &RecallResponse{
		Evidence: evidence,
		RetrievalStats: RetrievalStats{
			QueryTime:       500,
			VectorResults:   50,
			GraphResults:    30,
			SearchResults:   20,
			FusionScore:     0.75,
			TotalCandidates: 100,
		},
	}
	
	args := RecallArgs{Query: "benchmark test query"}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := formatter.Format(response, args)
		if err != nil {
			b.Fatalf("Format failed: %v", err)
		}
	}
}

func BenchmarkRecallE2EServer(b *testing.B) {
	// Setup complete server
	config := &ServerConfig{
		Storage: MultiViewStorageConfig{
			VectorStore: VectorStoreConfig{
				Provider:   "file",
				Dimensions: 384,
			},
		},
	}
	
	server, err := NewAgenticMemoryServer(config)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	
	args := RecallArgs{
		Query:      "end-to-end benchmark test query",
		MaxResults: 10,
		TimeBudget: 5000,
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, _, err := server.handleRecall(ctx, req, args)
		if err != nil {
			b.Fatalf("handleRecall failed: %v", err)
		}
	}
}

func BenchmarkRecallE2EServerConcurrent(b *testing.B) {
	// Setup complete server
	config := &ServerConfig{
		Storage: MultiViewStorageConfig{
			VectorStore: VectorStoreConfig{
				Provider:   "file",
				Dimensions: 384,
			},
		},
	}
	
	server, err := NewAgenticMemoryServer(config)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	
	args := RecallArgs{
		Query:      "concurrent end-to-end benchmark test query",
		MaxResults: 10,
		TimeBudget: 5000,
	}

	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, err := server.handleRecall(ctx, req, args)
			if err != nil {
				b.Fatalf("handleRecall failed: %v", err)
			}
		}
	})
}

func BenchmarkRecallMemoryAllocation(b *testing.B) {
	queryProcessor := NewQueryProcessor(nil)
	resultFuser := NewResultFuser()
	handler := NewRecallHandler(queryProcessor, resultFuser)
	
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	
	args := RecallArgs{
		Query:      "memory allocation benchmark test query",
		MaxResults: 10,
		TimeBudget: 5000,
	}

	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, _, err := handler.HandleRecall(ctx, req, args)
		if err != nil {
			b.Fatalf("HandleRecall failed: %v", err)
		}
	}
}

func BenchmarkRecallWithTimeout(b *testing.B) {
	queryProcessor := NewQueryProcessor(nil)
	resultFuser := NewResultFuser()
	handler := NewRecallHandler(queryProcessor, resultFuser)
	
	req := &mcp.CallToolRequest{}
	
	// Test with various timeout scenarios
	timeouts := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		5 * time.Second,
	}

	for _, timeout := range timeouts {
		b.Run("Timeout"+timeout.String(), func(b *testing.B) {
			args := RecallArgs{
				Query:      "timeout benchmark test query",
				MaxResults: 10,
				TimeBudget: int(timeout.Milliseconds()),
			}

			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), timeout+100*time.Millisecond)
				_, _, err := handler.HandleRecall(ctx, req, args)
				cancel()
				
				// Don't fail on timeout errors in benchmark
				if err != nil && err.Error() != "context deadline exceeded" {
					b.Fatalf("HandleRecall failed: %v", err)
				}
			}
		})
	}
}