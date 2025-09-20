package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
)

// KeywordSearcher handles text-based keyword matching and BM25 scoring
type KeywordSearcher struct {
	searchIndex SearchIndex
	config      *KeywordSearchConfig
}

// KeywordSearchConfig holds configuration for keyword search
type KeywordSearchConfig struct {
	DefaultK      int     `json:"default_k"`
	MinScore      float64 `json:"min_score"`
	MaxResults    int     `json:"max_results"`
	BM25K1        float64 `json:"bm25_k1"`        // Term frequency saturation parameter
	BM25B         float64 `json:"bm25_b"`         // Length normalization parameter
	CaseSensitive bool    `json:"case_sensitive"`
	StemWords     bool    `json:"stem_words"`
}

// KeywordSearchResult represents a single keyword search result
type KeywordSearchResult struct {
	ID           string                 `json:"id"`
	Content      string                 `json:"content"`
	Score        float64                `json:"score"`
	MatchedTerms []string               `json:"matched_terms"`
	Highlights   []string               `json:"highlights"`
	Metadata     map[string]interface{} `json:"metadata"`
	BM25Score    float64                `json:"bm25_score"`
}

// KeywordSearchResponse contains the complete search results
type KeywordSearchResponse struct {
	Results    []KeywordSearchResult  `json:"results"`
	QueryTime  float64                `json:"query_time_ms"`
	TotalFound int                    `json:"total_found"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// TermFrequency represents term frequency information
type TermFrequency struct {
	Term      string  `json:"term"`
	Frequency int     `json:"frequency"`
	TF        float64 `json:"tf"`
	IDF       float64 `json:"idf"`
	TFIDF     float64 `json:"tfidf"`
}

// NewKeywordSearcher creates a new KeywordSearcher instance
func NewKeywordSearcher() *KeywordSearcher {
	config := &KeywordSearchConfig{
		DefaultK:      10,
		MinScore:      0.1,
		MaxResults:    100,
		BM25K1:        1.2,
		BM25B:         0.75,
		CaseSensitive: false,
		StemWords:     false,
	}

	return &KeywordSearcher{
		config: config,
	}
}

// NewKeywordSearcherWithIndex creates a KeywordSearcher with a specific search index
func NewKeywordSearcherWithIndex(index SearchIndex, config *KeywordSearchConfig) *KeywordSearcher {
	if config == nil {
		config = &KeywordSearchConfig{
			DefaultK:      10,
			MinScore:      0.1,
			MaxResults:    100,
			BM25K1:        1.2,
			BM25B:         0.75,
			CaseSensitive: false,
			StemWords:     false,
		}
	}

	return &KeywordSearcher{
		searchIndex: index,
		config:      config,
	}
}

// Search performs keyword-based text search
func (ks *KeywordSearcher) Search(ctx context.Context, query string, k int, filters map[string]interface{}) (*KeywordSearchResponse, error) {
	if ks.searchIndex == nil {
		return nil, fmt.Errorf("search index not initialized")
	}

	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("empty query")
	}

	if k <= 0 {
		k = ks.config.DefaultK
	}

	if k > ks.config.MaxResults {
		k = ks.config.MaxResults
	}

	// Preprocess query
	processedQuery := ks.preprocessQuery(query)
	queryTerms := ks.tokenize(processedQuery)

	if len(queryTerms) == 0 {
		return &KeywordSearchResponse{
			Results:    []KeywordSearchResult{},
			TotalFound: 0,
			Metadata:   map[string]interface{}{"error": "no valid query terms"},
		}, nil
	}

	// Perform search using the search index
	searchResults, err := ks.searchIndex.Search(processedQuery, k*2) // Get more results for better ranking
	if err != nil {
		return nil, fmt.Errorf("search index query failed: %w", err)
	}

	// Convert to KeywordSearchResult format and calculate scores
	results := make([]KeywordSearchResult, 0, len(searchResults))
	for _, sr := range searchResults {
		// Match keywords and calculate score
		matchedTerms, highlights := ks.MatchKeywords(sr.Content, queryTerms)
		
		if len(matchedTerms) == 0 {
			continue // Skip if no terms matched
		}

		// Calculate BM25 score
		bm25Score := ks.calculateBM25Score(sr.Content, queryTerms, sr.Metadata)
		
		// Use BM25 score as primary score, fall back to search index score
		finalScore := bm25Score
		if finalScore == 0 {
			finalScore = sr.Score
		}

		// Skip results below minimum score threshold
		if finalScore < ks.config.MinScore {
			continue
		}

		result := KeywordSearchResult{
			ID:           sr.ID,
			Content:      sr.Content,
			Score:        finalScore,
			MatchedTerms: matchedTerms,
			Highlights:   highlights,
			Metadata:     sr.Metadata,
			BM25Score:    bm25Score,
		}

		results = append(results, result)
	}

	// Score and rank results
	scoredResults := ks.ScoreResults(results, queryTerms)

	// Limit to requested k
	if len(scoredResults) > k {
		scoredResults = scoredResults[:k]
	}

	response := &KeywordSearchResponse{
		Results:    scoredResults,
		TotalFound: len(scoredResults),
		Metadata:   make(map[string]interface{}),
	}

	// Add search metadata
	response.Metadata["query_terms"] = queryTerms
	response.Metadata["processed_query"] = processedQuery
	response.Metadata["bm25_k1"] = ks.config.BM25K1
	response.Metadata["bm25_b"] = ks.config.BM25B

	return response, nil
}

// MatchKeywords finds matching keywords in content and generates highlights
func (ks *KeywordSearcher) MatchKeywords(content string, queryTerms []string) ([]string, []string) {
	if content == "" || len(queryTerms) == 0 {
		return []string{}, []string{}
	}

	processedContent := ks.preprocessText(content)
	contentTerms := ks.tokenize(processedContent)
	
	// Create term frequency map for content
	termFreq := make(map[string]int)
	for _, term := range contentTerms {
		termFreq[term]++
	}

	var matchedTerms []string
	var highlights []string

	// Find matches
	for _, queryTerm := range queryTerms {
		if freq, exists := termFreq[queryTerm]; exists && freq > 0 {
			matchedTerms = append(matchedTerms, queryTerm)
			
			// Generate highlight snippet
			highlight := ks.generateHighlight(content, queryTerm)
			if highlight != "" {
				highlights = append(highlights, highlight)
			}
		}
	}

	return ks.deduplicateStrings(matchedTerms), ks.deduplicateStrings(highlights)
}

// ScoreResults calculates final scores and ranks results
func (ks *KeywordSearcher) ScoreResults(results []KeywordSearchResult, queryTerms []string) []KeywordSearchResult {
	// Calculate enhanced scores
	for i := range results {
		results[i].Score = ks.calculateEnhancedScore(results[i], queryTerms)
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		if math.Abs(results[i].Score-results[j].Score) < 1e-9 {
			// If scores are very close, use number of matched terms as tiebreaker
			return len(results[i].MatchedTerms) > len(results[j].MatchedTerms)
		}
		return results[i].Score > results[j].Score
	})

	return results
}

// preprocessQuery normalizes the query string
func (ks *KeywordSearcher) preprocessQuery(query string) string {
	return ks.preprocessText(query)
}

// preprocessText normalizes text for processing
func (ks *KeywordSearcher) preprocessText(text string) string {
	if !ks.config.CaseSensitive {
		text = strings.ToLower(text)
	}

	// Remove extra whitespace
	text = strings.TrimSpace(text)
	
	// Normalize whitespace
	words := strings.Fields(text)
	return strings.Join(words, " ")
}

// tokenize splits text into individual terms
func (ks *KeywordSearcher) tokenize(text string) []string {
	if text == "" {
		return []string{}
	}

	// Simple tokenization - split on whitespace and punctuation
	var tokens []string
	var currentToken strings.Builder

	for _, char := range text {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			currentToken.WriteRune(char)
		} else {
			if currentToken.Len() > 0 {
				token := currentToken.String()
				if len(token) >= 2 { // Filter out single characters
					tokens = append(tokens, token)
				}
				currentToken.Reset()
			}
		}
	}

	// Add final token if exists
	if currentToken.Len() > 0 {
		token := currentToken.String()
		if len(token) >= 2 {
			tokens = append(tokens, token)
		}
	}

	return ks.filterStopWords(tokens)
}

// filterStopWords removes common stop words
func (ks *KeywordSearcher) filterStopWords(tokens []string) []string {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "can": true, "this": true, "that": true,
		"these": true, "those": true, "i": true, "you": true, "he": true, "she": true,
		"it": true, "we": true, "they": true, "me": true, "him": true, "her": true,
		"us": true, "them": true, "my": true, "your": true, "his": true, "its": true,
		"our": true, "their": true,
	}

	var filtered []string
	for _, token := range tokens {
		if !stopWords[token] {
			filtered = append(filtered, token)
		}
	}

	return filtered
}

// generateHighlight creates a highlighted snippet around a matched term
func (ks *KeywordSearcher) generateHighlight(content, term string) string {
	if content == "" || term == "" {
		return ""
	}

	processedContent := ks.preprocessText(content)
	processedTerm := ks.preprocessText(term)

	// Find the term in the content
	index := strings.Index(processedContent, processedTerm)
	if index == -1 {
		return ""
	}

	// Create snippet around the match
	snippetLength := 100
	start := index - snippetLength/2
	if start < 0 {
		start = 0
	}

	end := index + len(processedTerm) + snippetLength/2
	if end > len(content) {
		end = len(content)
	}

	snippet := content[start:end]
	
	// Add ellipsis if truncated
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}

	return snippet
}

// calculateBM25Score computes BM25 score for a document
func (ks *KeywordSearcher) calculateBM25Score(content string, queryTerms []string, metadata map[string]interface{}) float64 {
	if content == "" || len(queryTerms) == 0 {
		return 0.0
	}

	contentTerms := ks.tokenize(ks.preprocessText(content))
	docLength := len(contentTerms)
	
	if docLength == 0 {
		return 0.0
	}

	// Get average document length from metadata or use default
	avgDocLength := 100.0 // Default average
	if avgLen, exists := metadata["avg_doc_length"]; exists {
		if avgLenFloat, ok := avgLen.(float64); ok {
			avgDocLength = avgLenFloat
		}
	}

	// Calculate term frequencies
	termFreq := make(map[string]int)
	for _, term := range contentTerms {
		termFreq[term]++
	}

	var bm25Score float64

	for _, queryTerm := range queryTerms {
		tf := float64(termFreq[queryTerm])
		if tf == 0 {
			continue
		}

		// Simple IDF calculation (would be better with corpus statistics)
		idf := math.Log(1000.0 / (1.0 + tf)) // Assume corpus of 1000 docs

		// BM25 formula
		numerator := tf * (ks.config.BM25K1 + 1)
		denominator := tf + ks.config.BM25K1*(1-ks.config.BM25B+ks.config.BM25B*(float64(docLength)/avgDocLength))
		
		bm25Score += idf * (numerator / denominator)
	}

	return bm25Score
}

// calculateEnhancedScore combines multiple scoring factors
func (ks *KeywordSearcher) calculateEnhancedScore(result KeywordSearchResult, queryTerms []string) float64 {
	baseScore := result.BM25Score
	if baseScore == 0 {
		baseScore = result.Score
	}

	// Boost score based on match ratio
	matchRatio := float64(len(result.MatchedTerms)) / float64(len(queryTerms))
	matchBoost := 1.0 + matchRatio*0.5

	// Boost score based on content length (prefer shorter, more focused content)
	contentLength := float64(len(result.Content))
	lengthPenalty := 1.0
	if contentLength > 1000 {
		lengthPenalty = 1000.0 / contentLength
	}

	finalScore := baseScore * matchBoost * lengthPenalty

	return finalScore
}

// deduplicateStrings removes duplicate strings while preserving order
func (ks *KeywordSearcher) deduplicateStrings(strings []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, str := range strings {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}

// SearchMultiple performs keyword search with multiple queries
func (ks *KeywordSearcher) SearchMultiple(ctx context.Context, queries []string, k int, filters map[string]interface{}) (*KeywordSearchResponse, error) {
	if len(queries) == 0 {
		return nil, fmt.Errorf("no queries provided")
	}

	allResults := make([]KeywordSearchResult, 0)
	seenIDs := make(map[string]bool)

	// Search with each query
	for _, query := range queries {
		response, err := ks.Search(ctx, query, k, filters)
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

	// Combine all query terms for scoring
	var allQueryTerms []string
	for _, query := range queries {
		terms := ks.tokenize(ks.preprocessQuery(query))
		allQueryTerms = append(allQueryTerms, terms...)
	}
	allQueryTerms = ks.deduplicateStrings(allQueryTerms)

	// Re-score and rank all results
	scoredResults := ks.ScoreResults(allResults, allQueryTerms)

	// Limit to requested k
	if len(scoredResults) > k {
		scoredResults = scoredResults[:k]
	}

	response := &KeywordSearchResponse{
		Results:    scoredResults,
		TotalFound: len(scoredResults),
		Metadata:   make(map[string]interface{}),
	}

	response.Metadata["query_count"] = len(queries)
	response.Metadata["combined_terms"] = allQueryTerms
	response.Metadata["deduplication"] = true

	return response, nil
}