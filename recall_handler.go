package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RecallHandler handles memory recall operations
type RecallHandler struct {
	queryProcessor *QueryProcessor
	resultFuser    *ResultFuser
	validator      *RecallArgsValidator
	formatter      *RecallResponseFormatter
	config         *RecallHandlerConfig
}

// RecallHandlerConfig holds configuration for the recall handler
type RecallHandlerConfig struct {
	DefaultMaxResults    int           `json:"default_max_results"`
	DefaultTimeBudget    time.Duration `json:"default_time_budget"`
	MaxTimeBudget        time.Duration `json:"max_time_budget"`
	EnableSelfCritique   bool          `json:"enable_self_critique"`
	EnableQueryExpansion bool          `json:"enable_query_expansion"`
}

// NewRecallHandler creates a new RecallHandler instance
func NewRecallHandler(queryProcessor *QueryProcessor, resultFuser *ResultFuser) *RecallHandler {
	config := &RecallHandlerConfig{
		DefaultMaxResults:    10,
		DefaultTimeBudget:    5 * time.Second,
		MaxTimeBudget:        30 * time.Second,
		EnableSelfCritique:   true,
		EnableQueryExpansion: true,
	}

	return &RecallHandler{
		queryProcessor: queryProcessor,
		resultFuser:    resultFuser,
		validator:      NewRecallArgsValidator(),
		formatter:      NewRecallResponseFormatter(),
		config:         config,
	}
}

// NewRecallHandlerWithConfig creates a RecallHandler with custom configuration
func NewRecallHandlerWithConfig(queryProcessor *QueryProcessor, resultFuser *ResultFuser, config *RecallHandlerConfig) *RecallHandler {
	if config == nil {
		return NewRecallHandler(queryProcessor, resultFuser)
	}

	return &RecallHandler{
		queryProcessor: queryProcessor,
		resultFuser:    resultFuser,
		validator:      NewRecallArgsValidator(),
		formatter:      NewRecallResponseFormatter(),
		config:         config,
	}
}

// HandleRecall processes a memory recall request
func (rh *RecallHandler) HandleRecall(ctx context.Context, req *mcp.CallToolRequest, args RecallArgs) (*mcp.CallToolResult, RecallResult, error) {
	startTime := time.Now()

	log.Printf("Handling recall request: query=%s, maxResults=%d", args.Query, args.MaxResults)

	// Validate arguments
	if err := rh.validator.Validate(args); err != nil {
		return nil, RecallResult{}, fmt.Errorf("argument validation failed: %w", err)
	}

	// Sanitize and prepare arguments
	sanitizedArgs, err := rh.validator.SanitizeInput(args)
	if err != nil {
		return nil, RecallResult{}, fmt.Errorf("input sanitization failed: %w", err)
	}

	// Convert to recall options
	options := rh.convertArgsToOptions(sanitizedArgs)

	// Set timeout context
	queryCtx, cancel := context.WithTimeout(ctx, options.TimeBudget)
	defer cancel()

	// Process the query
	response, err := rh.processQuery(queryCtx, sanitizedArgs.Query, options)
	if err != nil {
		return nil, RecallResult{}, fmt.Errorf("query processing failed: %w", err)
	}

	// Format the response
	result, err := rh.formatter.Format(response, sanitizedArgs)
	if err != nil {
		return nil, RecallResult{}, fmt.Errorf("response formatting failed: %w", err)
	}

	// Add processing metadata
	result.Stats.QueryTime = time.Since(startTime).Milliseconds()

	// Create MCP result
	mcpResult := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Retrieved %d pieces of evidence for query: %s",
					len(result.Evidence), args.Query),
			},
		},
	}

	log.Printf("Recall completed in %v, returned %d evidence items",
		time.Since(startTime), len(result.Evidence))

	return mcpResult, result, nil
}

// processQuery handles the core query processing logic
func (rh *RecallHandler) processQuery(ctx context.Context, query string, options *RecallOptions) (*RecallResponse, error) {
	// Process the query through the query processor
	processedQuery, err := rh.queryProcessor.Process(ctx, query, options)
	if err != nil {
		return nil, fmt.Errorf("query processing failed: %w", err)
	}

	// Create response structure
	response := NewRecallResponse()
	response.QueryExpansions = processedQuery.Expanded

	// Perform multi-view retrieval
	fusionInputs, err := rh.performMultiViewRetrieval(ctx, processedQuery, options)
	if err != nil {
		return nil, fmt.Errorf("multi-view retrieval failed: %w", err)
	}

	// Fuse results if we have multiple inputs
	if len(fusionInputs) > 0 {
		fusionResponse, err := rh.resultFuser.Fuse(ctx, fusionInputs)
		if err != nil {
			return nil, fmt.Errorf("result fusion failed: %w", err)
		}

		// Convert fused results to evidence
		evidence, err := rh.convertFusedResultsToEvidence(fusionResponse.Results)
		if err != nil {
			return nil, fmt.Errorf("evidence conversion failed: %w", err)
		}

		response.Evidence = evidence
		response.TotalResults = len(evidence)

		// Update retrieval stats
		response.RetrievalStats.FusionScore = rh.calculateAverageFusionScore(fusionResponse.Results)
		response.RetrievalStats.TotalCandidates = fusionResponse.TotalResults
	}

	// Add self-critique if enabled
	if rh.config.EnableSelfCritique && len(response.Evidence) > 0 {
		critique, err := rh.generateSelfCritique(ctx, query, response.Evidence)
		if err != nil {
			log.Printf("Self-critique generation failed: %v", err)
		} else {
			response.SelfCritique = critique
		}
	}

	// Detect conflicts if we have multiple evidence items
	if len(response.Evidence) > 1 {
		conflicts := rh.detectConflicts(response.Evidence)
		response.Conflicts = conflicts
	}

	return response, nil
}

