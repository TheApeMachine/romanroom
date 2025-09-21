package main

import (
	"context"
	"math"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestVectorSearcher(t *testing.T) {
	Convey("Given a VectorSearcher", t, func() {
		vs := NewVectorSearcher()

		Convey("When creating a new VectorSearcher", func() {
			So(vs, ShouldNotBeNil)
			So(vs.config, ShouldNotBeNil)
			So(vs.config.DefaultK, ShouldEqual, 10)
			So(vs.config.MinSimilarity, ShouldEqual, 0.1)
		})

		Convey("When creating with custom config", func() {
			config := &VectorSearchConfig{
				DefaultK:        5,
				MinSimilarity:   0.5,
				MaxResults:      50,
				NormalizeScores: false,
			}
			vs := NewVectorSearcherWithStore(nil, config)

			So(vs.config.DefaultK, ShouldEqual, 5)
			So(vs.config.MinSimilarity, ShouldEqual, 0.5)
			So(vs.config.MaxResults, ShouldEqual, 50)
			So(vs.config.NormalizeScores, ShouldBeFalse)
		})
	})
}

func TestVectorSearcherSearch(t *testing.T) {
	Convey("Given a VectorSearcher with mock store", t, func() {
		mockStore := NewMockVectorStore()
		vs := NewVectorSearcherWithStore(mockStore, nil)

		// Add some test data
		embedding1 := []float32{1.0, 0.0, 0.0}
		embedding2 := []float32{0.0, 1.0, 0.0}
		embedding3 := []float32{0.0, 0.0, 1.0}

		ctx := context.Background()
		mockStore.Store(ctx, "doc1", embedding1, map[string]interface{}{"content": "Document 1"})
		mockStore.Store(ctx, "doc2", embedding2, map[string]interface{}{"content": "Document 2"})
		mockStore.Store(ctx, "doc3", embedding3, map[string]interface{}{"content": "Document 3"})

		Convey("When searching with valid embedding", func() {
			ctx := context.Background()
			queryEmbedding := []float32{1.0, 0.0, 0.0}
			k := 2

			result, err := vs.Search(ctx, queryEmbedding, k, nil)

			Convey("Then it should return search results", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(len(result.Results), ShouldBeGreaterThan, 0)
				So(result.TotalFound, ShouldBeGreaterThan, 0)
				So(result.Metadata, ShouldNotBeNil)
			})
		})

		Convey("When searching with empty embedding", func() {
			ctx := context.Background()
			queryEmbedding := []float32{}
			k := 2

			result, err := vs.Search(ctx, queryEmbedding, k, nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
			})
		})

		Convey("When searching with zero k", func() {
			ctx := context.Background()
			queryEmbedding := []float32{1.0, 0.0, 0.0}
			k := 0

			result, err := vs.Search(ctx, queryEmbedding, k, nil)

			Convey("Then it should use default k", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
			})
		})

		Convey("When searching without vector store", func() {
			vsNoStore := NewVectorSearcher()
			ctx := context.Background()
			queryEmbedding := []float32{1.0, 0.0, 0.0}

			result, err := vsNoStore.Search(ctx, queryEmbedding, 5, nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
			})
		})
	})
}

