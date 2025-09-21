package main

import (
	"context"
	"fmt"
	"math"
	"sort"
)

// VectorSearcher handles vector-based similarity search
type VectorSearcher struct {
	vectorStore VectorStore
	config      *VectorSearchConfig
}

// VectorSearchConfig holds configuration for vector search
type VectorSearchConfig struct {
	DefaultK        int     `json:"default_k"`
	MinSimilarity   float64 `json:"min_similarity"`
	MaxResults      int     `json:"max_results"`
	NormalizeScores bool    `json:"normalize_scores"`
}

// VectorSearchResult represents a single vector search result
type VectorSearchResult struct {
	ID         string                 `json:"id"`
	Content    string                 `json:"content"`
	Score      float64                `json:"score"`
	Similarity float64                `json:"similarity"`
	Metadata   map[string]interface{} `json:"metadata"`
	Embedding  []float32              `json:"embedding,omitempty"`
}

// VectorSearchResponse contains the complete search results
type VectorSearchResponse struct {
	Results    []VectorSearchResult `json:"results"`
	QueryTime  float64              `json:"query_time_ms"`
	TotalFound int                  `json:"total_found"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// NewVectorSearcher creates a new VectorSearcher instance
func NewVectorSearcher() *VectorSearcher {
	config := &VectorSearchConfig{
		DefaultK:        10,
		MinSimilarity:   0.1,
		MaxResults:      100,
		NormalizeScores: true,
	}

	return &VectorSearcher{
		config: config,
	}
}

// NewVectorSearcherWithStore creates a VectorSearcher with a specific vector store
func NewVectorSearcherWithStore(store VectorStore, config *VectorSearchConfig) *VectorSearcher {
	if config == nil {
		config = &VectorSearchConfig{
			DefaultK:        10,
			MinSimilarity:   0.1,
			MaxResults:      100,
			NormalizeScores: true,
		}
	}

	return &VectorSearcher{
		vectorStore: store,
		config:      config,
	}
}

// Search performs vector similarity search
func (vs *VectorSearcher) Search(ctx context.Context, queryEmbedding []float32, k int, filters map[string]interface{}) (*VectorSearchResponse, error) {
	if vs.vectorStore == nil {
		return nil, fmt.Errorf("vector store not initialized")
	}

	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("empty query embedding")
	}

	if k <= 0 {
		k = vs.config.DefaultK
	}

	if k > vs.config.MaxResults {
		k = vs.config.MaxResults
	}

	// Normalize query embedding
	normalizedQuery := vs.normalizeVector(queryEmbedding)

	// Perform the search
	vectorResults, err := vs.vectorStore.Search(ctx, normalizedQuery, k, filters)
	if err != nil {
		return nil, fmt.Errorf("vector store search failed: %w", err)
	}

	// Convert to VectorSearchResult format
	results := make([]VectorSearchResult, 0, len(vectorResults))
	for _, vr := range vectorResults {
		// Calculate cosine similarity
		similarity := vs.CosineSimilarity(normalizedQuery, vr.Embedding)
		
		// Skip results below minimum similarity threshold
		if similarity < vs.config.MinSimilarity {
			continue
		}

		result := VectorSearchResult{
			ID:         vr.ID,
			Content:    vr.Content,
			Score:      vr.Score,
			Similarity: similarity,
			Metadata:   vr.Metadata,
		}

		results = append(results, result)
	}

	// Rank results by similarity
	rankedResults := vs.RankResults(results)

	response := &VectorSearchResponse{
		Results:    rankedResults,
		TotalFound: len(rankedResults),
		Metadata:   make(map[string]interface{}),
	}

	// Add search metadata
	response.Metadata["query_embedding_dim"] = len(queryEmbedding)
	response.Metadata["min_similarity"] = vs.config.MinSimilarity
	response.Metadata["normalized_scores"] = vs.config.NormalizeScores

	return response, nil
}

// CosineSimilarity calculates cosine similarity between two vectors
func (vs *VectorSearcher) CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	if len(a) == 0 {
		return 0.0
	}

	var dotProduct, normA, normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}

	similarity := dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
	
	// Clamp to [-1, 1] range to handle floating point precision issues
	if similarity > 1.0 {
		similarity = 1.0
	} else if similarity < -1.0 {
		similarity = -1.0
	}

	return similarity
}

// RankResults sorts results by similarity score in descending order
func (vs *VectorSearcher) RankResults(results []VectorSearchResult) []VectorSearchResult {
	// Create a copy to avoid modifying the original slice
	ranked := make([]VectorSearchResult, len(results))
	copy(ranked, results)

	// Sort by similarity (descending) and then by score (descending) as tiebreaker
	sort.Slice(ranked, func(i, j int) bool {
		if math.Abs(ranked[i].Similarity-ranked[j].Similarity) < 1e-9 {
			// If similarities are very close, use score as tiebreaker
			return ranked[i].Score > ranked[j].Score
		}
		return ranked[i].Similarity > ranked[j].Similarity
	})

	// Normalize scores if enabled
	if vs.config.NormalizeScores && len(ranked) > 0 {
		vs.normalizeScores(ranked)
	}

	return ranked
}

// normalizeVector normalizes a vector to unit length
func (vs *VectorSearcher) normalizeVector(vector []float32) []float32 {
	var norm float64
	for _, val := range vector {
		norm += float64(val) * float64(val)
	}

	if norm == 0.0 {
		return vector
	}

	norm = math.Sqrt(norm)
	normalized := make([]float32, len(vector))
	for i, val := range vector {
		normalized[i] = float32(float64(val) / norm)
	}

	return normalized
}

// normalizeScores normalizes similarity scores to [0, 1] range
func (vs *VectorSearcher) normalizeScores(results []VectorSearchResult) {
	if len(results) == 0 {
		return
	}

	// Find min and max similarities
	minSim := results[0].Similarity
	maxSim := results[0].Similarity

	for _, result := range results {
		if result.Similarity < minSim {
			minSim = result.Similarity
		}
		if result.Similarity > maxSim {
			maxSim = result.Similarity
		}
	}

	// Avoid division by zero
	if maxSim == minSim {
		for i := range results {
			results[i].Score = 1.0
		}
		return
	}

	// Normalize to [0, 1] range
	for i := range results {
		results[i].Score = (results[i].Similarity - minSim) / (maxSim - minSim)
	}
}

// SearchMultiple performs vector search with multiple query embeddings
func (vs *VectorSearcher) SearchMultiple(ctx context.Context, queryEmbeddings [][]float32, k int, filters map[string]interface{}) (*VectorSearchResponse, error) {
	if len(queryEmbeddings) == 0 {
		return nil, fmt.Errorf("no query embeddings provided")
	}

	allResults := make([]VectorSearchResult, 0)
	seenIDs := make(map[string]bool)

	// Search with each embedding
	for _, embedding := range queryEmbeddings {
		response, err := vs.Search(ctx, embedding, k, filters)
		if err != nil {
			continue // Skip failed searches
		}

		// Merge results, avoiding duplicates
		for _, result := range response.Results {
			if !seenIDs[result.ID] {
				seenIDs[result.ID] = true
				allResults = append(allResults, result)
			}
		}
	}

	// Rank all results
	rankedResults := vs.RankResults(allResults)

	// Limit to requested k
	if len(rankedResults) > k {
		rankedResults = rankedResults[:k]
	}

	response := &VectorSearchResponse{
		Results:    rankedResults,
		TotalFound: len(rankedResults),
		Metadata:   make(map[string]interface{}),
	}

	response.Metadata["query_count"] = len(queryEmbeddings)
	response.Metadata["deduplication"] = true

	return response, nil
}

// GetSimilarityMatrix calculates similarity matrix between multiple vectors
func (vs *VectorSearcher) GetSimilarityMatrix(vectors [][]float32) [][]float64 {
	n := len(vectors)
	matrix := make([][]float64, n)
	
	for i := 0; i < n; i++ {
		matrix[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			if i == j {
				matrix[i][j] = 1.0
			} else {
				matrix[i][j] = vs.CosineSimilarity(vectors[i], vectors[j])
			}
		}
	}

	return matrix
}

// FindNearestNeighbors finds k nearest neighbors for a given vector
func (vs *VectorSearcher) FindNearestNeighbors(ctx context.Context, targetEmbedding []float32, candidates []VectorSearchResult, k int) []VectorSearchResult {
	if len(candidates) == 0 {
		return []VectorSearchResult{}
	}

	// Calculate similarities
	results := make([]VectorSearchResult, len(candidates))
	for i, candidate := range candidates {
		similarity := vs.CosineSimilarity(targetEmbedding, candidate.Embedding)
		results[i] = VectorSearchResult{
			ID:         candidate.ID,
			Content:    candidate.Content,
			Score:      candidate.Score,
			Similarity: similarity,
			Metadata:   candidate.Metadata,
			Embedding:  candidate.Embedding,
		}
	}

	// Rank and limit results
	ranked := vs.RankResults(results)
	if len(ranked) > k {
		ranked = ranked[:k]
	}

	return ranked
}