package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// QueryProcessor handles query parsing, expansion, and processing
type QueryProcessor struct {
	vectorSearcher  *VectorSearcher
	keywordSearcher *KeywordSearcher
	queryExpander   *QueryExpander
	config          *QueryProcessorConfig
}

// QueryProcessorConfig holds configuration for query processing
type QueryProcessorConfig struct {
	MaxResults      int           `json:"max_results"`
	DefaultTimeout  time.Duration `json:"default_timeout"`
	EnableExpansion bool          `json:"enable_expansion"`
	MinQueryLength  int           `json:"min_query_length"`
}

// ProcessedQuery represents a parsed and expanded query
type ProcessedQuery struct {
	Original    string            `json:"original"`
	Parsed      *ParsedQuery      `json:"parsed"`
	Expanded    []string          `json:"expanded"`
	Entities    []string          `json:"entities"`
	Keywords    []string          `json:"keywords"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ParsedQuery contains structured query components
type ParsedQuery struct {
	Terms       []string          `json:"terms"`
	Phrases     []string          `json:"phrases"`
	Filters     map[string]string `json:"filters"`
	TimeRange   *TimeRange        `json:"time_range,omitempty"`
	QueryType   QueryType         `json:"query_type"`
}

// TimeRange represents a time-based filter
type TimeRange struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// QueryType indicates the type of query
type QueryType string

const (
	QueryTypeKeyword   QueryType = "keyword"
	QueryTypeEntity    QueryType = "entity"
	QueryTypeSemantic  QueryType = "semantic"
	QueryTypeHybrid    QueryType = "hybrid"
)

// NewQueryProcessor creates a new QueryProcessor instance
func NewQueryProcessor(config *QueryProcessorConfig) *QueryProcessor {
	if config == nil {
		config = &QueryProcessorConfig{
			MaxResults:      20,
			DefaultTimeout:  5 * time.Second,
			EnableExpansion: true,
			MinQueryLength:  2,
		}
	}

	return &QueryProcessor{
		vectorSearcher:  NewVectorSearcher(),
		keywordSearcher: NewKeywordSearcher(),
		queryExpander:   NewQueryExpander(),
		config:          config,
	}
}

// Process handles the complete query processing pipeline
func (qp *QueryProcessor) Process(ctx context.Context, query string, options *RecallOptions) (*ProcessedQuery, error) {
	if len(strings.TrimSpace(query)) < qp.config.MinQueryLength {
		return nil, fmt.Errorf("query too short: minimum length is %d characters", qp.config.MinQueryLength)
	}

	// Parse the query
	parsed, err := qp.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	// Expand the query if enabled
	var expanded []string
	if qp.config.EnableExpansion {
		expanded, err = qp.Expand(ctx, query, parsed)
		if err != nil {
			// Log error but continue with original query
			expanded = []string{query}
		}
	} else {
		expanded = []string{query}
	}

	// Extract entities and keywords
	entities := qp.extractEntities(parsed)
	keywords := qp.extractKeywords(parsed)

	processedQuery := &ProcessedQuery{
		Original: query,
		Parsed:   parsed,
		Expanded: expanded,
		Entities: entities,
		Keywords: keywords,
		Metadata: make(map[string]interface{}),
	}

	// Add processing metadata
	processedQuery.Metadata["processed_at"] = time.Now()
	processedQuery.Metadata["expansion_count"] = len(expanded)
	processedQuery.Metadata["entity_count"] = len(entities)

	return processedQuery, nil
}

// Parse extracts structured components from a query string
func (qp *QueryProcessor) Parse(query string) (*ParsedQuery, error) {
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}

	parsed := &ParsedQuery{
		Terms:     make([]string, 0),
		Phrases:   make([]string, 0),
		Filters:   make(map[string]string),
		QueryType: QueryTypeKeyword,
	}

	// Clean and normalize query
	query = strings.TrimSpace(query)
	query = strings.ToLower(query)

	// Extract phrases (quoted text)
	phrases := qp.extractPhrases(query)
	parsed.Phrases = phrases

	// Remove phrases from query to extract individual terms
	queryWithoutPhrases := qp.removePhrases(query)

	// Extract individual terms
	terms := strings.Fields(queryWithoutPhrases)
	parsed.Terms = qp.filterTerms(terms)

	// Extract filters (key:value pairs)
	filters := qp.extractFilters(query)
	parsed.Filters = filters

	// Extract time range if present
	timeRange := qp.extractTimeRange(query)
	parsed.TimeRange = timeRange

	// Determine query type
	parsed.QueryType = qp.determineQueryType(parsed)

	return parsed, nil
}

// Expand generates additional query variations for better recall
func (qp *QueryProcessor) Expand(ctx context.Context, originalQuery string, parsed *ParsedQuery) ([]string, error) {
	if qp.queryExpander == nil {
		return []string{originalQuery}, nil
	}

	expanded, err := qp.queryExpander.Expand(ctx, originalQuery, parsed)
	if err != nil {
		return []string{originalQuery}, err
	}

	// Always include the original query
	result := []string{originalQuery}
	result = append(result, expanded...)

	// Remove duplicates and limit results
	result = qp.deduplicateQueries(result)
	if len(result) > 10 { // Limit expansion to prevent performance issues
		result = result[:10]
	}

	return result, nil
}

// extractPhrases finds quoted phrases in the query
func (qp *QueryProcessor) extractPhrases(query string) []string {
	var phrases []string
	inQuotes := false
	var currentPhrase strings.Builder

	for i, char := range query {
		if char == '"' {
			if inQuotes {
				// End of phrase
				phrase := strings.TrimSpace(currentPhrase.String())
				if phrase != "" {
					phrases = append(phrases, phrase)
				}
				currentPhrase.Reset()
				inQuotes = false
			} else {
				// Start of phrase
				inQuotes = true
			}
		} else if inQuotes {
			currentPhrase.WriteRune(char)
		}
	}

	return phrases
}

// removePhrases removes quoted phrases from query
func (qp *QueryProcessor) removePhrases(query string) string {
	inQuotes := false
	var result strings.Builder

	for _, char := range query {
		if char == '"' {
			inQuotes = !inQuotes
		} else if !inQuotes {
			result.WriteRune(char)
		}
	}

	return result.String()
}

// extractFilters finds key:value filter pairs
func (qp *QueryProcessor) extractFilters(query string) map[string]string {
	filters := make(map[string]string)
	
	// Simple regex-like extraction for key:value pairs
	words := strings.Fields(query)
	for _, word := range words {
		if strings.Contains(word, ":") {
			parts := strings.SplitN(word, ":", 2)
			if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
				filters[parts[0]] = parts[1]
			}
		}
	}

	return filters
}

// extractTimeRange attempts to extract time-based filters
func (qp *QueryProcessor) extractTimeRange(query string) *TimeRange {
	// Simple implementation - could be enhanced with more sophisticated parsing
	timeKeywords := []string{"today", "yesterday", "last week", "last month", "recent"}
	
	for _, keyword := range timeKeywords {
		if strings.Contains(query, keyword) {
			// Return a basic time range based on keyword
			now := time.Now()
			switch keyword {
			case "today":
				start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				return &TimeRange{Start: &start, End: &now}
			case "yesterday":
				yesterday := now.AddDate(0, 0, -1)
				start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
				end := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 0, yesterday.Location())
				return &TimeRange{Start: &start, End: &end}
			case "last week":
				weekAgo := now.AddDate(0, 0, -7)
				return &TimeRange{Start: &weekAgo, End: &now}
			case "last month":
				monthAgo := now.AddDate(0, -1, 0)
				return &TimeRange{Start: &monthAgo, End: &now}
			case "recent":
				recent := now.AddDate(0, 0, -3) // Last 3 days
				return &TimeRange{Start: &recent, End: &now}
			}
		}
	}

	return nil
}

// filterTerms removes stop words and short terms
func (qp *QueryProcessor) filterTerms(terms []string) []string {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
	}

	var filtered []string
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if len(term) >= 2 && !stopWords[term] {
			filtered = append(filtered, term)
		}
	}

	return filtered
}

// determineQueryType classifies the query type
func (qp *QueryProcessor) determineQueryType(parsed *ParsedQuery) QueryType {
	// Simple heuristics for query type classification
	if len(parsed.Phrases) > 0 {
		return QueryTypeSemantic
	}
	
	if len(parsed.Filters) > 0 {
		return QueryTypeHybrid
	}

	// Check if query looks like entity search
	if len(parsed.Terms) <= 2 && len(parsed.Terms) > 0 {
		// Could be entity search
		for _, term := range parsed.Terms {
			if len(term) > 0 && strings.ToUpper(term[:1]) == term[:1] {
				return QueryTypeEntity
			}
		}
	}

	return QueryTypeKeyword
}

// extractEntities identifies potential entities in the parsed query
func (qp *QueryProcessor) extractEntities(parsed *ParsedQuery) []string {
	var entities []string

	// Look for capitalized terms (simple entity detection)
	for _, term := range parsed.Terms {
		if len(term) > 0 && strings.ToUpper(term[:1]) == term[:1] {
			entities = append(entities, term)
		}
	}

	// Add phrases as potential entities
	entities = append(entities, parsed.Phrases...)

	return qp.deduplicateStrings(entities)
}

// extractKeywords gets important keywords from the parsed query
func (qp *QueryProcessor) extractKeywords(parsed *ParsedQuery) []string {
	keywords := make([]string, 0)
	
	// Add all terms as keywords
	keywords = append(keywords, parsed.Terms...)
	
	// Add phrases as keywords
	keywords = append(keywords, parsed.Phrases...)

	return qp.deduplicateStrings(keywords)
}

// deduplicateQueries removes duplicate queries while preserving order
func (qp *QueryProcessor) deduplicateQueries(queries []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, query := range queries {
		if !seen[query] {
			seen[query] = true
			result = append(result, query)
		}
	}

	return result
}

// deduplicateStrings removes duplicate strings while preserving order
func (qp *QueryProcessor) deduplicateStrings(strings []string) []string {
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