package main

import (
	"fmt"
	"time"
)

// RecallOptions represents options for memory recall operations
type RecallOptions struct {
	MaxResults    int                    `json:"max_results"`
	TimeBudget    time.Duration          `json:"time_budget"`
	IncludeGraph  bool                   `json:"include_graph"`
	Filters       map[string]interface{} `json:"filters"`
	MinConfidence float64                `json:"min_confidence"`
	SortBy        string                 `json:"sort_by"`        // "relevance", "confidence", "date"
	SortOrder     string                 `json:"sort_order"`     // "asc", "desc"
	ExpandQuery   bool                   `json:"expand_query"`   // Whether to expand query with synonyms
	UseCache      bool                   `json:"use_cache"`      // Whether to use cached results
}

// WriteMetadata represents metadata for memory write operations
type WriteMetadata struct {
	Source          string                 `json:"source"`
	Timestamp       time.Time              `json:"timestamp"`
	UserID          string                 `json:"user_id,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
	Confidence      float64                `json:"confidence"`
	RequireEvidence bool                   `json:"require_evidence"`
	Language        string                 `json:"language,omitempty"`
	ContentType     string                 `json:"content_type,omitempty"`
	Version         string                 `json:"version,omitempty"`
	ExternalID      string                 `json:"external_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ManageOptions represents options for memory management operations
type ManageOptions struct {
	Operation     string    `json:"operation"`      // "pin", "forget", "decay", "merge"
	MemoryIDs     []string  `json:"memory_ids"`
	Query         string    `json:"query,omitempty"`
	Confidence    float64   `json:"confidence,omitempty"`
	TTL           time.Duration `json:"ttl,omitempty"`
	Force         bool      `json:"force"`          // Force operation even if conflicts exist
	DryRun        bool      `json:"dry_run"`        // Preview operation without executing
	BatchSize     int       `json:"batch_size"`     // Number of items to process in each batch
}

// SearchOptions represents options for search operations
type SearchOptions struct {
	Query         string                 `json:"query"`
	MaxResults    int                    `json:"max_results"`
	Offset        int                    `json:"offset"`
	Filters       map[string]interface{} `json:"filters"`
	MinConfidence float64                `json:"min_confidence"`
	SearchType    string                 `json:"search_type"`    // "vector", "text", "hybrid"
	IncludeScore  bool                   `json:"include_score"`
	Highlight     bool                   `json:"highlight"`      // Highlight matching terms
}

// GraphOptions represents options for graph operations
type GraphOptions struct {
	MaxDepth      int      `json:"max_depth"`
	EdgeTypes     []string `json:"edge_types,omitempty"`
	NodeTypes     []string `json:"node_types,omitempty"`
	MinWeight     float64  `json:"min_weight"`
	MaxNodes      int      `json:"max_nodes"`
	IncludeProps  bool     `json:"include_properties"`
	Algorithm     string   `json:"algorithm,omitempty"`    // "pagerank", "community", "shortest_path"
}

// NewRecallOptions creates a new RecallOptions with default values
func NewRecallOptions() *RecallOptions {
	return &RecallOptions{
		MaxResults:    10,
		TimeBudget:    5 * time.Second,
		IncludeGraph:  false,
		Filters:       make(map[string]interface{}),
		MinConfidence: 0.0,
		SortBy:        "relevance",
		SortOrder:     "desc",
		ExpandQuery:   true,
		UseCache:      true,
	}
}

// NewWriteMetadata creates a new WriteMetadata with default values
func NewWriteMetadata(source string) *WriteMetadata {
	return &WriteMetadata{
		Source:          source,
		Timestamp:       time.Now(),
		Tags:            make([]string, 0),
		Confidence:      1.0,
		RequireEvidence: false,
		Language:        "en",
		ContentType:     "text/plain",
		Version:         "1.0",
		Metadata:        make(map[string]interface{}),
	}
}

// NewManageOptions creates a new ManageOptions with default values
func NewManageOptions(operation string) *ManageOptions {
	return &ManageOptions{
		Operation:  operation,
		MemoryIDs:  make([]string, 0),
		Confidence: 0.0,
		TTL:        24 * time.Hour,
		Force:      false,
		DryRun:     false,
		BatchSize:  100,
	}
}

// NewSearchOptions creates a new SearchOptions with default values
func NewSearchOptions(query string) *SearchOptions {
	return &SearchOptions{
		Query:         query,
		MaxResults:    10,
		Offset:        0,
		Filters:       make(map[string]interface{}),
		MinConfidence: 0.0,
		SearchType:    "hybrid",
		IncludeScore:  true,
		Highlight:     false,
	}
}

// NewGraphOptions creates a new GraphOptions with default values
func NewGraphOptions() *GraphOptions {
	return &GraphOptions{
		MaxDepth:     3,
		EdgeTypes:    make([]string, 0),
		NodeTypes:    make([]string, 0),
		MinWeight:    0.0,
		MaxNodes:     100,
		IncludeProps: true,
	}
}

// Validate validates the RecallOptions
func (r *RecallOptions) Validate() error {
	if r.MaxResults <= 0 {
		return fmt.Errorf("max results must be positive, got %d", r.MaxResults)
	}
	if r.MaxResults > 1000 {
		return fmt.Errorf("max results cannot exceed 1000, got %d", r.MaxResults)
	}
	if r.TimeBudget <= 0 {
		return fmt.Errorf("time budget must be positive, got %v", r.TimeBudget)
	}
	if r.MinConfidence < 0 || r.MinConfidence > 1 {
		return fmt.Errorf("min confidence must be between 0 and 1, got %f", r.MinConfidence)
	}
	if r.SortBy != "" && r.SortBy != "relevance" && r.SortBy != "confidence" && r.SortBy != "date" {
		return fmt.Errorf("invalid sort by value: %s", r.SortBy)
	}
	if r.SortOrder != "" && r.SortOrder != "asc" && r.SortOrder != "desc" {
		return fmt.Errorf("invalid sort order value: %s", r.SortOrder)
	}
	return nil
}

// Validate validates the WriteMetadata
func (w *WriteMetadata) Validate() error {
	if w.Source == "" {
		return fmt.Errorf("source cannot be empty")
	}
	if w.Timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
	}
	if w.Confidence < 0 || w.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1, got %f", w.Confidence)
	}
	if w.Language != "" && len(w.Language) != 2 {
		return fmt.Errorf("language must be a 2-character code, got %s", w.Language)
	}
	return nil
}

