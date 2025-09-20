package main

import (
	"context"
	"time"
)

// VectorResult represents a result from vector similarity search
type VectorResult struct {
	ID         string                 `json:"id"`
	Score      float64                `json:"score"`
	Metadata   map[string]interface{} `json:"metadata"`
	Embedding  []float32              `json:"embedding,omitempty"`
	Content    string                 `json:"content,omitempty"`
}

// SearchResult represents a result from text search
type SearchResult struct {
	ID         string                 `json:"id"`
	Score      float64                `json:"score"`
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata"`
	Highlights []string               `json:"highlights,omitempty"`
}

// Path represents a path between nodes in the graph
type Path struct {
	Nodes []string  `json:"nodes"`
	Edges []string  `json:"edges"`
	Cost  float64   `json:"cost"`
}

// Community represents a community detected in the graph
type Community struct {
	ID      string   `json:"id"`
	Nodes   []string `json:"nodes"`
	Score   float64  `json:"score"`
	Summary string   `json:"summary,omitempty"`
}

// GraphTraversalOptions configures graph traversal operations
type GraphTraversalOptions struct {
	MaxDepth    int               `json:"max_depth"`
	MaxResults  int               `json:"max_results"`
	EdgeTypes   []EdgeType        `json:"edge_types,omitempty"`
	NodeTypes   []NodeType        `json:"node_types,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
}

// PageRankOptions configures PageRank algorithm execution
type PageRankOptions struct {
	Alpha       float64   `json:"alpha"`
	MaxIter     int       `json:"max_iter"`
	Tolerance   float64   `json:"tolerance"`
	Seeds       []string  `json:"seeds,omitempty"`
}

// VectorStore interface defines operations for vector similarity search
type VectorStore interface {
	// Store stores a vector with associated metadata
	Store(ctx context.Context, id string, embedding []float32, metadata map[string]interface{}) error
	
	// Search performs similarity search and returns top k results
	Search(ctx context.Context, query []float32, k int, filters map[string]interface{}) ([]VectorResult, error)
	
	// Delete removes a vector by ID
	Delete(ctx context.Context, id string) error
	
	// BatchStore stores multiple vectors in a single operation
	BatchStore(ctx context.Context, items []VectorStoreItem) error
	
	// Update updates the metadata for an existing vector
	Update(ctx context.Context, id string, metadata map[string]interface{}) error
	
	// GetByID retrieves a vector by ID
	GetByID(ctx context.Context, id string) (*VectorResult, error)
	
	// Count returns the total number of vectors stored
	Count(ctx context.Context) (int64, error)
	
	// Close closes the vector store connection
	Close() error
	
	// Health checks if the vector store is healthy
	Health(ctx context.Context) error
}

// GraphStore interface defines operations for graph storage and computation
type GraphStore interface {
	// CreateNode creates a new node in the graph
	CreateNode(ctx context.Context, node *Node) error
	
	// CreateEdge creates a new edge in the graph
	CreateEdge(ctx context.Context, edge *Edge) error
	
	// GetNode retrieves a node by ID
	GetNode(ctx context.Context, id string) (*Node, error)
	
	// GetEdge retrieves an edge by ID
	GetEdge(ctx context.Context, id string) (*Edge, error)
	
	// UpdateNode updates an existing node
	UpdateNode(ctx context.Context, node *Node) error
	
	// UpdateEdge updates an existing edge
	UpdateEdge(ctx context.Context, edge *Edge) error
	
	// DeleteNode deletes a node and all its edges
	DeleteNode(ctx context.Context, id string) error
	
	// DeleteEdge deletes an edge by ID
	DeleteEdge(ctx context.Context, id string) error
	
	// FindPaths finds paths between two nodes with traversal options
	FindPaths(ctx context.Context, from, to string, options GraphTraversalOptions) ([]Path, error)
	
	// PageRank computes PageRank scores for nodes with options
	PageRank(ctx context.Context, options PageRankOptions) (map[string]float64, error)
	
	// CommunityDetection performs community detection using Louvain algorithm
	CommunityDetection(ctx context.Context) ([]Community, error)
	
	// GetNeighbors returns neighboring nodes of a given node
	GetNeighbors(ctx context.Context, nodeID string, options GraphTraversalOptions) ([]Node, error)
	
	// FindEdgesByType finds edges by type with optional filters
	FindEdgesByType(ctx context.Context, edgeType EdgeType, filters map[string]interface{}) ([]*Edge, error)
	
	// FindNodesByType finds nodes by type with optional filters
	FindNodesByType(ctx context.Context, nodeType NodeType, filters map[string]interface{}) ([]*Node, error)
	
	// NodeCount returns the total number of nodes
	NodeCount(ctx context.Context) (int64, error)
	
	// EdgeCount returns the total number of edges
	EdgeCount(ctx context.Context) (int64, error)
	
	// Close closes the graph store connection
	Close() error
	
	// Health checks if the graph store is healthy
	Health(ctx context.Context) error
}

// IndexDocument represents a document to be indexed
type IndexDocument struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// SearchIndexOptions configures search operations
type SearchIndexOptions struct {
	Offset     int                    `json:"offset"`
	Limit      int                    `json:"limit"`
	Filters    map[string]interface{} `json:"filters,omitempty"`
	SortBy     string                 `json:"sort_by,omitempty"`
	SortOrder  string                 `json:"sort_order,omitempty"`
	Highlight  bool                   `json:"highlight"`
}

// VectorStoreConfig holds vector store specific configuration
type VectorStoreConfig struct {
	Provider    string            `json:"provider"`
	Dimensions  int               `json:"dimensions"`
	IndexType   string            `json:"index_type"`
	Metric      string            `json:"metric"`
	Parameters  map[string]interface{} `json:"parameters"`
	Timeout     time.Duration     `json:"timeout"`
}

// GraphStoreConfig holds graph store specific configuration
type GraphStoreConfig struct {
	Provider   string            `json:"provider"`
	URI        string            `json:"uri"`
	Database   string            `json:"database"`
	Username   string            `json:"username"`
	Password   string            `json:"password"`
	Parameters map[string]interface{} `json:"parameters"`
	Timeout    time.Duration     `json:"timeout"`
}

// SearchIndexConfig holds search index specific configuration
type SearchIndexConfig struct {
	Provider   string            `json:"provider"`
	URI        string            `json:"uri"`
	IndexName  string            `json:"index_name"`
	Shards     int               `json:"shards"`
	Replicas   int               `json:"replicas"`
	Parameters map[string]interface{} `json:"parameters"`
	Timeout    time.Duration     `json:"timeout"`
}

// SearchIndex interface defines operations for text search
type SearchIndex interface {
	// Index indexes a document
	Index(ctx context.Context, doc IndexDocument) error
	
	// Search performs text search with options
	Search(ctx context.Context, query string, options SearchIndexOptions) ([]SearchResult, error)
	
	// Delete removes a document from the index
	Delete(ctx context.Context, id string) error
	
	// BatchIndex indexes multiple documents in a single operation
	BatchIndex(ctx context.Context, docs []IndexDocument) error
	
	// Update updates an existing document
	Update(ctx context.Context, id string, doc IndexDocument) error
	
	// GetDocument retrieves a document by ID
	GetDocument(ctx context.Context, id string) (*IndexDocument, error)
	
	// DocumentExists checks if a document exists
	DocumentExists(ctx context.Context, id string) (bool, error)
	
	// DocumentCount returns the total number of documents indexed
	DocumentCount(ctx context.Context) (int64, error)
	
	// IndexSize returns the index size
	IndexSize(ctx context.Context) (int64, error)
	
	// Suggest provides search suggestions
	Suggest(ctx context.Context, query string, field string, size int) ([]string, error)
	
	// MultiSearch performs multiple searches
	MultiSearch(ctx context.Context, queries []string, options SearchIndexOptions) ([][]SearchResult, error)
	
	// CreateIndex creates a new index
	CreateIndex(ctx context.Context, indexName string, mapping map[string]interface{}) error
	
	// DeleteIndex deletes an index
	DeleteIndex(ctx context.Context, indexName string) error
	
	// RefreshIndex refreshes the index
	RefreshIndex(ctx context.Context) error
	
	// Close closes the search index connection
	Close() error
	
	// Health checks if the search index is healthy
	Health(ctx context.Context) error
}

// VectorStoreItem represents an item to be stored in the vector store
type VectorStoreItem struct {
	ID        string                 `json:"id"`
	Embedding []float32              `json:"embedding"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// SearchIndexItem represents an item to be indexed in the search index
type SearchIndexItem struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

