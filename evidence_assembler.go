package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// EvidenceAssembler creates structured Evidence objects from search results
type EvidenceAssembler struct {
	config *EvidenceAssemblerConfig
}

// EvidenceAssemblerConfig holds configuration for evidence assembly
type EvidenceAssemblerConfig struct {
	MaxEvidenceItems    int     `json:"max_evidence_items"`    // Maximum evidence items to create
	MinConfidence       float64 `json:"min_confidence"`        // Minimum confidence threshold
	RequireProvenance   bool    `json:"require_provenance"`    // Whether provenance is required
	IncludeRelationMaps bool    `json:"include_relation_maps"` // Whether to include relation maps
	IncludeGraphPaths   bool    `json:"include_graph_paths"`   // Whether to include graph paths
	MaxContentLength    int     `json:"max_content_length"`    // Maximum content length per evidence
	WhySelectedDetail   string  `json:"why_selected_detail"`   // Level of detail for why_selected ("basic", "detailed", "verbose")
	ValidateEvidence    bool    `json:"validate_evidence"`     // Whether to validate evidence
	DeduplicateContent  bool    `json:"deduplicate_content"`   // Whether to deduplicate similar content
	SimilarityThreshold float64 `json:"similarity_threshold"`  // Threshold for content similarity
}

// AssemblyContext provides context for evidence assembly
type AssemblyContext struct {
	Query           string                 `json:"query"`
	QueryTerms      []string               `json:"query_terms"`
	UserID          string                 `json:"user_id,omitempty"`
	SessionID       string                 `json:"session_id,omitempty"`
	RequestTime     time.Time              `json:"request_time"`
	GraphContext    *GraphContext          `json:"graph_context,omitempty"`
	RetrievalMethod string                 `json:"retrieval_method"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// GraphContext provides graph-related context for evidence assembly
type GraphContext struct {
	QueryEntities   []string           `json:"query_entities"`
	RelatedEntities []string           `json:"related_entities"`
	GraphPaths      []Path             `json:"graph_paths,omitempty"`
	Communities     []Community        `json:"communities,omitempty"`
	PageRankScores  map[string]float64 `json:"pagerank_scores,omitempty"`
}

// AssemblyInput represents input for evidence assembly
type AssemblyInput struct {
	ID              string                 `json:"id"`
	Content         string                 `json:"content"`
	Score           float64                `json:"score"`
	Source          string                 `json:"source"`
	Timestamp       time.Time              `json:"timestamp,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	MatchedTerms    []string               `json:"matched_terms,omitempty"`
	GraphPath       []string               `json:"graph_path,omitempty"`
	RelatedEntities []string               `json:"related_entities,omitempty"`
}