// Validate validates the ManageOptions
func (m *ManageOptions) Validate() error {
	if m.Operation == "" {
		return fmt.Errorf("operation cannot be empty")
	}
	validOps := []string{"pin", "forget", "decay", "merge"}
	validOp := false
	for _, op := range validOps {
		if m.Operation == op {
			validOp = true
			break
		}
	}
	if !validOp {
		return fmt.Errorf("invalid operation: %s, must be one of %v", m.Operation, validOps)
	}
	if m.Confidence < 0 || m.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1, got %f", m.Confidence)
	}
	if m.TTL < 0 {
		return fmt.Errorf("TTL cannot be negative, got %v", m.TTL)
	}
	if m.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive, got %d", m.BatchSize)
	}
	return nil
}

// Validate validates the SearchOptions
func (s *SearchOptions) Validate() error {
	if s.Query == "" {
		return fmt.Errorf("query cannot be empty")
	}
	if s.MaxResults <= 0 {
		return fmt.Errorf("max results must be positive, got %d", s.MaxResults)
	}
	if s.MaxResults > 1000 {
		return fmt.Errorf("max results cannot exceed 1000, got %d", s.MaxResults)
	}
	if s.Offset < 0 {
		return fmt.Errorf("offset cannot be negative, got %d", s.Offset)
	}
	if s.MinConfidence < 0 || s.MinConfidence > 1 {
		return fmt.Errorf("min confidence must be between 0 and 1, got %f", s.MinConfidence)
	}
	if s.SearchType != "" && s.SearchType != "vector" && s.SearchType != "text" && s.SearchType != "hybrid" {
		return fmt.Errorf("invalid search type: %s", s.SearchType)
	}
	return nil
}

// Validate validates the GraphOptions
func (g *GraphOptions) Validate() error {
	if g.MaxDepth <= 0 {
		return fmt.Errorf("max depth must be positive, got %d", g.MaxDepth)
	}
	if g.MaxDepth > 10 {
		return fmt.Errorf("max depth cannot exceed 10, got %d", g.MaxDepth)
	}
	if g.MinWeight < 0 {
		return fmt.Errorf("min weight cannot be negative, got %f", g.MinWeight)
	}
	if g.MaxNodes <= 0 {
		return fmt.Errorf("max nodes must be positive, got %d", g.MaxNodes)
	}
	if g.MaxNodes > 10000 {
		return fmt.Errorf("max nodes cannot exceed 10000, got %d", g.MaxNodes)
	}
	if g.Algorithm != "" && g.Algorithm != "pagerank" && g.Algorithm != "community" && g.Algorithm != "shortest_path" {
		return fmt.Errorf("invalid algorithm: %s", g.Algorithm)
	}
	return nil
}

// SetFilter sets a filter value
func (r *RecallOptions) SetFilter(key string, value interface{}) {
	if r.Filters == nil {
		r.Filters = make(map[string]interface{})
	}
	r.Filters[key] = value
}

// GetFilter gets a filter value
func (r *RecallOptions) GetFilter(key string) (interface{}, bool) {
	if r.Filters == nil {
		return nil, false
	}
	value, exists := r.Filters[key]
	return value, exists
}

// AddTag adds a tag to the metadata
func (w *WriteMetadata) AddTag(tag string) {
	if w.Tags == nil {
		w.Tags = make([]string, 0)
	}
	// Check if tag already exists
	for _, existingTag := range w.Tags {
		if existingTag == tag {
			return
		}
	}
	w.Tags = append(w.Tags, tag)
}

// RemoveTag removes a tag from the metadata
func (w *WriteMetadata) RemoveTag(tag string) {
	if w.Tags == nil {
		return
	}
	for i, existingTag := range w.Tags {
		if existingTag == tag {
			w.Tags = append(w.Tags[:i], w.Tags[i+1:]...)
			return
		}
	}
}

// HasTag checks if a tag exists in the metadata
func (w *WriteMetadata) HasTag(tag string) bool {
	if w.Tags == nil {
		return false
	}
	for _, existingTag := range w.Tags {
		if existingTag == tag {
			return true
		}
	}
	return false
}

// SetMetadata sets a metadata value
func (w *WriteMetadata) SetMetadata(key string, value interface{}) {
	if w.Metadata == nil {
		w.Metadata = make(map[string]interface{})
	}
	w.Metadata[key] = value
}

// GetMetadata gets a metadata value
func (w *WriteMetadata) GetMetadata(key string) (interface{}, bool) {
	if w.Metadata == nil {
		return nil, false
	}
	value, exists := w.Metadata[key]
	return value, exists
}