// performMultiViewRetrieval executes retrieval across multiple methods
func (rh *RecallHandler) performMultiViewRetrieval(ctx context.Context, processedQuery *ProcessedQuery, options *RecallOptions) ([]FusionInput, error) {
	var fusionInputs []FusionInput

	// Vector search (if available)
	if rh.queryProcessor.vectorSearcher != nil {
		vectorResults, err := rh.performVectorSearch(ctx, processedQuery, options)
		if err != nil {
			log.Printf("Vector search failed: %v", err)
		} else if len(vectorResults) > 0 {
			fusionInputs = append(fusionInputs, rh.convertVectorResultsToFusionInput(vectorResults))
		}
	}

	// Keyword search (if available)
	if rh.queryProcessor.keywordSearcher != nil {
		keywordResults, err := rh.performKeywordSearch(ctx, processedQuery, options)
		if err != nil {
			log.Printf("Keyword search failed: %v", err)
		} else if len(keywordResults) > 0 {
			fusionInputs = append(fusionInputs, rh.convertKeywordResultsToFusionInput(keywordResults))
		}
	}

	// Graph search (placeholder - would be implemented when graph components are available)
	// This would be implemented in later tasks when graph algorithms are available

	return fusionInputs, nil
}

// performVectorSearch executes vector-based similarity search
func (rh *RecallHandler) performVectorSearch(_ context.Context, _ *ProcessedQuery, _ *RecallOptions) ([]VectorSearchResult, error) {
	// For now, return empty results since we don't have embeddings
	// This would be implemented when we have actual embedding generation
	return []VectorSearchResult{}, nil
}

// performKeywordSearch executes keyword-based text search
func (rh *RecallHandler) performKeywordSearch(ctx context.Context, processedQuery *ProcessedQuery, options *RecallOptions) ([]KeywordSearchResult, error) {
	// Use the original query for keyword search
	response, err := rh.queryProcessor.keywordSearcher.Search(ctx, processedQuery.Original, options.MaxResults, options.Filters)
	if err != nil {
		return []KeywordSearchResult{}, err
	}

	return response.Results, nil
}

// convertVectorResultsToFusionInput converts vector search results to fusion input
func (rh *RecallHandler) convertVectorResultsToFusionInput(results []VectorSearchResult) FusionInput {
	fusionItems := make([]FusionItem, len(results))
	for i, result := range results {
		fusionItems[i] = FusionItem{
			ID:       result.ID,
			Content:  result.Content,
			Score:    result.Score,
			Rank:     i + 1,
			Metadata: result.Metadata,
		}
	}

	return FusionInput{
		Method:  "vector",
		Results: fusionItems,
		Weight:  1.0,
		Metadata: map[string]interface{}{
			"search_type":  "vector_similarity",
			"result_count": len(results),
		},
	}
}

// convertKeywordResultsToFusionInput converts keyword search results to fusion input
func (rh *RecallHandler) convertKeywordResultsToFusionInput(results []KeywordSearchResult) FusionInput {
	fusionItems := make([]FusionItem, len(results))
	for i, result := range results {
		fusionItems[i] = FusionItem{
			ID:       result.ID,
			Content:  result.Content,
			Score:    result.Score,
			Rank:     i + 1,
			Metadata: result.Metadata,
		}
	}

	return FusionInput{
		Method:  "keyword",
		Results: fusionItems,
		Weight:  1.0,
		Metadata: map[string]interface{}{
			"search_type":  "keyword_matching",
			"result_count": len(results),
		},
	}
}

// convertFusedResultsToEvidence converts fused results to evidence format
func (rh *RecallHandler) convertFusedResultsToEvidence(fusedResults []FusedResult) ([]Evidence, error) {
	evidence := make([]Evidence, len(fusedResults))

	for i, result := range fusedResults {
		// Extract provenance information from metadata
		provenance := ProvenanceInfo{
			Source:    "unknown",
			Timestamp: time.Now().Format(time.RFC3339),
			Version:   "1.0.0",
		}

		if source, ok := result.Metadata["source"].(string); ok {
			provenance.Source = source
		}
		if timestamp, ok := result.Metadata["timestamp"].(string); ok {
			provenance.Timestamp = timestamp
		}
		if version, ok := result.Metadata["version"].(string); ok {
			provenance.Version = version
		}
		if userID, ok := result.Metadata["user_id"].(string); ok {
			provenance.UserID = userID
		}

		// Generate why_selected explanation
		whySelected := rh.generateWhySelectedExplanation(result)

		// Create relation map from source methods
		relationMap := make(map[string]string)
		for _, method := range result.SourceMethods {
			relationMap[method] = fmt.Sprintf("score_%.3f", result.MethodScores[method])
		}

		evidence[i] = Evidence{
			Content:     result.Content,
			Source:      provenance.Source,
			Confidence:  result.FinalScore,
			WhySelected: whySelected,
			RelationMap: relationMap,
			Provenance:  provenance,
		}
	}

	return evidence, nil
}