// AssemblyResponse contains the assembled evidence and statistics
type AssemblyResponse struct {
	Evidence      []Evidence             `json:"evidence"`
	TotalEvidence int                    `json:"total_evidence"`
	AssemblyStats AssemblyStats          `json:"assembly_stats"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// AssemblyStats contains statistics about the assembly process
type AssemblyStats struct {
	AssemblyTime           float64        `json:"assembly_time_ms"`
	InputCount             int            `json:"input_count"`
	ValidatedCount         int            `json:"validated_count"`
	DeduplicatedCount      int            `json:"deduplicated_count"`
	ProvenanceCount        int            `json:"provenance_count"`
	RelationMapCount       int            `json:"relation_map_count"`
	GraphPathCount         int            `json:"graph_path_count"`
	ConfidenceDistribution map[string]int `json:"confidence_distribution"`
}

// NewEvidenceAssembler creates a new EvidenceAssembler with default configuration
func NewEvidenceAssembler() *EvidenceAssembler {
	config := &EvidenceAssemblerConfig{
		MaxEvidenceItems:    50,
		MinConfidence:       0.1,
		RequireProvenance:   true,
		IncludeRelationMaps: true,
		IncludeGraphPaths:   true,
		MaxContentLength:    2000,
		WhySelectedDetail:   "detailed",
		ValidateEvidence:    true,
		DeduplicateContent:  true,
		SimilarityThreshold: 0.8,
	}

	return &EvidenceAssembler{
		config: config,
	}
}

// NewEvidenceAssemblerWithConfig creates an EvidenceAssembler with custom configuration
func NewEvidenceAssemblerWithConfig(config *EvidenceAssemblerConfig) *EvidenceAssembler {
	if config == nil {
		return NewEvidenceAssembler()
	}

	// Create a new config to avoid modifying the original
	newConfig := *config

	// Set defaults for missing values
	if newConfig.MaxEvidenceItems <= 0 {
		newConfig.MaxEvidenceItems = 50
	}
	if newConfig.MaxContentLength <= 0 {
		newConfig.MaxContentLength = 2000
	}
	if newConfig.WhySelectedDetail == "" {
		newConfig.WhySelectedDetail = "detailed"
	}
	if newConfig.SimilarityThreshold <= 0 {
		newConfig.SimilarityThreshold = 0.8
	}

	return &EvidenceAssembler{
		config: &newConfig,
	}
}

// Assemble creates Evidence objects from assembly inputs
func (ea *EvidenceAssembler) Assemble(ctx context.Context, inputs []AssemblyInput, assemblyCtx *AssemblyContext) (*AssemblyResponse, error) {
	startTime := time.Now()

	if len(inputs) == 0 {
		return &AssemblyResponse{
			Evidence:      []Evidence{},
			TotalEvidence: 0,
			AssemblyStats: AssemblyStats{},
			Metadata:      make(map[string]interface{}),
		}, nil
	}

	if assemblyCtx == nil {
		assemblyCtx = &AssemblyContext{
			RequestTime: time.Now(),
		}
	}

	// Filter inputs by confidence threshold
	filteredInputs := ea.filterByConfidence(inputs)
	confidenceFilteredCount := len(filteredInputs)

	// Deduplicate content if enabled
	if ea.config.DeduplicateContent {
		filteredInputs = ea.deduplicateInputs(filteredInputs)
	}

	// Limit to max evidence items
	if len(filteredInputs) > ea.config.MaxEvidenceItems {
		filteredInputs = filteredInputs[:ea.config.MaxEvidenceItems]
	}

	// Create evidence objects
	evidence := make([]Evidence, 0, len(filteredInputs))
	stats := AssemblyStats{
		InputCount: len(inputs),
	}

	for _, input := range filteredInputs {
		evidenceItem, err := ea.createEvidence(input, assemblyCtx)
		if err != nil {
			continue // Skip invalid evidence
		}

		// Add provenance if required (before validation)
		if ea.config.RequireProvenance {
			ea.AddProvenance(evidenceItem, input, assemblyCtx)
			stats.ProvenanceCount++
		}

		// Validate evidence if enabled
		if ea.config.ValidateEvidence {
			if err := ea.ValidateEvidence(evidenceItem); err != nil {
				continue // Skip invalid evidence
			}
			stats.ValidatedCount++
		}

		// Add relation maps if enabled
		if ea.config.IncludeRelationMaps {
			ea.addRelationMap(evidenceItem, input, assemblyCtx)
			stats.RelationMapCount++
		}

		// Add graph paths if enabled and available
		if ea.config.IncludeGraphPaths && len(input.GraphPath) > 0 {
			evidenceItem.GraphPath = input.GraphPath
			stats.GraphPathCount++
		}

		evidence = append(evidence, *evidenceItem)
	}

	// Sort evidence by confidence (descending)
	sort.Slice(evidence, func(i, j int) bool {
		return evidence[i].Confidence > evidence[j].Confidence
	})

	// Calculate final statistics
	stats.AssemblyTime = float64(time.Since(startTime).Nanoseconds()) / 1e6
	stats.DeduplicatedCount = confidenceFilteredCount - len(filteredInputs)
	stats.ConfidenceDistribution = ea.calculateConfidenceDistribution(evidence)

	response := &AssemblyResponse{
		Evidence:      evidence,
		TotalEvidence: len(evidence),
		AssemblyStats: stats,
		Metadata:      make(map[string]interface{}),
	}

	// Add metadata
	response.Metadata["assembly_config"] = ea.config
	response.Metadata["context_query"] = assemblyCtx.Query
	response.Metadata["retrieval_method"] = assemblyCtx.RetrievalMethod

	return response, nil
}

// AddProvenance adds provenance information to evidence
func (ea *EvidenceAssembler) AddProvenance(evidence *Evidence, input AssemblyInput, assemblyCtx *AssemblyContext) {
	provenance := ProvenanceInfo{
		Source:    input.Source,
		Timestamp: input.Timestamp.Format(time.RFC3339),
		Version:   "1.0",
	}

	if assemblyCtx.UserID != "" {
		provenance.UserID = assemblyCtx.UserID
	}

	// Add additional provenance from metadata
	if input.Metadata != nil {
		if version, exists := input.Metadata["version"]; exists {
			if versionStr, ok := version.(string); ok {
				provenance.Version = versionStr
			}
		}

		if author, exists := input.Metadata["author"]; exists {
			if authorStr, ok := author.(string); ok {
				if provenance.UserID == "" {
					provenance.UserID = authorStr
				}
			}
		}
	}

	evidence.Provenance = provenance
}

// ValidateEvidence validates an evidence object
func (ea *EvidenceAssembler) ValidateEvidence(evidence *Evidence) error {
	if evidence == nil {
		return fmt.Errorf("evidence cannot be nil")
	}

	if evidence.Content == "" {
		return fmt.Errorf("evidence content cannot be empty")
	}

	if evidence.Source == "" {
		return fmt.Errorf("evidence source cannot be empty")
	}

	if evidence.Confidence < 0 || evidence.Confidence > 1 {
		return fmt.Errorf("evidence confidence must be between 0 and 1, got %f", evidence.Confidence)
	}

	if ea.config.RequireProvenance {
		if evidence.Provenance.Source == "" {
			return fmt.Errorf("evidence provenance source is required")
		}
		if evidence.Provenance.Timestamp == "" {
			return fmt.Errorf("evidence provenance timestamp is required")
		}
	}

	// Validate content length
	if len(evidence.Content) > ea.config.MaxContentLength {
		return fmt.Errorf("evidence content exceeds maximum length of %d characters", ea.config.MaxContentLength)
	}

	return nil
}

// createEvidence creates an Evidence object from AssemblyInput
func (ea *EvidenceAssembler) createEvidence(input AssemblyInput, assemblyCtx *AssemblyContext) (*Evidence, error) {
	// Truncate content if too long
	content := input.Content
	if len(content) > ea.config.MaxContentLength {
		content = content[:ea.config.MaxContentLength-3] + "..."
	}

	evidence := &Evidence{
		Content:     content,
		Source:      input.Source,
		Confidence:  input.Score,
		WhySelected: ea.generateWhySelected(input, assemblyCtx),
		RelationMap: make(map[string]string),
		Provenance:  ProvenanceInfo{}, // Will be filled by AddProvenance
		GraphPath:   []string{},       // Will be filled if available
	}

	return evidence, nil
}

// generateWhySelected generates explanation for why evidence was selected
func (ea *EvidenceAssembler) generateWhySelected(input AssemblyInput, _ *AssemblyContext) string {
	var reasons []string

	// Add score-based reason
	if input.Score >= 0.8 {
		reasons = append(reasons, "high relevance score")
	} else if input.Score >= 0.6 {
		reasons = append(reasons, "good relevance score")
	} else {
		reasons = append(reasons, "moderate relevance score")
	}

	// Add term matching reasons
	if len(input.MatchedTerms) > 0 {
		switch ea.config.WhySelectedDetail {
		case "basic":
			reasons = append(reasons, fmt.Sprintf("matches %d query terms", len(input.MatchedTerms)))
		case "detailed":
			if len(input.MatchedTerms) <= 3 {
				reasons = append(reasons, fmt.Sprintf("matches terms: %s", strings.Join(input.MatchedTerms, ", ")))
			} else {
				reasons = append(reasons, fmt.Sprintf("matches %d terms including: %s",
					len(input.MatchedTerms), strings.Join(input.MatchedTerms[:3], ", ")))
			}
		case "verbose":
			reasons = append(reasons, fmt.Sprintf("matches all terms: %s", strings.Join(input.MatchedTerms, ", ")))
		}
	}

	// Add graph-based reasons
	if len(input.GraphPath) > 0 {
		reasons = append(reasons, fmt.Sprintf("connected via %d-hop graph path", len(input.GraphPath)-1))
	}

	if len(input.RelatedEntities) > 0 {
		reasons = append(reasons, fmt.Sprintf("related to %d entities", len(input.RelatedEntities)))
	}

	// Add source-based reasons
	if input.Source != "" {
		if strings.Contains(strings.ToLower(input.Source), "official") {
			reasons = append(reasons, "from official source")
		} else if strings.Contains(strings.ToLower(input.Source), "verified") {
			reasons = append(reasons, "from verified source")
		}
	}

	// Add recency reasons
	if !input.Timestamp.IsZero() {
		age := time.Since(input.Timestamp).Hours() / 24.0
		if age < 1 {
			reasons = append(reasons, "very recent content")
		} else if age < 7 {
			reasons = append(reasons, "recent content")
		}
	}

	if len(reasons) == 0 {
		return "selected based on relevance"
	}

	return strings.Join(reasons, "; ")
}

// addRelationMap adds relation map to evidence
func (ea *EvidenceAssembler) addRelationMap(evidence *Evidence, input AssemblyInput, assemblyCtx *AssemblyContext) {
	if evidence.RelationMap == nil {
		evidence.RelationMap = make(map[string]string)
	}

	// Add related entities
	for _, entity := range input.RelatedEntities {
		evidence.RelationMap[entity] = "related_entity"
	}

	// Add query entities if available in context
	if assemblyCtx.GraphContext != nil {
		for _, entity := range assemblyCtx.GraphContext.QueryEntities {
			evidence.RelationMap[entity] = "query_entity"
		}

		for _, entity := range assemblyCtx.GraphContext.RelatedEntities {
			if _, exists := evidence.RelationMap[entity]; !exists {
				evidence.RelationMap[entity] = "contextual_entity"
			}
		}
	}

	// Add source relation
	if input.Source != "" {
		evidence.RelationMap["source"] = input.Source
	}

	// Add metadata relations
	if input.Metadata != nil {
		if category, exists := input.Metadata["category"]; exists {
			if categoryStr, ok := category.(string); ok {
				evidence.RelationMap["category"] = categoryStr
			}
		}

		if topic, exists := input.Metadata["topic"]; exists {
			if topicStr, ok := topic.(string); ok {
				evidence.RelationMap["topic"] = topicStr
			}
		}
	}
}

// filterByConfidence filters inputs by minimum confidence threshold
func (ea *EvidenceAssembler) filterByConfidence(inputs []AssemblyInput) []AssemblyInput {
	if ea.config.MinConfidence <= 0 {
		return inputs
	}

	var filtered []AssemblyInput
	for _, input := range inputs {
		if input.Score >= ea.config.MinConfidence {
			filtered = append(filtered, input)
		}
	}

	return filtered
}

// deduplicateInputs removes similar content based on similarity threshold
func (ea *EvidenceAssembler) deduplicateInputs(inputs []AssemblyInput) []AssemblyInput {
	if len(inputs) <= 1 {
		return inputs
	}

	var deduplicated []AssemblyInput
	seen := make(map[string]bool)

	for _, input := range inputs {
		isDuplicate := false

		// Check against already selected inputs
		for _, existing := range deduplicated {
			similarity := ea.calculateContentSimilarity(input.Content, existing.Content)
			if similarity >= ea.config.SimilarityThreshold {
				isDuplicate = true
				break
			}
		}

		// Also check for exact content matches
		contentKey := strings.ToLower(strings.TrimSpace(input.Content))
		if seen[contentKey] {
			isDuplicate = true
		}

		if !isDuplicate {
			deduplicated = append(deduplicated, input)
			seen[contentKey] = true
		}
	}

	return deduplicated
}

// calculateContentSimilarity calculates similarity between two content strings
func (ea *EvidenceAssembler) calculateContentSimilarity(content1, content2 string) float64 {
	if content1 == "" || content2 == "" {
		return 0.0
	}

	// Simple Jaccard similarity based on words
	words1 := ea.tokenizeContent(content1)
	words2 := ea.tokenizeContent(content2)

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
func (ea *EvidenceAssembler) tokenizeContent(content string) []string {
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

// calculateConfidenceDistribution calculates distribution of confidence scores
func (ea *EvidenceAssembler) calculateConfidenceDistribution(evidence []Evidence) map[string]int {
	distribution := map[string]int{
		"0.0-0.2": 0,
		"0.2-0.4": 0,
		"0.4-0.6": 0,
		"0.6-0.8": 0,
		"0.8-1.0": 0,
	}

	for _, ev := range evidence {
		switch {
		case ev.Confidence < 0.2:
			distribution["0.0-0.2"]++
		case ev.Confidence < 0.4:
			distribution["0.2-0.4"]++
		case ev.Confidence < 0.6:
			distribution["0.4-0.6"]++
		case ev.Confidence < 0.8:
			distribution["0.6-0.8"]++
		default:
			distribution["0.8-1.0"]++
		}
	}

	return distribution
}

// AssembleFromFusedResults creates evidence from fused search results
func (ea *EvidenceAssembler) AssembleFromFusedResults(ctx context.Context, fusedResults []FusedResult, assemblyCtx *AssemblyContext) (*AssemblyResponse, error) {
	inputs := make([]AssemblyInput, len(fusedResults))

	for i, result := range fusedResults {
		inputs[i] = AssemblyInput{
			ID:       result.ID,
			Content:  result.Content,
			Score:    result.FinalScore,
			Source:   "fused_search",
			Metadata: result.Metadata,
		}

		// Extract matched terms from method scores
		var matchedTerms []string
		for method := range result.MethodScores {
			matchedTerms = append(matchedTerms, method)
		}
		inputs[i].MatchedTerms = matchedTerms
	}

	return ea.Assemble(ctx, inputs, assemblyCtx)
}

// AssembleFromVectorResults creates evidence from vector search results
func (ea *EvidenceAssembler) AssembleFromVectorResults(ctx context.Context, vectorResults []VectorSearchResult, assemblyCtx *AssemblyContext) (*AssemblyResponse, error) {
	inputs := make([]AssemblyInput, len(vectorResults))

	for i, result := range vectorResults {
		inputs[i] = AssemblyInput{
			ID:       result.ID,
			Content:  result.Content,
			Score:    result.Score,
			Source:   "vector_search",
			Metadata: result.Metadata,
		}
	}

	if assemblyCtx == nil {
		assemblyCtx = &AssemblyContext{}
	}
	assemblyCtx.RetrievalMethod = "vector"

	return ea.Assemble(ctx, inputs, assemblyCtx)
}

// AssembleFromKeywordResults creates evidence from keyword search results
func (ea *EvidenceAssembler) AssembleFromKeywordResults(ctx context.Context, keywordResults []KeywordSearchResult, assemblyCtx *AssemblyContext) (*AssemblyResponse, error) {
	inputs := make([]AssemblyInput, len(keywordResults))

	for i, result := range keywordResults {
		inputs[i] = AssemblyInput{
			ID:           result.ID,
			Content:      result.Content,
			Score:        result.Score,
			Source:       "keyword_search",
			Metadata:     result.Metadata,
			MatchedTerms: result.MatchedTerms,
		}
	}

	if assemblyCtx == nil {
		assemblyCtx = &AssemblyContext{}
	}
	assemblyCtx.RetrievalMethod = "keyword"

	return ea.Assemble(ctx, inputs, assemblyCtx)
}

// GetConfig returns the current configuration
func (ea *EvidenceAssembler) GetConfig() *EvidenceAssemblerConfig {
	return ea.config
}

// UpdateConfig updates the configuration
func (ea *EvidenceAssembler) UpdateConfig(config *EvidenceAssemblerConfig) {
	if config != nil {
		ea.config = config
	}
}