func TestVectorSearcherCosineSimilarity(t *testing.T) {
	Convey("Given a VectorSearcher", t, func() {
		vs := NewVectorSearcher()

		Convey("When calculating cosine similarity of identical vectors", func() {
			a := []float32{1.0, 2.0, 3.0}
			b := []float32{1.0, 2.0, 3.0}

			similarity := vs.CosineSimilarity(a, b)

			Convey("Then similarity should be 1.0", func() {
				So(similarity, ShouldAlmostEqual, 1.0, 0.0001)
			})
		})

		Convey("When calculating cosine similarity of orthogonal vectors", func() {
			a := []float32{1.0, 0.0, 0.0}
			b := []float32{0.0, 1.0, 0.0}

			similarity := vs.CosineSimilarity(a, b)

			Convey("Then similarity should be 0.0", func() {
				So(similarity, ShouldAlmostEqual, 0.0, 0.0001)
			})
		})

		Convey("When calculating cosine similarity of opposite vectors", func() {
			a := []float32{1.0, 0.0, 0.0}
			b := []float32{-1.0, 0.0, 0.0}

			similarity := vs.CosineSimilarity(a, b)

			Convey("Then similarity should be -1.0", func() {
				So(similarity, ShouldAlmostEqual, -1.0, 0.0001)
			})
		})

		Convey("When calculating cosine similarity of different length vectors", func() {
			a := []float32{1.0, 2.0}
			b := []float32{1.0, 2.0, 3.0}

			similarity := vs.CosineSimilarity(a, b)

			Convey("Then similarity should be 0.0", func() {
				So(similarity, ShouldEqual, 0.0)
			})
		})

		Convey("When calculating cosine similarity of empty vectors", func() {
			a := []float32{}
			b := []float32{}

			similarity := vs.CosineSimilarity(a, b)

			Convey("Then similarity should be 0.0", func() {
				So(similarity, ShouldEqual, 0.0)
			})
		})

		Convey("When calculating cosine similarity of zero vectors", func() {
			a := []float32{0.0, 0.0, 0.0}
			b := []float32{1.0, 2.0, 3.0}

			similarity := vs.CosineSimilarity(a, b)

			Convey("Then similarity should be 0.0", func() {
				So(similarity, ShouldEqual, 0.0)
			})
		})
	})
}

func TestVectorSearcherRankResults(t *testing.T) {
	Convey("Given a VectorSearcher", t, func() {
		vs := NewVectorSearcher()

		Convey("When ranking results by similarity", func() {
			results := []VectorSearchResult{
				{ID: "doc1", Similarity: 0.5, Score: 0.5},
				{ID: "doc2", Similarity: 0.9, Score: 0.9},
				{ID: "doc3", Similarity: 0.3, Score: 0.3},
				{ID: "doc4", Similarity: 0.7, Score: 0.7},
			}

			ranked := vs.RankResults(results)

			Convey("Then results should be sorted by similarity descending", func() {
				So(len(ranked), ShouldEqual, 4)
				So(ranked[0].ID, ShouldEqual, "doc2")
				So(ranked[1].ID, ShouldEqual, "doc4")
				So(ranked[2].ID, ShouldEqual, "doc1")
				So(ranked[3].ID, ShouldEqual, "doc3")
			})
		})

		Convey("When ranking results with identical similarities", func() {
			results := []VectorSearchResult{
				{ID: "doc1", Similarity: 0.5, Score: 0.3},
				{ID: "doc2", Similarity: 0.5, Score: 0.7},
				{ID: "doc3", Similarity: 0.5, Score: 0.5},
			}

			ranked := vs.RankResults(results)

			Convey("Then results should be sorted by score as tiebreaker", func() {
				So(len(ranked), ShouldEqual, 3)
				So(ranked[0].Score, ShouldBeGreaterThanOrEqualTo, ranked[1].Score)
				So(ranked[1].Score, ShouldBeGreaterThanOrEqualTo, ranked[2].Score)
			})
		})

		Convey("When ranking empty results", func() {
			results := []VectorSearchResult{}

			ranked := vs.RankResults(results)

			Convey("Then it should return empty slice", func() {
				So(len(ranked), ShouldEqual, 0)
			})
		})
	})
}

func TestVectorSearcherSearchMultiple(t *testing.T) {
	Convey("Given a VectorSearcher with mock store", t, func() {
		mockStore := NewMockVectorStore()
		vs := NewVectorSearcherWithStore(mockStore, nil)

		// Add test data
		embedding1 := []float32{1.0, 0.0, 0.0}
		embedding2 := []float32{0.0, 1.0, 0.0}
		ctx := context.Background()
		mockStore.Store(ctx, "doc1", embedding1, map[string]interface{}{"content": "Document 1"})
		mockStore.Store(ctx, "doc2", embedding2, map[string]interface{}{"content": "Document 2"})

		Convey("When searching with multiple embeddings", func() {
			ctx := context.Background()
			queryEmbeddings := [][]float32{
				{1.0, 0.0, 0.0},
				{0.0, 1.0, 0.0},
			}
			k := 5

			result, err := vs.SearchMultiple(ctx, queryEmbeddings, k, nil)

			Convey("Then it should return merged results", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(result.Metadata["query_count"], ShouldEqual, 2)
				So(result.Metadata["deduplication"], ShouldBeTrue)
			})
		})

		Convey("When searching with empty embeddings", func() {
			ctx := context.Background()
			queryEmbeddings := [][]float32{}
			k := 5

			result, err := vs.SearchMultiple(ctx, queryEmbeddings, k, nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
			})
		})
	})
}