// generateWhySelectedExplanation creates an explanation for why evidence was selected
func (rh *RecallHandler) generateWhySelectedExplanation(result FusedResult) string {
	methods := result.SourceMethods
	if len(methods) == 0 {
		return "Selected based on relevance score"
	}

	if len(methods) == 1 {
		return fmt.Sprintf("Selected via %s search (score: %.3f)", methods[0], result.FinalScore)
	}

	return fmt.Sprintf("Selected via multi-view fusion of %v (combined score: %.3f)",
		methods, result.FinalScore)
}

// calculateAverageFusionScore calculates the average fusion score
func (rh *RecallHandler) calculateAverageFusionScore(results []FusedResult) float64 {
	if len(results) == 0 {
		return 0.0
	}

	var total float64
	for _, result := range results {
		total += result.FinalScore
	}

	return total / float64(len(results))
}

// generateSelfCritique generates a self-critique of the retrieval results
func (rh *RecallHandler) generateSelfCritique(_ context.Context, _ string, evidence []Evidence) (string, error) {
	if len(evidence) == 0 {
		return "No evidence found for the query. Consider expanding search terms or checking data availability.", nil
	}

	// Simple heuristic-based critique
	avgConfidence := 0.0
	for _, e := range evidence {
		avgConfidence += e.Confidence
	}
	avgConfidence /= float64(len(evidence))

	if avgConfidence < 0.3 {
		return fmt.Sprintf("Retrieved %d evidence items but confidence is low (avg: %.2f). Results may not be highly relevant to the query.",
			len(evidence), avgConfidence), nil
	}

	if avgConfidence > 0.8 {
		return fmt.Sprintf("Retrieved %d high-confidence evidence items (avg: %.2f). Results appear highly relevant to the query.",
			len(evidence), avgConfidence), nil
	}

	return fmt.Sprintf("Retrieved %d evidence items with moderate confidence (avg: %.2f). Results are reasonably relevant but may benefit from query refinement.",
		len(evidence), avgConfidence), nil
}

// detectConflicts identifies potential conflicts between evidence items
func (rh *RecallHandler) detectConflicts(evidence []Evidence) []ConflictInfo {
	var conflicts []ConflictInfo

	// Simple conflict detection based on source diversity and confidence differences
	sourceMap := make(map[string][]int)
	for i, e := range evidence {
		sourceMap[e.Source] = append(sourceMap[e.Source], i)
	}

	// Look for evidence from different sources with significantly different confidence scores
	for i := 0; i < len(evidence); i++ {
		for j := i + 1; j < len(evidence); j++ {
			if evidence[i].Source != evidence[j].Source {
				confidenceDiff := evidence[i].Confidence - evidence[j].Confidence
				if confidenceDiff > 0.5 || confidenceDiff < -0.5 {
					conflicts = append(conflicts, ConflictInfo{
						ID:   fmt.Sprintf("conflict_%d_%d", i, j),
						Type: "confidence_mismatch",
						Description: fmt.Sprintf("Significant confidence difference between sources %s and %s",
							evidence[i].Source, evidence[j].Source),
						ConflictingIDs: []string{fmt.Sprintf("evidence_%d", i), fmt.Sprintf("evidence_%d", j)},
						Severity:       "medium",
					})
				}
			}
		}
	}

	return conflicts
}

// convertArgsToOptions converts RecallArgs to RecallOptions
func (rh *RecallHandler) convertArgsToOptions(args RecallArgs) *RecallOptions {
	options := NewRecallOptions()

	if args.MaxResults > 0 {
		options.MaxResults = args.MaxResults
	} else {
		options.MaxResults = rh.config.DefaultMaxResults
	}

	if args.TimeBudget > 0 {
		timeBudget := time.Duration(args.TimeBudget) * time.Millisecond
		if timeBudget > rh.config.MaxTimeBudget {
			timeBudget = rh.config.MaxTimeBudget
		}
		options.TimeBudget = timeBudget
	} else {
		options.TimeBudget = rh.config.DefaultTimeBudget
	}

	options.IncludeGraph = args.IncludeGraph
	options.Filters = args.Filters
	options.ExpandQuery = rh.config.EnableQueryExpansion

	return options
}

// GetConfig returns the current configuration
func (rh *RecallHandler) GetConfig() *RecallHandlerConfig {
	return rh.config
}

// UpdateConfig updates the configuration
func (rh *RecallHandler) UpdateConfig(config *RecallHandlerConfig) {
	if config != nil {
		rh.config = config
	}
}
