package main

import (
	"context"
	"fmt"
	"math"
	"sort"
)

// ResultFuser combines results from multiple retrieval methods using Reciprocal Rank Fusion (RRF)
type ResultFuser struct {
	config *ResultFuserConfig
}

// ResultFuserConfig holds configuration for result fusion
type ResultFuserConfig struct {
	RRFConstant     float64 `json:"rrf_constant"`      // RRF k parameter (typically 60)
	VectorWeight    float64 `json:"vector_weight"`     // Weight for vector search results
	KeywordWeight   float64 `json:"keyword_weight"`    // Weight for keyword search results
	GraphWeight     float64 `json:"graph_weight"`      // Weight for graph search results
	MinScore        float64 `json:"min_score"`         // Minimum score threshold
	MaxResults      int     `json:"max_results"`       // Maximum number of results to return
	NormalizeScores bool    `json:"normalize_scores"`  // Whether to normalize scores before fusion
}

// FusionInput represents input from a single retrieval method
type FusionInput struct {
	Method  string        `json:"method"`   // "vector", "keyword", "graph"
	Results []FusionItem  `json:"results"`
	Weight  float64       `json:"weight"`
	Metadata map[string]interface{} `json:"metadata"`
}

// FusionItem represents a single item to be fused
type FusionItem struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Score    float64                `json:"score"`
	Rank     int                    `json:"rank"`
	Metadata map[string]interface{} `json:"metadata"`
}

