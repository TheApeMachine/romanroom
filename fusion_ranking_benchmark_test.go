package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// BenchmarkResultFusion benchmarks the result fusion process
func BenchmarkResultFusion(b *testing.B) {
	fuser := NewResultFuser()
	ctx := context.Background()

	// Create test data sets of different sizes
	testSizes := []int{10, 50, 100, 500}

	for _, size := range testSizes {
		b.Run(fmt.Sprintf("FuseResults_%d", size), func(b *testing.B) {
			vectorResults := make([]VectorSearchResult, size)
			keywordResults := make([]KeywordSearchResult, size/2) // Partial overlap

			// Generate test data
			for i := 0; i < size; i++ {
				vectorResults[i] = VectorSearchResult{
					ID:       fmt.Sprintf("vec_doc_%d", i),
					Content:  fmt.Sprintf("Vector document %d about machine learning and AI", i),
					Score:    0.9 - float64(i)*0.001,
					Metadata: map[string]interface{}{"type": "vector", "index": i},
				}
			}

			for i := 0; i < size/2; i++ {
				keywordResults[i] = KeywordSearchResult{
					ID:           fmt.Sprintf("vec_doc_%d", i*2), // Create overlap
					Content:      fmt.Sprintf("Keyword document %d about machine learning", i*2),
					Score:        0.85 - float64(i)*0.001,
					MatchedTerms: []string{"machine", "learning"},
					Metadata:     map[string]interface{}{"type": "keyword", "index": i * 2},
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := fuser.FuseVectorAndKeyword(ctx, vectorResults, keywordResults)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkRRFAlgorithm benchmarks the RRF algorithm specifically
func BenchmarkRRFAlgorithm(b *testing.B) {
	fuser := NewResultFuser()

	testSizes := []int{10, 50, 100, 500}

	for _, size := range testSizes {
		b.Run(fmt.Sprintf("RRF_%d", size), func(b *testing.B) {
			// Create fusion inputs
			inputs := []FusionInput{
				{
					Method: "vector",
					Weight: 1.0,
					Results: make([]FusionItem, size),
				},
				{
					Method: "keyword",
					Weight: 1.0,
					Results: make([]FusionItem, size),
				},
			}

			// Generate test data
			for i := 0; i < size; i++ {
				inputs[0].Results[i] = FusionItem{
					ID:      fmt.Sprintf("doc_%d", i),
					Content: fmt.Sprintf("Document %d", i),
					Score:   0.9 - float64(i)*0.001,
					Rank:    i + 1,
				}
				inputs[1].Results[i] = FusionItem{
					ID:      fmt.Sprintf("doc_%d", (size-1-i)), // Reverse order
					Content: fmt.Sprintf("Document %d", (size - 1 - i)),
					Score:   0.8 - float64(i)*0.001,
					Rank:    i + 1,
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = fuser.RRF(inputs)
			}
		})
	}
}

// BenchmarkResultRanking benchmarks the result ranking process
func BenchmarkResultRanking(b *testing.B) {
	ranker := NewResultRanker()
	ctx := context.Background()

	testSizes := []int{10, 50, 100, 500}

	for _, size := range testSizes {
		b.Run(fmt.Sprintf("RankResults_%d", size), func(b *testing.B) {
			now := time.Now()
			results := make([]RankableResult, size)

			// Generate test data
			for i := 0; i < size; i++ {
				results[i] = RankableResult{
					ID:        fmt.Sprintf("doc_%d", i),
					Content:   fmt.Sprintf("This is document %d about machine learning and artificial intelligence systems with comprehensive coverage of algorithms and applications", i),
					BaseScore: 0.9 - float64(i)*0.001,
					Source:    fmt.Sprintf("source_%d", i%10),
					Timestamp: now.Add(-time.Duration(i) * time.Hour),
					Metadata: map[string]interface{}{
						"title":          fmt.Sprintf("Document %d Title", i),
						"authority_score": 0.8 - float64(i%10)*0.05,
						"quality_score":   0.7 + float64(i%5)*0.05,
					},
				}
			}

			context := &RankingContext{
				Query:       "machine learning algorithms",
				TimeContext: now,
				UserPreferences: map[string]interface{}{
					"topics":  []string{"machine learning", "AI"},
					"sources": []string{"source_0", "source_1"},
				},
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := ranker.Rank(ctx, results, context)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDiversityCalculation benchmarks diversity score calculation
func BenchmarkDiversityCalculation(b *testing.B) {
	ranker := NewResultRanker()

	testSizes := []int{10, 25, 50, 100}

	for _, size := range testSizes {
		b.Run(fmt.Sprintf("Diversity_%d", size), func(b *testing.B) {
			results := make([]RankableResult, size)

			// Generate test data with varying similarity
			for i := 0; i < size; i++ {
				var content string
				if i%3 == 0 {
					content = "Machine learning algorithms and neural networks for artificial intelligence"
				} else if i%3 == 1 {
					content = "Deep learning systems and computer vision applications in modern AI"
				} else {
					content = fmt.Sprintf("Unique document %d about different topics and specialized content", i)
				}

				results[i] = RankableResult{
					ID:      fmt.Sprintf("doc_%d", i),
					Content: content,
				}
			}

			context := &RankingContext{}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				ranker.calculateDiversityScores(results, context)
			}
		})
	}
}

// BenchmarkEvidenceAssembly benchmarks evidence assembly process
func BenchmarkEvidenceAssembly(b *testing.B) {
	assembler := NewEvidenceAssembler()
	ctx := context.Background()

	testSizes := []int{10, 50, 100, 500}

	for _, size := range testSizes {
		b.Run(fmt.Sprintf("AssembleEvidence_%d", size), func(b *testing.B) {
			now := time.Now()
			inputs := make([]AssemblyInput, size)

			// Generate test data
			for i := 0; i < size; i++ {
				inputs[i] = AssemblyInput{
					ID:      fmt.Sprintf("doc_%d", i),
					Content: fmt.Sprintf("This is comprehensive document %d about machine learning algorithms and their applications in artificial intelligence systems with detailed explanations and examples", i),
					Score:   0.9 - float64(i)*0.001,
					Source:  fmt.Sprintf("source_%d", i%10),
					Timestamp: now.Add(-time.Duration(i) * time.Hour),
					MatchedTerms: []string{"machine", "learning", "algorithms"},
					RelatedEntities: []string{
						fmt.Sprintf("entity_%d", i%5),
						fmt.Sprintf("concept_%d", i%3),
					},
					Metadata: map[string]interface{}{
						"author":   fmt.Sprintf("Author %d", i%20),
						"category": fmt.Sprintf("Category %d", i%5),
						"topic":    "machine_learning",
					},
				}
			}

			context := &AssemblyContext{
				Query:           "machine learning algorithms",
				QueryTerms:      []string{"machine", "learning", "algorithms"},
				RequestTime:     now,
				RetrievalMethod: "benchmark_test",
				GraphContext: &GraphContext{
					QueryEntities:   []string{"machine_learning", "algorithms"},
					RelatedEntities: []string{"neural_networks", "deep_learning"},
				},
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := assembler.Assemble(ctx, inputs, context)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkContentSimilarity benchmarks content similarity calculation
func BenchmarkContentSimilarity(b *testing.B) {
	assembler := NewEvidenceAssembler()

	// Test different content lengths
	contentLengths := []int{50, 200, 500, 1000}

	for _, length := range contentLengths {
		b.Run(fmt.Sprintf("Similarity_%d_chars", length), func(b *testing.B) {
			// Generate test content
			baseContent := "machine learning algorithms artificial intelligence neural networks deep learning"
			content1 := ""
			content2 := ""

			for len(content1) < length {
				content1 += baseContent + " "
			}
			content1 = content1[:length]

			for len(content2) < length {
				content2 += baseContent + " systems applications " // Slightly different
			}
			content2 = content2[:length]

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = assembler.calculateContentSimilarity(content1, content2)
			}
		})
	}
}

// BenchmarkDeduplication benchmarks content deduplication
func BenchmarkDeduplication(b *testing.B) {
	assembler := NewEvidenceAssembler()

	testSizes := []int{10, 50, 100, 200}

	for _, size := range testSizes {
		b.Run(fmt.Sprintf("Deduplication_%d", size), func(b *testing.B) {
			inputs := make([]AssemblyInput, size)

			// Generate test data with some duplicates
			for i := 0; i < size; i++ {
				var content string
				if i%5 == 0 {
					content = "This is a duplicate document about machine learning algorithms"
				} else if i%7 == 0 {
					content = "Another duplicate document about neural networks and deep learning"
				} else {
					content = fmt.Sprintf("Unique document %d about artificial intelligence and machine learning applications", i)
				}

				inputs[i] = AssemblyInput{
					ID:      fmt.Sprintf("doc_%d", i),
					Content: content,
					Score:   0.8,
					Source:  "test_source",
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = assembler.deduplicateInputs(inputs)
			}
		})
	}
}

// BenchmarkEndToEndPipeline benchmarks the complete fusion-ranking-assembly pipeline
func BenchmarkEndToEndPipeline(b *testing.B) {
	fuser := NewResultFuser()
	ranker := NewResultRanker()
	assembler := NewEvidenceAssembler()
	ctx := context.Background()

	testSizes := []int{10, 50, 100}

	for _, size := range testSizes {
		b.Run(fmt.Sprintf("EndToEnd_%d", size), func(b *testing.B) {
			now := time.Now()

			// Generate test data
			vectorResults := make([]VectorSearchResult, size)
			keywordResults := make([]KeywordSearchResult, size/2)

			for i := 0; i < size; i++ {
				vectorResults[i] = VectorSearchResult{
					ID:       fmt.Sprintf("doc_%d", i),
					Content:  fmt.Sprintf("Vector document %d about machine learning", i),
					Score:    0.9 - float64(i)*0.001,
					Metadata: map[string]interface{}{"type": "vector"},
				}
			}

			for i := 0; i < size/2; i++ {
				keywordResults[i] = KeywordSearchResult{
					ID:           fmt.Sprintf("doc_%d", i*2),
					Content:      fmt.Sprintf("Keyword document %d about machine learning", i*2),
					Score:        0.85 - float64(i)*0.001,
					MatchedTerms: []string{"machine", "learning"},
					Metadata:     map[string]interface{}{"type": "keyword"},
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Step 1: Fusion
				fusionResponse, err := fuser.FuseVectorAndKeyword(ctx, vectorResults, keywordResults)
				if err != nil {
					b.Fatal(err)
				}

				// Step 2: Ranking
				rankableResults := make([]RankableResult, len(fusionResponse.Results))
				for j, result := range fusionResponse.Results {
					rankableResults[j] = RankableResult{
						ID:        result.ID,
						Content:   result.Content,
						BaseScore: result.FinalScore,
						Source:    "test_source",
						Timestamp: now.Add(-time.Duration(j) * time.Hour),
						Metadata:  result.Metadata,
					}
				}

				rankingContext := &RankingContext{
					Query:       "machine learning",
					TimeContext: now,
				}

				rankingResponse, err := ranker.Rank(ctx, rankableResults, rankingContext)
				if err != nil {
					b.Fatal(err)
				}

				// Step 3: Evidence Assembly
				assemblyInputs := make([]AssemblyInput, len(rankingResponse.Results))
				for j, result := range rankingResponse.Results {
					assemblyInputs[j] = AssemblyInput{
						ID:        result.ID,
						Content:   result.Content,
						Score:     result.FinalScore,
						Source:    result.Source,
						Timestamp: result.Timestamp,
						Metadata:  result.Metadata,
					}
				}

				assemblyContext := &AssemblyContext{
					Query:           "machine learning",
					RequestTime:     now,
					RetrievalMethod: "benchmark",
				}

				_, err = assembler.Assemble(ctx, assemblyInputs, assemblyContext)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkMemoryUsage benchmarks memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("MemoryUsage", func(b *testing.B) {
		fuser := NewResultFuser()
		ctx := context.Background()

		// Large dataset
		size := 1000
		vectorResults := make([]VectorSearchResult, size)
		keywordResults := make([]KeywordSearchResult, size)

		for i := 0; i < size; i++ {
			vectorResults[i] = VectorSearchResult{
				ID:      fmt.Sprintf("vec_doc_%d", i),
				Content: fmt.Sprintf("Vector document %d with comprehensive content about machine learning algorithms and artificial intelligence systems", i),
				Score:   0.9 - float64(i)*0.0001,
				Metadata: map[string]interface{}{
					"type":   "vector",
					"index":  i,
					"tags":   []string{"ml", "ai", "algorithms"},
					"scores": map[string]float64{"relevance": 0.8, "quality": 0.9},
				},
			}
			keywordResults[i] = KeywordSearchResult{
				ID:           fmt.Sprintf("key_doc_%d", i),
				Content:      fmt.Sprintf("Keyword document %d about machine learning and neural networks", i),
				Score:        0.85 - float64(i)*0.0001,
				MatchedTerms: []string{"machine", "learning", "neural", "networks"},
				Metadata: map[string]interface{}{
					"type":     "keyword",
					"index":    i,
					"matches":  4,
					"bm25":     0.75,
				},
			}
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := fuser.FuseVectorAndKeyword(ctx, vectorResults, keywordResults)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}