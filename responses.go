package main

import (
	"fmt"
	"time"
)

// RecallResponse represents the response from a memory recall operation
type RecallResponse struct {
	Evidence        []Evidence         `json:"evidence"`
	CommunityCards  []CommunityCard    `json:"community_cards,omitempty"`
	Conflicts       []ConflictInfo     `json:"conflicts,omitempty"`
	RetrievalStats  RetrievalStats     `json:"retrieval_stats"`
	SelfCritique    string             `json:"self_critique,omitempty"`
	QueryExpansions []string           `json:"query_expansions,omitempty"`
	TotalResults    int                `json:"total_results"`
	HasMore         bool               `json:"has_more"`
	NextOffset      int                `json:"next_offset,omitempty"`
}

// WriteResponse represents the response from a memory write operation
type WriteResponse struct {
	MemoryID       string         `json:"memory_id"`
	CandidateCount int            `json:"candidate_count"`
	ConflictsFound []ConflictInfo `json:"conflicts_found,omitempty"`
	EntitiesLinked []string       `json:"entities_linked"`
	ProvenanceID   string         `json:"provenance_id"`
	ChunksCreated  int            `json:"chunks_created"`
	GraphUpdates   GraphUpdates   `json:"graph_updates"`
	ProcessingTime time.Duration  `json:"processing_time"`
	Warnings       []string       `json:"warnings,omitempty"`
}

// ManageResponse represents the response from a memory management operation
type ManageResponse struct {
	Operation      string        `json:"operation"`
	AffectedCount  int           `json:"affected_count"`
	Success        bool          `json:"success"`
	Message        string        `json:"message"`
	ProcessingTime time.Duration `json:"processing_time"`
	Errors         []string      `json:"errors,omitempty"`
	Preview        []string      `json:"preview,omitempty"` // For dry-run operations
}

// StatsResponse represents the response from a memory stats operation
type StatsResponse struct {
	TotalMemories    int                    `json:"total_memories"`
	GraphNodes       int                    `json:"graph_nodes"`
	GraphEdges       int                    `json:"graph_edges"`
	VectorDimensions int                    `json:"vector_dimensions"`
	StorageUsage     StorageUsageStats      `json:"storage_usage"`
	PerformanceStats PerformanceStats       `json:"performance_stats"`
	MemoryStats      MemoryStats            `json:"memory_stats"`
	SystemHealth     SystemHealthStats      `json:"system_health"`
	Timestamp        time.Time              `json:"timestamp"`
}

// SearchResponse represents the response from a search operation
type SearchResponse struct {
	Results        []SearchResult    `json:"results"`
	TotalResults   int               `json:"total_results"`
	HasMore        bool              `json:"has_more"`
	NextOffset     int               `json:"next_offset,omitempty"`
	SearchStats    SearchStats       `json:"search_stats"`
	Suggestions    []string          `json:"suggestions,omitempty"`
	Facets         map[string][]Facet `json:"facets,omitempty"`
}

// GraphResponse represents the response from a graph operation
type GraphResponse struct {
	Nodes          []Node            `json:"nodes"`
	Edges          []Edge            `json:"edges"`
	Paths          []Path            `json:"paths,omitempty"`
	Communities    []Community       `json:"communities,omitempty"`
	Metrics        GraphMetrics      `json:"metrics"`
	Algorithm      string            `json:"algorithm,omitempty"`
	ProcessingTime time.Duration     `json:"processing_time"`
}

// Supporting response structures
type GraphUpdates struct {
	NodesCreated int `json:"nodes_created"`
	NodesUpdated int `json:"nodes_updated"`
	EdgesCreated int `json:"edges_created"`
	EdgesUpdated int `json:"edges_updated"`
}

type StorageUsageStats struct {
	VectorStore map[string]interface{} `json:"vector_store"`
	GraphStore  map[string]interface{} `json:"graph_store"`
	SearchIndex map[string]interface{} `json:"search_index"`
	TotalSize   int64                  `json:"total_size_bytes"`
}

type PerformanceStats struct {
	AvgQueryTime    time.Duration `json:"avg_query_time"`
	CacheHitRate    float64       `json:"cache_hit_rate"`
	MemoryUsage     int64         `json:"memory_usage_bytes"`
	ActiveConns     int           `json:"active_connections"`
	RequestsPerSec  float64       `json:"requests_per_second"`
	ErrorRate       float64       `json:"error_rate"`
}

type MemoryStats struct {
	ByType        map[string]int    `json:"by_type"`
	BySource      map[string]int    `json:"by_source"`
	ByConfidence  map[string]int    `json:"by_confidence"`
	ByAge         map[string]int    `json:"by_age"`
	TopEntities   []EntityStat      `json:"top_entities"`
	RecentUpdates []RecentUpdate    `json:"recent_updates"`
}

