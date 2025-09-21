package main

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"
)

// ResultRanker handles advanced ranking and scoring of search results
type ResultRanker struct {
	config *ResultRankerConfig
}

// ResultRankerConfig holds configuration for result ranking
type ResultRankerConfig struct {
	RelevanceWeight    float64 `json:"relevance_weight"`     // Weight for relevance score
	FreshnessWeight    float64 `json:"freshness_weight"`     // Weight for recency
	AuthorityWeight    float64 `json:"authority_weight"`     // Weight for source authority
	DiversityWeight    float64 `json:"diversity_weight"`     // Weight for result diversity
	QualityWeight      float64 `json:"quality_weight"`       // Weight for content quality
	PersonalizationWeight float64 `json:"personalization_weight"` // Weight for personalization
	BoostThreshold     float64 `json:"boost_threshold"`      // Threshold for score boosting
	PenaltyThreshold   float64 `json:"penalty_threshold"`    // Threshold for score penalty
	MaxResults         int     `json:"max_results"`          // Maximum results to rank
	DiversityRadius    float64 `json:"diversity_radius"`     // Radius for diversity calculation
}

// RankableResult represents a result that can be ranked
type RankableResult struct {
	ID              string                 `json:"id"`
	Content         string                 `json:"content"`
	BaseScore       float64                `json:"base_score"`
	RelevanceScore  float64                `json:"relevance_score"`
	FreshnessScore  float64                `json:"freshness_score"`
	AuthorityScore  float64                `json:"authority_score"`
	QualityScore    float64                `json:"quality_score"`
	DiversityScore  float64                `json:"diversity_score"`
	PersonalizationScore float64           `json:"personalization_score"`
	FinalScore      float64                `json:"final_score"`
	Rank            int                    `json:"rank"`
	Boosts          []string               `json:"boosts,omitempty"`
	Penalties       []string               `json:"penalties,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
	Timestamp       time.Time              `json:"timestamp,omitempty"`
	Source          string                 `json:"source,omitempty"`
}

// RankingContext provides context for ranking decisions
type RankingContext struct {
	Query           string                 `json:"query"`
	UserID          string                 `json:"user_id,omitempty"`
	UserPreferences map[string]interface{} `json:"user_preferences,omitempty"`
	SessionContext  map[string]interface{} `json:"session_context,omitempty"`
	TimeContext     time.Time              `json:"time_context"`
	DomainContext   string                 `json:"domain_context,omitempty"`
}

// RankingResponse contains the ranked results and statistics
type RankingResponse struct {
	Results      []RankableResult       `json:"results"`
	TotalResults int                    `json:"total_results"`
	RankingStats RankingStats           `json:"ranking_stats"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// RankingStats contains statistics about the ranking process
type RankingStats struct {
	RankingTime     float64            `json:"ranking_time_ms"`
	ScoreDistribution map[string]float64 `json:"score_distribution"`
	BoostCount      int                `json:"boost_count"`
	PenaltyCount    int                `json:"penalty_count"`
	DiversityMetric float64            `json:"diversity_metric"`
	QualityMetric   float64            `json:"quality_metric"`
}

// NewResultRanker creates a new ResultRanker with default configuration
func NewResultRanker() *ResultRanker {
	config := &ResultRankerConfig{
		RelevanceWeight:       1.0,
		FreshnessWeight:       0.2,
		AuthorityWeight:       0.3,
		DiversityWeight:       0.1,
		QualityWeight:         0.4,
		PersonalizationWeight: 0.2,
		BoostThreshold:        0.8,
		PenaltyThreshold:      0.3,
		MaxResults:            100,
		DiversityRadius:       0.5,
	}

	return &ResultRanker{
		config: config,
	}
}

// NewResultRankerWithConfig creates a ResultRanker with custom configuration
func NewResultRankerWithConfig(config *ResultRankerConfig) *ResultRanker {
	if config == nil {
		return NewResultRanker()
	}

	// Create a new config to avoid modifying the original
	newConfig := *config

	// Set defaults for missing values
	if newConfig.MaxResults <= 0 {
		newConfig.MaxResults = 100
	}
	if newConfig.DiversityRadius <= 0 {
		newConfig.DiversityRadius = 0.5
	}

	return &ResultRanker{
		config: &newConfig,
	}
}

// Rank ranks a list of results using multiple scoring factors
func (rr *ResultRanker) Rank(ctx context.Context, results []RankableResult, context *RankingContext) (*RankingResponse, error) {
	startTime := time.Now()

	if len(results) == 0 {
		return &RankingResponse{
			Results:      []RankableResult{},
			TotalResults: 0,
			RankingStats: RankingStats{},
			Metadata:     make(map[string]interface{}),
		}, nil
	}

	if context == nil {
		context = &RankingContext{
			TimeContext: time.Now(),
		}
	}

	// Limit input size
	if len(results) > rr.config.MaxResults*2 {
		results = results[:rr.config.MaxResults*2]
	}

	// Calculate individual scores
	scoredResults := make([]RankableResult, len(results))
	copy(scoredResults, results)

	for i := range scoredResults {
		rr.calculateRelevanceScore(&scoredResults[i], context)
		rr.calculateFreshnessScore(&scoredResults[i], context)
		rr.calculateAuthorityScore(&scoredResults[i], context)
		rr.calculateQualityScore(&scoredResults[i], context)
		rr.calculatePersonalizationScore(&scoredResults[i], context)
	}

	// Calculate diversity scores (requires all results)
	rr.calculateDiversityScores(scoredResults, context)

	// Calculate final scores and apply boosts/penalties
	for i := range scoredResults {
		rr.calculateFinalScore(&scoredResults[i])
		rr.applyBoostsAndPenalties(&scoredResults[i], context)
	}

	// Sort results by final score
	sortedResults := rr.SortResults(scoredResults)

	// Limit to max results
	if len(sortedResults) > rr.config.MaxResults {
		sortedResults = sortedResults[:rr.config.MaxResults]
	}

	// Assign final ranks
	for i := range sortedResults {
		sortedResults[i].Rank = i + 1
	}

	// Calculate ranking statistics
	stats := rr.calculateRankingStats(sortedResults, time.Since(startTime))

	response := &RankingResponse{
		Results:      sortedResults,
		TotalResults: len(sortedResults),
		RankingStats: stats,
		Metadata:     make(map[string]interface{}),
	}

	// Add metadata
	response.Metadata["ranking_algorithm"] = "multi_factor"
	response.Metadata["weights"] = map[string]float64{
		"relevance":       rr.config.RelevanceWeight,
		"freshness":       rr.config.FreshnessWeight,
		"authority":       rr.config.AuthorityWeight,
		"diversity":       rr.config.DiversityWeight,
		"quality":         rr.config.QualityWeight,
		"personalization": rr.config.PersonalizationWeight,
	}

	return response, nil
}

// Score calculates a comprehensive score for a single result
func (rr *ResultRanker) Score(result *RankableResult, context *RankingContext) float64 {
	if context == nil {
		context = &RankingContext{
			TimeContext: time.Now(),
		}
	}

	// Calculate individual scores
	rr.calculateRelevanceScore(result, context)
	rr.calculateFreshnessScore(result, context)
	rr.calculateAuthorityScore(result, context)
	rr.calculateQualityScore(result, context)
	rr.calculatePersonalizationScore(result, context)

	// Calculate final score
	rr.calculateFinalScore(result)
	rr.applyBoostsAndPenalties(result, context)

	return result.FinalScore
}

// SortResults sorts results by final score in descending order
func (rr *ResultRanker) SortResults(results []RankableResult) []RankableResult {
	// Create a copy to avoid modifying the original slice
	sorted := make([]RankableResult, len(results))
	copy(sorted, results)

	// Sort by final score (descending), then by base score as tiebreaker
	sort.Slice(sorted, func(i, j int) bool {
		if math.Abs(sorted[i].FinalScore-sorted[j].FinalScore) < 1e-9 {
			// If final scores are very close, use base score as tiebreaker
			if math.Abs(sorted[i].BaseScore-sorted[j].BaseScore) < 1e-9 {
				// If base scores are also close, use relevance score
				return sorted[i].RelevanceScore > sorted[j].RelevanceScore
			}
			return sorted[i].BaseScore > sorted[j].BaseScore
		}
		return sorted[i].FinalScore > sorted[j].FinalScore
	})

	return sorted
}

// calculateRelevanceScore calculates relevance score based on query matching
func (rr *ResultRanker) calculateRelevanceScore(result *RankableResult, context *RankingContext) {
	if context.Query == "" {
		result.RelevanceScore = result.BaseScore
		return
	}

	// Use base score as starting point
	relevanceScore := result.BaseScore

	// Boost for exact matches in content
	queryLower := strings.ToLower(context.Query)
	contentLower := strings.ToLower(result.Content)

	if strings.Contains(contentLower, queryLower) {
		relevanceScore *= 1.2
	}

	// Boost for matches in title/metadata
	if title, exists := result.Metadata["title"]; exists {
		if titleStr, ok := title.(string); ok {
			if strings.Contains(strings.ToLower(titleStr), queryLower) {
				relevanceScore *= 1.3
			}
		}
	}

	// Normalize to [0, 1] range
	result.RelevanceScore = math.Min(relevanceScore, 1.0)
}

// calculateFreshnessScore calculates freshness score based on recency
func (rr *ResultRanker) calculateFreshnessScore(result *RankableResult, context *RankingContext) {
	if result.Timestamp.IsZero() {
		result.FreshnessScore = 0.5 // Default neutral score
		return
	}

	// Calculate age in days
	age := context.TimeContext.Sub(result.Timestamp).Hours() / 24.0

	// Exponential decay with 30-day half-life: score = e^(-ln(2) * age / 30)
	result.FreshnessScore = math.Exp(-math.Ln2 * age / 30.0)
}

// calculateAuthorityScore calculates authority score based on source credibility
func (rr *ResultRanker) calculateAuthorityScore(result *RankableResult, context *RankingContext) {
	// Default authority score
	result.AuthorityScore = 0.5

	// Check for authority indicators in metadata
	if authority, exists := result.Metadata["authority_score"]; exists {
		if authScore, ok := authority.(float64); ok {
			result.AuthorityScore = authScore
		}
	}

	// Boost for trusted sources
	trustedSources := []string{"official", "verified", "academic", "government"}
	sourceLower := strings.ToLower(result.Source)

	for _, trusted := range trustedSources {
		if strings.Contains(sourceLower, trusted) {
			result.AuthorityScore = math.Min(result.AuthorityScore*1.2, 1.0)
			break
		}
	}
}

// calculateQualityScore calculates content quality score
func (rr *ResultRanker) calculateQualityScore(result *RankableResult, context *RankingContext) {
	// Start with base quality
	qualityScore := 0.5

	// Content length factor (prefer moderate length)
	contentLength := float64(len(result.Content))
	if contentLength > 100 && contentLength < 2000 {
		qualityScore += 0.2
	} else if contentLength < 50 {
		qualityScore -= 0.2
	}

	// Check for quality indicators in metadata
	if quality, exists := result.Metadata["quality_score"]; exists {
		if qualScore, ok := quality.(float64); ok {
			qualityScore = (qualityScore + qualScore) / 2.0
		}
	}

	// Penalize very short or very long content
	if contentLength < 20 {
		qualityScore *= 0.5
	} else if contentLength > 5000 {
		qualityScore *= 0.8
	}

	result.QualityScore = math.Max(0.0, math.Min(qualityScore, 1.0))
}

// calculatePersonalizationScore calculates personalization score based on user preferences
func (rr *ResultRanker) calculatePersonalizationScore(result *RankableResult, context *RankingContext) {
	// Default neutral score
	result.PersonalizationScore = 0.5

	if context.UserPreferences == nil || len(context.UserPreferences) == 0 {
		return
	}

	// Check for preferred topics
	if preferredTopics, exists := context.UserPreferences["topics"]; exists {
		if topics, ok := preferredTopics.([]string); ok {
			for _, topic := range topics {
				if strings.Contains(strings.ToLower(result.Content), strings.ToLower(topic)) {
					result.PersonalizationScore = math.Min(result.PersonalizationScore+0.2, 1.0)
				}
			}
		}
	}

	// Check for preferred sources
	if preferredSources, exists := context.UserPreferences["sources"]; exists {
		if sources, ok := preferredSources.([]string); ok {
			for _, source := range sources {
				if strings.Contains(strings.ToLower(result.Source), strings.ToLower(source)) {
					result.PersonalizationScore = math.Min(result.PersonalizationScore+0.3, 1.0)
				}
			}
		}
	}
}

// calculateDiversityScores calculates diversity scores for all results
func (rr *ResultRanker) calculateDiversityScores(results []RankableResult, context *RankingContext) {
	if len(results) <= 1 {
		for i := range results {
			results[i].DiversityScore = 1.0
		}
		return
	}

	// Calculate pairwise similarities and diversity scores
	for i := range results {
		diversityScore := 1.0
		similaritySum := 0.0
		comparisons := 0

		for j := range results {
			if i != j {
				similarity := rr.calculateContentSimilarity(results[i].Content, results[j].Content)
				similaritySum += similarity
				comparisons++

				// Penalize high similarity
				if similarity > rr.config.DiversityRadius {
					diversityScore *= (1.0 - similarity*0.5)
				}
			}
		}

		// Average similarity penalty
		if comparisons > 0 {
			avgSimilarity := similaritySum / float64(comparisons)
			diversityScore *= (1.0 - avgSimilarity*0.3)
		}

		results[i].DiversityScore = math.Max(0.1, math.Min(diversityScore, 1.0))
	}
}

// calculateContentSimilarity calculates similarity between two content strings
func (rr *ResultRanker) calculateContentSimilarity(content1, content2 string) float64 {
	if content1 == "" || content2 == "" {
		return 0.0
	}

	// Simple Jaccard similarity based on words
	words1 := rr.tokenizeContent(content1)
	words2 := rr.tokenizeContent(content2)

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Create sets
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, word := range words1 {
		set1[word] = true
	}
	for _, word := range words2 {
		set2[word] = true
	}

	// Calculate intersection and union
	intersection := 0
	union := len(set1)

	for word := range set2 {
		if set1[word] {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// tokenizeContent tokenizes content into words
func (rr *ResultRanker) tokenizeContent(content string) []string {
	// Simple tokenization - split on whitespace and convert to lowercase
	words := strings.Fields(strings.ToLower(content))
	
	// Filter out very short words
	var filtered []string
	for _, word := range words {
		if len(word) >= 3 {
			filtered = append(filtered, word)
		}
	}

	return filtered
}

// calculateFinalScore combines all individual scores into a final score
func (rr *ResultRanker) calculateFinalScore(result *RankableResult) {
	finalScore := 0.0

	finalScore += result.RelevanceScore * rr.config.RelevanceWeight
	finalScore += result.FreshnessScore * rr.config.FreshnessWeight
	finalScore += result.AuthorityScore * rr.config.AuthorityWeight
	finalScore += result.QualityScore * rr.config.QualityWeight
	finalScore += result.DiversityScore * rr.config.DiversityWeight
	finalScore += result.PersonalizationScore * rr.config.PersonalizationWeight

	// Normalize by total weight
	totalWeight := rr.config.RelevanceWeight + rr.config.FreshnessWeight + 
		rr.config.AuthorityWeight + rr.config.QualityWeight + 
		rr.config.DiversityWeight + rr.config.PersonalizationWeight

	if totalWeight > 0 {
		finalScore /= totalWeight
	}

	result.FinalScore = finalScore
}

// applyBoostsAndPenalties applies score boosts and penalties based on various factors
func (rr *ResultRanker) applyBoostsAndPenalties(result *RankableResult, context *RankingContext) {
	originalScore := result.FinalScore

	// Apply boosts
	if result.FinalScore >= rr.config.BoostThreshold {
		result.FinalScore *= 1.1
		result.Boosts = append(result.Boosts, "high_score_boost")
	}

	if result.AuthorityScore >= 0.8 {
		result.FinalScore *= 1.05
		result.Boosts = append(result.Boosts, "authority_boost")
	}

	if result.FreshnessScore >= 0.9 {
		result.FinalScore *= 1.03
		result.Boosts = append(result.Boosts, "freshness_boost")
	}

	// Apply penalties
	if result.FinalScore <= rr.config.PenaltyThreshold {
		result.FinalScore *= 0.9
		result.Penalties = append(result.Penalties, "low_score_penalty")
	}

	if result.QualityScore <= 0.3 {
		result.FinalScore *= 0.95
		result.Penalties = append(result.Penalties, "quality_penalty")
	}

	// Ensure score doesn't exceed 1.0 or go below 0.0
	result.FinalScore = math.Max(0.0, math.Min(result.FinalScore, 1.0))

	// Log significant changes
	if math.Abs(result.FinalScore-originalScore) > 0.1 {
		if result.Metadata == nil {
			result.Metadata = make(map[string]interface{})
		}
		result.Metadata["score_adjustment"] = result.FinalScore - originalScore
	}
}

// calculateRankingStats calculates statistics about the ranking process
func (rr *ResultRanker) calculateRankingStats(results []RankableResult, duration time.Duration) RankingStats {
	stats := RankingStats{
		RankingTime:       float64(duration.Nanoseconds()) / 1e6, // Convert to milliseconds
		ScoreDistribution: make(map[string]float64),
	}

	if len(results) == 0 {
		return stats
	}

	// Calculate score distribution
	var totalRelevance, totalFreshness, totalAuthority, totalQuality, totalDiversity, totalPersonalization float64
	var diversitySum, qualitySum float64

	for _, result := range results {
		totalRelevance += result.RelevanceScore
		totalFreshness += result.FreshnessScore
		totalAuthority += result.AuthorityScore
		totalQuality += result.QualityScore
		totalDiversity += result.DiversityScore
		totalPersonalization += result.PersonalizationScore

		diversitySum += result.DiversityScore
		qualitySum += result.QualityScore

		// Count boosts and penalties
		stats.BoostCount += len(result.Boosts)
		stats.PenaltyCount += len(result.Penalties)
	}

	count := float64(len(results))
	stats.ScoreDistribution["avg_relevance"] = totalRelevance / count
	stats.ScoreDistribution["avg_freshness"] = totalFreshness / count
	stats.ScoreDistribution["avg_authority"] = totalAuthority / count
	stats.ScoreDistribution["avg_quality"] = totalQuality / count
	stats.ScoreDistribution["avg_diversity"] = totalDiversity / count
	stats.ScoreDistribution["avg_personalization"] = totalPersonalization / count

	stats.DiversityMetric = diversitySum / count
	stats.QualityMetric = qualitySum / count

	return stats
}

// GetConfig returns the current configuration
func (rr *ResultRanker) GetConfig() *ResultRankerConfig {
	return rr.config
}

// UpdateConfig updates the configuration
func (rr *ResultRanker) UpdateConfig(config *ResultRankerConfig) {
	if config != nil {
		rr.config = config
	}
}