// FusedResult represents the final fused result
type FusedResult struct {
	ID           string                 `json:"id"`
	Content      string                 `json:"content"`
	FinalScore   float64                `json:"final_score"`
	RRFScore     float64                `json:"rrf_score"`
	CombinedScore float64               `json:"combined_score"`
	SourceMethods []string              `json:"source_methods"`
	MethodScores  map[string]float64    `json:"method_scores"`
	Rank         int                    `json:"rank"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// FusionResponse contains the complete fusion results
type FusionResponse struct {
	Results      []FusedResult          `json:"results"`
	TotalResults int                    `json:"total_results"`
	FusionStats  FusionStats            `json:"fusion_stats"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// FusionStats contains statistics about the fusion process
type FusionStats struct {
	InputMethods    []string           `json:"input_methods"`
	MethodCounts    map[string]int     `json:"method_counts"`
	OverlapMatrix   map[string]map[string]int `json:"overlap_matrix"`
	FusionTime      float64            `json:"fusion_time_ms"`
	RRFConstant     float64            `json:"rrf_constant"`
	WeightedFusion  bool               `json:"weighted_fusion"`
}

// NewResultFuser creates a new ResultFuser with default configuration
func NewResultFuser() *ResultFuser {
	config := &ResultFuserConfig{
		RRFConstant:     60.0,
		VectorWeight:    1.0,
		KeywordWeight:   1.0,
		GraphWeight:     1.0,
		MinScore:        0.0,
		MaxResults:      100,
		NormalizeScores: true,
	}

	return &ResultFuser{
		config: config,
	}
}

// NewResultFuserWithConfig creates a ResultFuser with custom configuration
func NewResultFuserWithConfig(config *ResultFuserConfig) *ResultFuser {
	if config == nil {
		return NewResultFuser()
	}

	// Create a new config to avoid modifying the original
	newConfig := *config

	// Set defaults for missing values
	if newConfig.RRFConstant <= 0 {
		newConfig.RRFConstant = 60.0
	}
	if newConfig.MaxResults <= 0 {
		newConfig.MaxResults = 100
	}

	return &ResultFuser{
		config: &newConfig,
	}
}

// Fuse combines results from multiple retrieval methods
func (rf *ResultFuser) Fuse(ctx context.Context, inputs []FusionInput) (*FusionResponse, error) {
	if len(inputs) == 0 {
		return &FusionResponse{
			Results:      []FusedResult{},
			TotalResults: 0,
			FusionStats:  FusionStats{},
			Metadata:     make(map[string]interface{}),
		}, nil
	}

	// Validate and prepare inputs
	validInputs, err := rf.validateInputs(inputs)
	if err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Normalize scores if enabled
	if rf.config.NormalizeScores {
		rf.normalizeInputScores(validInputs)
	}

	// Calculate RRF scores
	rrfResults := rf.RRF(validInputs)

	// Calculate combined scores with weights
	combinedResults := rf.CombineScores(rrfResults, validInputs)

	// Filter by minimum score
	filteredResults := rf.filterByMinScore(combinedResults)

	// Sort by final score and limit results
	finalResults := rf.rankAndLimit(filteredResults)

	// Calculate fusion statistics
	stats := rf.calculateFusionStats(validInputs, finalResults)

	response := &FusionResponse{
		Results:      finalResults,
		TotalResults: len(finalResults),
		FusionStats:  stats,
		Metadata:     make(map[string]interface{}),
	}

	// Add metadata
	response.Metadata["fusion_method"] = "RRF"
	response.Metadata["rrf_constant"] = rf.config.RRFConstant
	response.Metadata["normalized_scores"] = rf.config.NormalizeScores
	response.Metadata["min_score_threshold"] = rf.config.MinScore

	return response, nil
}

// RRF implements Reciprocal Rank Fusion algorithm
func (rf *ResultFuser) RRF(inputs []FusionInput) map[string]*FusedResult {
	results := make(map[string]*FusedResult)

	// Process each input method
	for _, input := range inputs {
		for rank, item := range input.Results {
			// RRF score = 1 / (k + rank), where rank is 1-based
			rrfScore := 1.0 / (rf.config.RRFConstant + float64(rank+1))

			if existing, exists := results[item.ID]; exists {
				// Accumulate RRF scores
				existing.RRFScore += rrfScore
				existing.SourceMethods = append(existing.SourceMethods, input.Method)
				existing.MethodScores[input.Method] = item.Score
				
				// Merge metadata
				rf.mergeMetadata(existing.Metadata, item.Metadata)
			} else {
				// Create new result
				results[item.ID] = &FusedResult{
					ID:            item.ID,
					Content:       item.Content,
					RRFScore:      rrfScore,
					SourceMethods: []string{input.Method},
					MethodScores:  map[string]float64{input.Method: item.Score},
					Metadata:      rf.copyMetadata(item.Metadata),
				}
			}
		}
	}

	return results
}

// CombineScores combines RRF scores with weighted method scores
func (rf *ResultFuser) CombineScores(rrfResults map[string]*FusedResult, inputs []FusionInput) []*FusedResult {
	// Create weight map for methods
	methodWeights := rf.getMethodWeights(inputs)

	var results []*FusedResult
	for _, result := range rrfResults {
		// Calculate weighted score
		var weightedScore float64
		var totalWeight float64

		for method, score := range result.MethodScores {
			if weight, exists := methodWeights[method]; exists {
				weightedScore += score * weight
				totalWeight += weight
			}
		}

		// Normalize weighted score
		if totalWeight > 0 {
			weightedScore /= totalWeight
		}

		// Combine RRF and weighted scores
		// RRF provides ranking information, weighted score provides magnitude
		result.CombinedScore = result.RRFScore * (1.0 + weightedScore)
		result.FinalScore = result.CombinedScore

		results = append(results, result)
	}

	return results
}

// validateInputs validates and prepares fusion inputs
func (rf *ResultFuser) validateInputs(inputs []FusionInput) ([]FusionInput, error) {
	var validInputs []FusionInput

	for i, input := range inputs {
		if input.Method == "" {
			return nil, fmt.Errorf("input %d: method cannot be empty", i)
		}

		if len(input.Results) == 0 {
			continue // Skip empty inputs
		}

		// Assign ranks if not set
		for j := range input.Results {
			if input.Results[j].Rank == 0 {
				input.Results[j].Rank = j + 1
			}
		}

		// Set weight if not specified
		if input.Weight == 0 {
			input.Weight = rf.getDefaultWeight(input.Method)
		}

		validInputs = append(validInputs, input)
	}

	if len(validInputs) == 0 {
		return nil, fmt.Errorf("no valid inputs provided")
	}

	return validInputs, nil
}

// normalizeInputScores normalizes scores within each input method
func (rf *ResultFuser) normalizeInputScores(inputs []FusionInput) {
	for i := range inputs {
		if len(inputs[i].Results) == 0 {
			continue
		}

		// Find min and max scores
		minScore := inputs[i].Results[0].Score
		maxScore := inputs[i].Results[0].Score

		for _, result := range inputs[i].Results {
			if result.Score < minScore {
				minScore = result.Score
			}
			if result.Score > maxScore {
				maxScore = result.Score
			}
		}

		// Normalize to [0, 1] range
		scoreRange := maxScore - minScore
		if scoreRange > 0 {
			for j := range inputs[i].Results {
				inputs[i].Results[j].Score = (inputs[i].Results[j].Score - minScore) / scoreRange
			}
		} else {
			// All scores are the same, set to 1.0
			for j := range inputs[i].Results {
				inputs[i].Results[j].Score = 1.0
			}
		}
	}
}

// getMethodWeights returns weights for different methods
func (rf *ResultFuser) getMethodWeights(inputs []FusionInput) map[string]float64 {
	weights := make(map[string]float64)

	for _, input := range inputs {
		if input.Weight > 0 {
			weights[input.Method] = input.Weight
		} else {
			weights[input.Method] = rf.getDefaultWeight(input.Method)
		}
	}

	return weights
}

// getDefaultWeight returns default weight for a method
func (rf *ResultFuser) getDefaultWeight(method string) float64 {
	switch method {
	case "vector":
		return rf.config.VectorWeight
	case "keyword":
		return rf.config.KeywordWeight
	case "graph":
		return rf.config.GraphWeight
	default:
		return 1.0
	}
}

// filterByMinScore filters results by minimum score threshold
func (rf *ResultFuser) filterByMinScore(results []*FusedResult) []*FusedResult {
	if rf.config.MinScore <= 0 {
		return results
	}

	var filtered []*FusedResult
	for _, result := range results {
		if result.FinalScore >= rf.config.MinScore {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// rankAndLimit sorts results by final score and limits to max results
func (rf *ResultFuser) rankAndLimit(results []*FusedResult) []FusedResult {
	// Sort by final score (descending)
	sort.Slice(results, func(i, j int) bool {
		if math.Abs(results[i].FinalScore-results[j].FinalScore) < 1e-9 {
			// If scores are very close, use RRF score as tiebreaker
			return results[i].RRFScore > results[j].RRFScore
		}
		return results[i].FinalScore > results[j].FinalScore
	})

	// Assign final ranks
	for i := range results {
		results[i].Rank = i + 1
	}

	// Limit to max results
	if len(results) > rf.config.MaxResults {
		results = results[:rf.config.MaxResults]
	}

	// Convert to value slice
	finalResults := make([]FusedResult, len(results))
	for i, result := range results {
		finalResults[i] = *result
	}

	return finalResults
}

// calculateFusionStats calculates statistics about the fusion process
func (rf *ResultFuser) calculateFusionStats(inputs []FusionInput, results []FusedResult) FusionStats {
	stats := FusionStats{
		InputMethods:   make([]string, 0, len(inputs)),
		MethodCounts:   make(map[string]int),
		OverlapMatrix:  make(map[string]map[string]int),
		RRFConstant:    rf.config.RRFConstant,
		WeightedFusion: rf.hasWeightedFusion(inputs),
	}

	// Collect method information
	for _, input := range inputs {
		stats.InputMethods = append(stats.InputMethods, input.Method)
		stats.MethodCounts[input.Method] = len(input.Results)
		stats.OverlapMatrix[input.Method] = make(map[string]int)
	}

	// Calculate overlap matrix
	for _, result := range results {
		methods := result.SourceMethods
		for i, method1 := range methods {
			for j, method2 := range methods {
				if i != j {
					stats.OverlapMatrix[method1][method2]++
				}
			}
		}
	}

	return stats
}

// hasWeightedFusion checks if any method has non-default weights
func (rf *ResultFuser) hasWeightedFusion(inputs []FusionInput) bool {
	for _, input := range inputs {
		defaultWeight := rf.getDefaultWeight(input.Method)
		if math.Abs(input.Weight-defaultWeight) > 1e-9 {
			return true
		}
	}
	return false
}

// mergeMetadata merges metadata from two sources
func (rf *ResultFuser) mergeMetadata(target, source map[string]interface{}) {
	if target == nil || source == nil {
		return
	}

	for key, value := range source {
		if _, exists := target[key]; !exists {
			target[key] = value
		}
	}
}

// copyMetadata creates a copy of metadata map
func (rf *ResultFuser) copyMetadata(source map[string]interface{}) map[string]interface{} {
	if source == nil {
		return make(map[string]interface{})
	}

	target := make(map[string]interface{})
	for key, value := range source {
		target[key] = value
	}

	return target
}

// FuseVectorAndKeyword is a convenience method for fusing vector and keyword results
func (rf *ResultFuser) FuseVectorAndKeyword(ctx context.Context, vectorResults []VectorSearchResult, keywordResults []KeywordSearchResult) (*FusionResponse, error) {
	inputs := make([]FusionInput, 0, 2)

	// Convert vector results
	if len(vectorResults) > 0 {
		vectorItems := make([]FusionItem, len(vectorResults))
		for i, vr := range vectorResults {
			vectorItems[i] = FusionItem{
				ID:       vr.ID,
				Content:  vr.Content,
				Score:    vr.Score,
				Rank:     i + 1,
				Metadata: vr.Metadata,
			}
		}

		inputs = append(inputs, FusionInput{
			Method:  "vector",
			Results: vectorItems,
			Weight:  rf.config.VectorWeight,
		})
	}

	// Convert keyword results
	if len(keywordResults) > 0 {
		keywordItems := make([]FusionItem, len(keywordResults))
		for i, kr := range keywordResults {
			keywordItems[i] = FusionItem{
				ID:       kr.ID,
				Content:  kr.Content,
				Score:    kr.Score,
				Rank:     i + 1,
				Metadata: kr.Metadata,
			}
		}

		inputs = append(inputs, FusionInput{
			Method:  "keyword",
			Results: keywordItems,
			Weight:  rf.config.KeywordWeight,
		})
	}

	return rf.Fuse(ctx, inputs)
}

// GetConfig returns the current configuration
func (rf *ResultFuser) GetConfig() *ResultFuserConfig {
	return rf.config
}

// UpdateConfig updates the configuration
func (rf *ResultFuser) UpdateConfig(config *ResultFuserConfig) {
	if config != nil {
		rf.config = config
	}
}