type SystemHealthStats struct {
	Status           string            `json:"status"` // "healthy", "degraded", "unhealthy"
	Uptime           time.Duration     `json:"uptime"`
	LastHealthCheck  time.Time         `json:"last_health_check"`
	ComponentHealth  map[string]string `json:"component_health"`
	Alerts           []Alert           `json:"alerts,omitempty"`
}

type SearchStats struct {
	QueryTime       time.Duration `json:"query_time"`
	VectorResults   int           `json:"vector_results"`
	TextResults     int           `json:"text_results"`
	FusionScore     float64       `json:"fusion_score"`
	CacheHit        bool          `json:"cache_hit"`
}

type GraphMetrics struct {
	NodeCount       int     `json:"node_count"`
	EdgeCount       int     `json:"edge_count"`
	Density         float64 `json:"density"`
	AvgDegree       float64 `json:"avg_degree"`
	ClusteringCoeff float64 `json:"clustering_coefficient"`
	Diameter        int     `json:"diameter,omitempty"`
}

type Facet struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type EntityStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	Type  string `json:"type"`
}

type RecentUpdate struct {
	MemoryID  string    `json:"memory_id"`
	Operation string    `json:"operation"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
}

type Alert struct {
	Level     string    `json:"level"`     // "info", "warning", "error", "critical"
	Message   string    `json:"message"`
	Component string    `json:"component"`
	Timestamp time.Time `json:"timestamp"`
}

// NewRecallResponse creates a new RecallResponse with default values
func NewRecallResponse() *RecallResponse {
	return &RecallResponse{
		Evidence:        make([]Evidence, 0),
		CommunityCards:  make([]CommunityCard, 0),
		Conflicts:       make([]ConflictInfo, 0),
		QueryExpansions: make([]string, 0),
		TotalResults:    0,
		HasMore:         false,
		NextOffset:      0,
	}
}

// NewWriteResponse creates a new WriteResponse with default values
func NewWriteResponse(memoryID string) *WriteResponse {
	return &WriteResponse{
		MemoryID:       memoryID,
		CandidateCount: 0,
		ConflictsFound: make([]ConflictInfo, 0),
		EntitiesLinked: make([]string, 0),
		ProvenanceID:   "",
		ChunksCreated:  0,
		GraphUpdates:   GraphUpdates{},
		ProcessingTime: 0,
		Warnings:       make([]string, 0),
	}
}

// NewManageResponse creates a new ManageResponse with default values
func NewManageResponse(operation string) *ManageResponse {
	return &ManageResponse{
		Operation:      operation,
		AffectedCount:  0,
		Success:        false,
		Message:        "",
		ProcessingTime: 0,
		Errors:         make([]string, 0),
		Preview:        make([]string, 0),
	}
}

// NewStatsResponse creates a new StatsResponse with default values
func NewStatsResponse() *StatsResponse {
	return &StatsResponse{
		TotalMemories:    0,
		GraphNodes:       0,
		GraphEdges:       0,
		VectorDimensions: 0,
		StorageUsage:     StorageUsageStats{},
		PerformanceStats: PerformanceStats{},
		MemoryStats:      MemoryStats{},
		SystemHealth:     SystemHealthStats{},
		Timestamp:        time.Now(),
	}
}

// NewSearchResponse creates a new SearchResponse with default values
func NewSearchResponse() *SearchResponse {
	return &SearchResponse{
		Results:      make([]SearchResult, 0),
		TotalResults: 0,
		HasMore:      false,
		NextOffset:   0,
		SearchStats:  SearchStats{},
		Suggestions:  make([]string, 0),
		Facets:       make(map[string][]Facet),
	}
}

// NewGraphResponse creates a new GraphResponse with default values
func NewGraphResponse() *GraphResponse {
	return &GraphResponse{
		Nodes:          make([]Node, 0),
		Edges:          make([]Edge, 0),
		Paths:          make([]Path, 0),
		Communities:    make([]Community, 0),
		Metrics:        GraphMetrics{},
		ProcessingTime: 0,
	}
}

// Validation methods
func (r *RecallResponse) Validate() error {
	if r.Evidence == nil {
		return fmt.Errorf("evidence cannot be nil")
	}
	if r.TotalResults < 0 {
		return fmt.Errorf("total results cannot be negative, got %d", r.TotalResults)
	}
	if r.NextOffset < 0 {
		return fmt.Errorf("next offset cannot be negative, got %d", r.NextOffset)
	}
	return nil
}

func (w *WriteResponse) Validate() error {
	if w.MemoryID == "" {
		return fmt.Errorf("memory ID cannot be empty")
	}
	if w.CandidateCount < 0 {
		return fmt.Errorf("candidate count cannot be negative, got %d", w.CandidateCount)
	}
	if w.ChunksCreated < 0 {
		return fmt.Errorf("chunks created cannot be negative, got %d", w.ChunksCreated)
	}
	if w.ProcessingTime < 0 {
		return fmt.Errorf("processing time cannot be negative, got %v", w.ProcessingTime)
	}
	return nil
}

func (m *ManageResponse) Validate() error {
	if m.Operation == "" {
		return fmt.Errorf("operation cannot be empty")
	}
	if m.AffectedCount < 0 {
		return fmt.Errorf("affected count cannot be negative, got %d", m.AffectedCount)
	}
	if m.ProcessingTime < 0 {
		return fmt.Errorf("processing time cannot be negative, got %v", m.ProcessingTime)
	}
	return nil
}

func (s *StatsResponse) Validate() error {
	if s.TotalMemories < 0 {
		return fmt.Errorf("total memories cannot be negative, got %d", s.TotalMemories)
	}
	if s.GraphNodes < 0 {
		return fmt.Errorf("graph nodes cannot be negative, got %d", s.GraphNodes)
	}
	if s.GraphEdges < 0 {
		return fmt.Errorf("graph edges cannot be negative, got %d", s.GraphEdges)
	}
	if s.VectorDimensions < 0 {
		return fmt.Errorf("vector dimensions cannot be negative, got %d", s.VectorDimensions)
	}
	if s.Timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
	}
	return nil
}

func (s *SearchResponse) Validate() error {
	if s.Results == nil {
		return fmt.Errorf("results cannot be nil")
	}
	if s.TotalResults < 0 {
		return fmt.Errorf("total results cannot be negative, got %d", s.TotalResults)
	}
	if s.NextOffset < 0 {
		return fmt.Errorf("next offset cannot be negative, got %d", s.NextOffset)
	}
	return nil
}

func (g *GraphResponse) Validate() error {
	if g.Nodes == nil {
		return fmt.Errorf("nodes cannot be nil")
	}
	if g.Edges == nil {
		return fmt.Errorf("edges cannot be nil")
	}
	if g.ProcessingTime < 0 {
		return fmt.Errorf("processing time cannot be negative, got %v", g.ProcessingTime)
	}
	return nil
}

// Helper methods
func (r *RecallResponse) AddEvidence(evidence Evidence) {
	if r.Evidence == nil {
		r.Evidence = make([]Evidence, 0)
	}
	r.Evidence = append(r.Evidence, evidence)
	r.TotalResults = len(r.Evidence)
}

func (r *RecallResponse) AddConflict(conflict ConflictInfo) {
	if r.Conflicts == nil {
		r.Conflicts = make([]ConflictInfo, 0)
	}
	r.Conflicts = append(r.Conflicts, conflict)
}

func (w *WriteResponse) AddWarning(warning string) {
	if w.Warnings == nil {
		w.Warnings = make([]string, 0)
	}
	w.Warnings = append(w.Warnings, warning)
}

func (w *WriteResponse) AddEntityLink(entityID string) {
	if w.EntitiesLinked == nil {
		w.EntitiesLinked = make([]string, 0)
	}
	// Check if entity already exists
	for _, existing := range w.EntitiesLinked {
		if existing == entityID {
			return
		}
	}
	w.EntitiesLinked = append(w.EntitiesLinked, entityID)
}

func (m *ManageResponse) AddError(err string) {
	if m.Errors == nil {
		m.Errors = make([]string, 0)
	}
	m.Errors = append(m.Errors, err)
	m.Success = false
}

func (m *ManageResponse) AddPreviewItem(item string) {
	if m.Preview == nil {
		m.Preview = make([]string, 0)
	}
	m.Preview = append(m.Preview, item)
}

func (s *SearchResponse) AddResult(result SearchResult) {
	if s.Results == nil {
		s.Results = make([]SearchResult, 0)
	}
	s.Results = append(s.Results, result)
	s.TotalResults = len(s.Results)
}

func (s *SearchResponse) AddSuggestion(suggestion string) {
	if s.Suggestions == nil {
		s.Suggestions = make([]string, 0)
	}
	s.Suggestions = append(s.Suggestions, suggestion)
}

func (g *GraphResponse) AddNode(node Node) {
	if g.Nodes == nil {
		g.Nodes = make([]Node, 0)
	}
	g.Nodes = append(g.Nodes, node)
	g.Metrics.NodeCount = len(g.Nodes)
}

func (g *GraphResponse) AddEdge(edge Edge) {
	if g.Edges == nil {
		g.Edges = make([]Edge, 0)
	}
	g.Edges = append(g.Edges, edge)
	g.Metrics.EdgeCount = len(g.Edges)
}

func (g *GraphResponse) AddPath(path Path) {
	if g.Paths == nil {
		g.Paths = make([]Path, 0)
	}
	g.Paths = append(g.Paths, path)
}

func (g *GraphResponse) AddCommunity(community Community) {
	if g.Communities == nil {
		g.Communities = make([]Community, 0)
	}
	g.Communities = append(g.Communities, community)
}