func TestVectorSearcherHelperMethods(t *testing.T) {
	Convey("Given a VectorSearcher", t, func() {
		vs := NewVectorSearcher()

		Convey("When normalizing a vector", func() {
			vector := []float32{3.0, 4.0, 0.0}
			normalized := vs.normalizeVector(vector)

			Convey("Then it should have unit length", func() {
				var norm float64
				for _, val := range normalized {
					norm += float64(val) * float64(val)
				}
				So(math.Sqrt(norm), ShouldAlmostEqual, 1.0, 0.0001)
			})
		})

		Convey("When normalizing a zero vector", func() {
			vector := []float32{0.0, 0.0, 0.0}
			normalized := vs.normalizeVector(vector)

			Convey("Then it should return the same vector", func() {
				So(normalized, ShouldResemble, vector)
			})
		})

		Convey("When getting similarity matrix", func() {
			vectors := [][]float32{
				{1.0, 0.0, 0.0},
				{0.0, 1.0, 0.0},
				{1.0, 0.0, 0.0},
			}

			matrix := vs.GetSimilarityMatrix(vectors)

			Convey("Then it should return correct similarity matrix", func() {
				So(len(matrix), ShouldEqual, 3)
				So(len(matrix[0]), ShouldEqual, 3)
				So(matrix[0][0], ShouldAlmostEqual, 1.0, 0.0001) // Self-similarity
				So(matrix[0][1], ShouldAlmostEqual, 0.0, 0.0001) // Orthogonal
				So(matrix[0][2], ShouldAlmostEqual, 1.0, 0.0001) // Identical
			})
		})

		Convey("When finding nearest neighbors", func() {
			ctx := context.Background()
			targetEmbedding := []float32{1.0, 0.0, 0.0}
			candidates := []VectorSearchResult{
				{ID: "doc1", Embedding: []float32{1.0, 0.0, 0.0}},
				{ID: "doc2", Embedding: []float32{0.0, 1.0, 0.0}},
				{ID: "doc3", Embedding: []float32{0.7071, 0.7071, 0.0}},
			}

			neighbors := vs.FindNearestNeighbors(ctx, targetEmbedding, candidates, 2)

			Convey("Then it should return k nearest neighbors", func() {
				So(len(neighbors), ShouldEqual, 2)
				So(neighbors[0].ID, ShouldEqual, "doc1") // Most similar
				So(neighbors[0].Similarity, ShouldAlmostEqual, 1.0, 0.0001)
			})
		})

		Convey("When finding nearest neighbors with empty candidates", func() {
			ctx := context.Background()
			targetEmbedding := []float32{1.0, 0.0, 0.0}
			candidates := []VectorSearchResult{}

			neighbors := vs.FindNearestNeighbors(ctx, targetEmbedding, candidates, 2)

			Convey("Then it should return empty results", func() {
				So(len(neighbors), ShouldEqual, 0)
			})
		})
	})
}

// Benchmark tests
func BenchmarkVectorSearcherCosineSimilarity(b *testing.B) {
	vs := NewVectorSearcher()
	a := make([]float32, 1536) // Common embedding dimension
	vectorB := make([]float32, 1536)
	
	// Initialize with random values
	for i := range a {
		a[i] = float32(i) / 1536.0
		vectorB[i] = float32(1536-i) / 1536.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs.CosineSimilarity(a, vectorB)
	}
}

func BenchmarkVectorSearcherRankResults(b *testing.B) {
	vs := NewVectorSearcher()
	
	// Create test results
	results := make([]VectorSearchResult, 100)
	for i := range results {
		results[i] = VectorSearchResult{
			ID:         string(rune('a' + i)),
			Similarity: float64(i) / 100.0,
			Score:      float64(i) / 100.0,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs.RankResults(results)
	}
}

func BenchmarkVectorSearcherNormalizeVector(b *testing.B) {
	vs := NewVectorSearcher()
	vector := make([]float32, 1536)
	
	// Initialize with random values
	for i := range vector {
		vector[i] = float32(i) / 1536.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs.normalizeVector(vector)
	}
}