package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// RecallResponseFormatter formats recall responses for MCP output
type RecallResponseFormatter struct {
	config *RecallFormatterConfig
}

// RecallFormatterConfig holds configuration for response formatting
type RecallFormatterConfig struct {
	MaxEvidenceLength    int     `json:"max_evidence_length"`
	TruncateContent      bool    `json:"truncate_content"`
	IncludeMetadata      bool    `json:"include_metadata"`
	IncludeDebugInfo     bool    `json:"include_debug_info"`
	SortByConfidence     bool    `json:"sort_by_confidence"`
	MinConfidenceDisplay float64 `json:"min_confidence_display"`
	FormatTimestamps     bool    `json:"format_timestamps"`
	GroupBySimilarity    bool    `json:"group_by_similarity"`
}

// FormattingContext provides context for formatting decisions
type FormattingContext struct {
	OriginalQuery   string                 `json:"original_query"`
	RequestTime     time.Time              `json:"request_time"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	UserPreferences map[string]interface{} `json:"user_preferences"`
}

// NewRecallResponseFormatter creates a new formatter with default configuration
func NewRecallResponseFormatter() *RecallResponseFormatter {
	config := &RecallFormatterConfig{
		MaxEvidenceLength:    500,
		TruncateContent:      true,
		IncludeMetadata:      true,
		IncludeDebugInfo:     false,
		SortByConfidence:     true,
		MinConfidenceDisplay: 0.0,
		FormatTimestamps:     true,
		GroupBySimilarity:    false,
	}

	return &RecallResponseFormatter{
		config: config,
	}
}

// NewRecallResponseFormatterWithConfig creates a formatter with custom configuration
func NewRecallResponseFormatterWithConfig(config *RecallFormatterConfig) *RecallResponseFormatter {
	if config == nil {
		return NewRecallResponseFormatter()
	}

	return &RecallResponseFormatter{
		config: config,
	}
}

// Format formats a RecallResponse into a RecallResult for MCP output
func (f *RecallResponseFormatter) Format(response *RecallResponse, args RecallArgs) (RecallResult, error) {
	if response == nil {
		return RecallResult{}, fmt.Errorf("response cannot be nil")
	}

	context := &FormattingContext{
		OriginalQuery: args.Query,
		RequestTime:   time.Now(),
	}

	// Format evidence
	formattedEvidence, err := f.formatEvidence(response.Evidence, context)
	if err != nil {
		return RecallResult{}, fmt.Errorf("failed to format evidence: %w", err)
	}

	// Format community cards
	formattedCommunityCards := f.formatCommunityCards(response.CommunityCards, context)

	// Format conflicts
	formattedConflicts := f.formatConflicts(response.Conflicts, context)

	// Format retrieval stats
	formattedStats := f.formatRetrievalStats(response.RetrievalStats, context)

	// Format self-critique
	formattedCritique := f.formatSelfCritique(response.SelfCritique, context)

	result := RecallResult{
		Evidence:       formattedEvidence,
		CommunityCards: formattedCommunityCards,
		Conflicts:      formattedConflicts,
		Stats:          formattedStats,
		SelfCritique:   formattedCritique,
	}

	return result, nil
}

// BuildRecallResult builds a RecallResult from individual components
func (f *RecallResponseFormatter) BuildRecallResult(evidence []Evidence, communityCards []CommunityCard, conflicts []ConflictInfo, stats RetrievalStats) RecallResult {
	context := &FormattingContext{
		RequestTime: time.Now(),
	}

	// Format each component
	formattedEvidence, _ := f.formatEvidence(evidence, context)
	formattedCommunityCards := f.formatCommunityCards(communityCards, context)
	formattedConflicts := f.formatConflicts(conflicts, context)
	formattedStats := f.formatRetrievalStats(stats, context)

	return RecallResult{
		Evidence:       formattedEvidence,
		CommunityCards: formattedCommunityCards,
		Conflicts:      formattedConflicts,
		Stats:          formattedStats,
	}
}

// AddMetadata adds metadata to the formatted result
func (f *RecallResponseFormatter) AddMetadata(result *RecallResult, metadata map[string]interface{}) {
	if !f.config.IncludeMetadata || metadata == nil {
		return
	}

	// Add metadata to evidence items
	for i := range result.Evidence {
		if result.Evidence[i].RelationMap == nil {
			result.Evidence[i].RelationMap = make(map[string]string)
		}

		// Add relevant metadata as relation map entries
		for key, value := range metadata {
			if f.isRelevantMetadata(key) {
				result.Evidence[i].RelationMap[fmt.Sprintf("meta_%s", key)] = fmt.Sprintf("%v", value)
			}
		}
	}

	// Add processing metadata to stats
	if processingTime, ok := metadata["processing_time"].(time.Duration); ok {
		result.Stats.QueryTime = processingTime.Milliseconds()
	}
}

// formatEvidence formats evidence items
func (f *RecallResponseFormatter) formatEvidence(evidence []Evidence, context *FormattingContext) ([]Evidence, error) {
	if len(evidence) == 0 {
		return []Evidence{}, nil
	}

	// Filter by minimum confidence
	filtered := f.filterByConfidence(evidence)

	// Sort if configured
	if f.config.SortByConfidence {
		f.sortByConfidence(filtered)
	}

	// Group by similarity if configured
	if f.config.GroupBySimilarity {
		filtered = f.groupBySimilarity(filtered)
	}

	// Format each evidence item
	formatted := make([]Evidence, len(filtered))
	for i, e := range filtered {
		formatted[i] = f.formatEvidenceItem(e, context)
	}

	return formatted, nil
}

// formatEvidenceItem formats a single evidence item
func (f *RecallResponseFormatter) formatEvidenceItem(evidence Evidence, context *FormattingContext) Evidence {
	formatted := evidence

	// Truncate content if configured
	if f.config.TruncateContent && len(evidence.Content) > f.config.MaxEvidenceLength {
		formatted.Content = evidence.Content[:f.config.MaxEvidenceLength] + "..."
	}

	// Format timestamps in provenance
	if f.config.FormatTimestamps {
		formatted.Provenance = f.formatProvenance(evidence.Provenance)
	}

	// Enhance why_selected explanation
	formatted.WhySelected = f.enhanceWhySelected(evidence.WhySelected, context)

	// Clean up relation map
	formatted.RelationMap = f.cleanRelationMap(evidence.RelationMap)

	return formatted
}

// formatCommunityCards formats community cards
func (f *RecallResponseFormatter) formatCommunityCards(cards []CommunityCard, context *FormattingContext) []CommunityCard {
	if len(cards) == 0 {
		return []CommunityCard{}
	}

	formatted := make([]CommunityCard, len(cards))
	for i, card := range cards {
		formatted[i] = f.formatCommunityCard(card, context)
	}

	// Sort by entity count (descending)
	sort.Slice(formatted, func(i, j int) bool {
		return formatted[i].EntityCount > formatted[j].EntityCount
	})

	return formatted
}

// formatCommunityCard formats a single community card
func (f *RecallResponseFormatter) formatCommunityCard(card CommunityCard, _ *FormattingContext) CommunityCard {
	formatted := card

	// Ensure title is not empty
	if formatted.Title == "" {
		formatted.Title = fmt.Sprintf("Community %s", card.ID)
	}

	// Truncate summary if too long
	if len(formatted.Summary) > 200 {
		formatted.Summary = formatted.Summary[:200] + "..."
	}

	// Limit entities list
	if len(formatted.Entities) > 10 {
		formatted.Entities = formatted.Entities[:10]
	}

	return formatted
}

// formatConflicts formats conflict information
func (f *RecallResponseFormatter) formatConflicts(conflicts []ConflictInfo, context *FormattingContext) []ConflictInfo {
	if len(conflicts) == 0 {
		return []ConflictInfo{}
	}

	formatted := make([]ConflictInfo, len(conflicts))
	for i, conflict := range conflicts {
		formatted[i] = f.formatConflict(conflict, context)
	}

	// Sort by severity
	sort.Slice(formatted, func(i, j int) bool {
		return f.getSeverityWeight(formatted[i].Severity) > f.getSeverityWeight(formatted[j].Severity)
	})

	return formatted
}

// formatConflict formats a single conflict
func (f *RecallResponseFormatter) formatConflict(conflict ConflictInfo, context *FormattingContext) ConflictInfo {
	formatted := conflict

	// Enhance description with context
	if context.OriginalQuery != "" {
		formatted.Description = fmt.Sprintf("%s (related to query: %s)",
			conflict.Description, f.truncateQuery(context.OriginalQuery))
	}

	return formatted
}

// formatRetrievalStats formats retrieval statistics
func (f *RecallResponseFormatter) formatRetrievalStats(stats RetrievalStats, _ *FormattingContext) RetrievalStats {
	formatted := stats

	// Ensure all counts are non-negative
	if formatted.VectorResults < 0 {
		formatted.VectorResults = 0
	}
	if formatted.GraphResults < 0 {
		formatted.GraphResults = 0
	}
	if formatted.SearchResults < 0 {
		formatted.SearchResults = 0
	}
	if formatted.TotalCandidates < 0 {
		formatted.TotalCandidates = 0
	}

	// Ensure fusion score is in valid range
	if formatted.FusionScore < 0 {
		formatted.FusionScore = 0
	}
	if formatted.FusionScore > 1 {
		formatted.FusionScore = 1
	}

	return formatted
}

// formatSelfCritique formats self-critique text
func (f *RecallResponseFormatter) formatSelfCritique(critique string, context *FormattingContext) string {
	if critique == "" {
		return ""
	}

	// Add context if available
	if context.OriginalQuery != "" {
		return fmt.Sprintf("Query: \"%s\" - %s",
			f.truncateQuery(context.OriginalQuery), critique)
	}

	return critique
}

// formatProvenance formats provenance information
func (f *RecallResponseFormatter) formatProvenance(provenance ProvenanceInfo) ProvenanceInfo {
	formatted := provenance

	// Format timestamp if it's a valid time string
	if formatted.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, formatted.Timestamp); err == nil {
			formatted.Timestamp = t.Format("2006-01-02 15:04:05")
		}
	}

	return formatted
}

// enhanceWhySelected enhances the why_selected explanation
func (f *RecallResponseFormatter) enhanceWhySelected(whySelected string, context *FormattingContext) string {
	if whySelected == "" {
		return "Selected based on relevance to query"
	}

	// Add query context if not already present
	if context.OriginalQuery != "" && !strings.Contains(whySelected, "query") {
		return fmt.Sprintf("%s for query: \"%s\"", whySelected, f.truncateQuery(context.OriginalQuery))
	}

	return whySelected
}

// cleanRelationMap cleans up the relation map
func (f *RecallResponseFormatter) cleanRelationMap(relationMap map[string]string) map[string]string {
	if relationMap == nil {
		return make(map[string]string)
	}

	cleaned := make(map[string]string)
	for key, value := range relationMap {
		// Skip empty keys or values
		if key == "" || value == "" {
			continue
		}

		// Clean up key names
		cleanKey := strings.ReplaceAll(key, "_", " ")
		cleanKey = strings.Title(cleanKey)

		cleaned[cleanKey] = value
	}

	return cleaned
}

// filterByConfidence filters evidence by minimum confidence
func (f *RecallResponseFormatter) filterByConfidence(evidence []Evidence) []Evidence {
	if f.config.MinConfidenceDisplay <= 0 {
		return evidence
	}

	var filtered []Evidence
	for _, e := range evidence {
		if e.Confidence >= f.config.MinConfidenceDisplay {
			filtered = append(filtered, e)
		}
	}

	return filtered
}

// sortByConfidence sorts evidence by confidence score (descending)
func (f *RecallResponseFormatter) sortByConfidence(evidence []Evidence) {
	sort.Slice(evidence, func(i, j int) bool {
		return evidence[i].Confidence > evidence[j].Confidence
	})
}

// groupBySimilarity groups similar evidence items
func (f *RecallResponseFormatter) groupBySimilarity(evidence []Evidence) []Evidence {
	// Simple similarity grouping based on source
	sourceGroups := make(map[string][]Evidence)

	for _, e := range evidence {
		sourceGroups[e.Source] = append(sourceGroups[e.Source], e)
	}

	// Flatten back to slice, keeping highest confidence from each source
	var grouped []Evidence
	for _, group := range sourceGroups {
		if len(group) == 1 {
			grouped = append(grouped, group[0])
		} else {
			// Sort by confidence and take the best
			sort.Slice(group, func(i, j int) bool {
				return group[i].Confidence > group[j].Confidence
			})
			grouped = append(grouped, group[0])
		}
	}

	return grouped
}

// getSeverityWeight returns a numeric weight for severity levels
func (f *RecallResponseFormatter) getSeverityWeight(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

// truncateQuery truncates a query for display purposes
func (f *RecallResponseFormatter) truncateQuery(query string) string {
	if len(query) <= 50 {
		return query
	}
	return query[:47] + "..."
}

// isRelevantMetadata checks if metadata key is relevant for display
func (f *RecallResponseFormatter) isRelevantMetadata(key string) bool {
	relevantKeys := []string{
		"processing_time", "method", "score", "rank", "source_type",
		"confidence", "timestamp", "version",
	}

	for _, relevant := range relevantKeys {
		if key == relevant {
			return true
		}
	}

	return false
}

// GetConfig returns the current configuration
func (f *RecallResponseFormatter) GetConfig() *RecallFormatterConfig {
	return f.config
}

// UpdateConfig updates the configuration
func (f *RecallResponseFormatter) UpdateConfig(config *RecallFormatterConfig) {
	if config != nil {
		f.config = config
	}
}

// FormatForDisplay formats a RecallResult for human-readable display
func (f *RecallResponseFormatter) FormatForDisplay(result RecallResult) string {
	var builder strings.Builder

	builder.WriteString("=== Memory Recall Results ===\n")
	builder.WriteString(fmt.Sprintf("Found %d evidence items\n\n", len(result.Evidence)))

	// Format evidence
	for i, evidence := range result.Evidence {
		builder.WriteString(fmt.Sprintf("Evidence %d (Confidence: %.2f):\n", i+1, evidence.Confidence))
		builder.WriteString(fmt.Sprintf("  Content: %s\n", evidence.Content))
		builder.WriteString(fmt.Sprintf("  Source: %s\n", evidence.Source))
		builder.WriteString(fmt.Sprintf("  Why Selected: %s\n", evidence.WhySelected))

		if len(evidence.RelationMap) > 0 {
			builder.WriteString("  Relations:\n")
			for key, value := range evidence.RelationMap {
				builder.WriteString(fmt.Sprintf("    %s: %s\n", key, value))
			}
		}
		builder.WriteString("\n")
	}

	// Format conflicts if any
	if len(result.Conflicts) > 0 {
		builder.WriteString("=== Conflicts Detected ===\n")
		for _, conflict := range result.Conflicts {
			builder.WriteString(fmt.Sprintf("- %s (%s): %s\n",
				conflict.Type, conflict.Severity, conflict.Description))
		}
		builder.WriteString("\n")
	}

	// Format stats
	builder.WriteString("=== Retrieval Statistics ===\n")
	builder.WriteString(fmt.Sprintf("Query Time: %dms\n", result.Stats.QueryTime))
	builder.WriteString(fmt.Sprintf("Vector Results: %d\n", result.Stats.VectorResults))
	builder.WriteString(fmt.Sprintf("Graph Results: %d\n", result.Stats.GraphResults))
	builder.WriteString(fmt.Sprintf("Search Results: %d\n", result.Stats.SearchResults))
	builder.WriteString(fmt.Sprintf("Fusion Score: %.3f\n", result.Stats.FusionScore))

	// Add self-critique if available
	if result.SelfCritique != "" {
		builder.WriteString("\n=== Self-Critique ===\n")
		builder.WriteString(result.SelfCritique)
		builder.WriteString("\n")
	}

	return builder.String()
}
