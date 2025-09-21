package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// MultiViewStorage coordinates operations across vector, graph, and search storage backends
type MultiViewStorage struct {
	vectorStore VectorStore
	graphStore  GraphStore
	searchIndex SearchIndex
	config      *MultiViewStorageConfig
	mu          sync.RWMutex
}

// MultiViewStorageConfig holds configuration for the multi-view storage system
type MultiViewStorageConfig struct {
	VectorStore VectorStoreConfig `json:"vector_store"`
	GraphStore  GraphStoreConfig  `json:"graph_store"`
	SearchIndex SearchIndexConfig `json:"search_index"`
	Timeout     time.Duration     `json:"timeout"`
	RetryCount  int               `json:"retry_count"`
}

// StorageStats provides statistics about the multi-view storage system
type StorageStats struct {
	VectorCount   int64           `json:"vector_count"`
	NodeCount     int64           `json:"node_count"`
	EdgeCount     int64           `json:"edge_count"`
	DocumentCount int64           `json:"document_count"`
	StorageHealth map[string]bool `json:"storage_health"`
	LastUpdated   time.Time       `json:"last_updated"`
}

// NewMultiViewStorage creates a new multi-view storage coordinator
func NewMultiViewStorage(vectorStore VectorStore, graphStore GraphStore, searchIndex SearchIndex, config *MultiViewStorageConfig) *MultiViewStorage {
	return &MultiViewStorage{
		vectorStore: vectorStore,
		graphStore:  graphStore,
		searchIndex: searchIndex,
		config:      config,
	}
}

// StoreChunk stores a chunk across all storage backends
func (mvs *MultiViewStorage) StoreChunk(ctx context.Context, chunk *Chunk) error {
	mvs.mu.Lock()
	defer mvs.mu.Unlock()

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, mvs.config.Timeout)
	defer cancel()

	var errors []error

	// Store in vector database
	if err := mvs.vectorStore.Store(timeoutCtx, chunk.ID, chunk.Embedding, chunk.Metadata); err != nil {
		errors = append(errors, fmt.Errorf("vector store error: %w", err))
	}

	// Store in search index
	searchDoc := IndexDocument{
		ID:       chunk.ID,
		Content:  chunk.Content,
		Metadata: chunk.Metadata,
	}
	if err := mvs.searchIndex.Index(timeoutCtx, searchDoc); err != nil {
		errors = append(errors, fmt.Errorf("search index error: %w", err))
	}

	// Store entities and claims in graph
	for _, entity := range chunk.Entities {
		node := &Node{
			ID:   entity.ID,
			Type: EntityNode,
			Properties: map[string]interface{}{
				"name":       entity.Name,
				"type":       entity.Type,
				"confidence": entity.Confidence,
				"chunk_id":   chunk.ID,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := mvs.graphStore.CreateNode(timeoutCtx, node); err != nil {
			// Node might already exist, try to update
			if updateErr := mvs.graphStore.UpdateNode(timeoutCtx, node); updateErr != nil {
				// If both create and update fail, it's likely a duplicate. Log as a warning.
				log.Printf("warning: failed to create or update node %s: %v", node.ID, updateErr)
			}
		}
	}

	// Store claims in graph
	for _, claim := range chunk.Claims {
		node := &Node{
			ID:   claim.ID,
			Type: ClaimNode,
			Properties: map[string]interface{}{
				"subject":    claim.Subject,
				"predicate":  claim.Predicate,
				"object":     claim.Object,
				"confidence": claim.Confidence,
				"chunk_id":   chunk.ID,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := mvs.graphStore.CreateNode(timeoutCtx, node); err != nil {
			if updateErr := mvs.graphStore.UpdateNode(timeoutCtx, node); updateErr != nil {
				// If both create and update fail, it's likely a duplicate. Log as a warning.
				log.Printf("warning: failed to create or update claim node %s: %v", node.ID, updateErr)
			}
		}
	}

	// If we have errors but some operations succeeded, log them but don't fail
	if len(errors) > 0 {
		return fmt.Errorf("partial storage failure: %v", errors)
	}

	return nil
}

// RetrieveMultiView performs retrieval across all storage backends and fuses results
func (mvs *MultiViewStorage) RetrieveMultiView(ctx context.Context, query string, embedding []float32, options RetrievalOptions) (*MultiViewResults, error) {
	mvs.mu.RLock()
	defer mvs.mu.RUnlock()

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, mvs.config.Timeout)
	defer cancel()

	results := &MultiViewResults{
		Query:     query,
		Timestamp: time.Now(),
	}

	// Perform parallel retrieval
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Vector search
	wg.Add(1)
	go func() {
		defer wg.Done()
		vectorResults, err := mvs.vectorStore.Search(timeoutCtx, embedding, options.MaxResults, options.Filters)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			results.Errors = append(results.Errors, fmt.Errorf("vector search error: %w", err))
		} else {
			results.VectorResults = vectorResults
		}
	}()

	// Text search
	wg.Add(1)
	go func() {
		defer wg.Done()
		searchOptions := SearchIndexOptions{
			Limit:   options.MaxResults,
			Filters: options.Filters,
		}
		searchResults, err := mvs.searchIndex.Search(timeoutCtx, query, searchOptions)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			results.Errors = append(results.Errors, fmt.Errorf("text search error: %w", err))
		} else {
			results.SearchResults = searchResults
		}
	}()

	// Graph search (if we have entity information)
	if options.IncludeGraph {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Simple graph search - find nodes related to query terms
			graphResults, err := mvs.performGraphSearch(timeoutCtx, query, options)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results.Errors = append(results.Errors, fmt.Errorf("graph search error: %w", err))
			} else {
				results.GraphResults = graphResults
			}
		}()
	}

	wg.Wait()

	return results, nil
}

// DeleteChunk removes a chunk from all storage backends
func (mvs *MultiViewStorage) DeleteChunk(ctx context.Context, chunkID string) error {
	mvs.mu.Lock()
	defer mvs.mu.Unlock()

	timeoutCtx, cancel := context.WithTimeout(ctx, mvs.config.Timeout)
	defer cancel()

	var errors []error

	// Delete from vector store
	if err := mvs.vectorStore.Delete(timeoutCtx, chunkID); err != nil {
		errors = append(errors, fmt.Errorf("vector store delete error: %w", err))
	}

	// Delete from search index
	if err := mvs.searchIndex.Delete(timeoutCtx, chunkID); err != nil {
		errors = append(errors, fmt.Errorf("search index delete error: %w", err))
	}

	// Delete related nodes from graph (nodes with chunk_id property)
	nodes, err := mvs.graphStore.FindNodesByType(timeoutCtx, EntityNode, map[string]interface{}{
		"chunk_id": chunkID,
	})
	if err == nil {
		for _, node := range nodes {
			if err := mvs.graphStore.DeleteNode(timeoutCtx, node.ID); err != nil {
				errors = append(errors, fmt.Errorf("graph node delete error: %w", err))
			}
		}
	}

	claims, err := mvs.graphStore.FindNodesByType(timeoutCtx, ClaimNode, map[string]interface{}{
		"chunk_id": chunkID,
	})
	if err == nil {
		for _, claim := range claims {
			if err := mvs.graphStore.DeleteNode(timeoutCtx, claim.ID); err != nil {
				errors = append(errors, fmt.Errorf("graph claim delete error: %w", err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("partial delete failure: %v", errors)
	}

	return nil
}

// GetStats returns statistics about the storage system
func (mvs *MultiViewStorage) GetStats(ctx context.Context) (*StorageStats, error) {
	mvs.mu.RLock()
	defer mvs.mu.RUnlock()

	timeoutCtx, cancel := context.WithTimeout(ctx, mvs.config.Timeout)
	defer cancel()

	stats := &StorageStats{
		StorageHealth: make(map[string]bool),
		LastUpdated:   time.Now(),
	}

	// Get vector count
	if count, err := mvs.vectorStore.Count(timeoutCtx); err == nil {
		stats.VectorCount = count
		stats.StorageHealth["vector"] = true
	} else {
		stats.StorageHealth["vector"] = false
	}

	// Get node count
	if count, err := mvs.graphStore.NodeCount(timeoutCtx); err == nil {
		stats.NodeCount = count
		stats.StorageHealth["graph_nodes"] = true
	} else {
		stats.StorageHealth["graph_nodes"] = false
	}

	// Get edge count
	if count, err := mvs.graphStore.EdgeCount(timeoutCtx); err == nil {
		stats.EdgeCount = count
		stats.StorageHealth["graph_edges"] = true
	} else {
		stats.StorageHealth["graph_edges"] = false
	}

	// Get document count
	if count, err := mvs.searchIndex.DocumentCount(timeoutCtx); err == nil {
		stats.DocumentCount = count
		stats.StorageHealth["search"] = true
	} else {
		stats.StorageHealth["search"] = false
	}

	return stats, nil
}

// Health checks the health of all storage backends
func (mvs *MultiViewStorage) Health(ctx context.Context) error {
	mvs.mu.RLock()
	defer mvs.mu.RUnlock()

	timeoutCtx, cancel := context.WithTimeout(ctx, mvs.config.Timeout)
	defer cancel()

	var errors []error

	if err := mvs.vectorStore.Health(timeoutCtx); err != nil {
		errors = append(errors, fmt.Errorf("vector store unhealthy: %w", err))
	}

	if err := mvs.graphStore.Health(timeoutCtx); err != nil {
		errors = append(errors, fmt.Errorf("graph store unhealthy: %w", err))
	}

	if err := mvs.searchIndex.Health(timeoutCtx); err != nil {
		errors = append(errors, fmt.Errorf("search index unhealthy: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("storage health check failed: %v", errors)
	}

	return nil
}

// Close closes all storage backends
func (mvs *MultiViewStorage) Close() error {
	mvs.mu.Lock()
	defer mvs.mu.Unlock()

	var errors []error

	if err := mvs.vectorStore.Close(); err != nil {
		errors = append(errors, fmt.Errorf("vector store close error: %w", err))
	}

	if err := mvs.graphStore.Close(); err != nil {
		errors = append(errors, fmt.Errorf("graph store close error: %w", err))
	}

	if err := mvs.searchIndex.Close(); err != nil {
		errors = append(errors, fmt.Errorf("search index close error: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("storage close failed: %v", errors)
	}

	return nil
}

// performGraphSearch performs a simple graph-based search
func (mvs *MultiViewStorage) performGraphSearch(ctx context.Context, query string, _ RetrievalOptions) ([]GraphSearchResult, error) {
	// This is a simplified implementation
	// In a real system, this would involve more sophisticated graph traversal

	var results []GraphSearchResult

	// Find entities that might match the query
	entities, err := mvs.graphStore.FindNodesByType(ctx, EntityNode, nil)
	if err != nil {
		return nil, err
	}

	// Split query into terms for better matching
	queryTerms := strings.Fields(strings.ToLower(query))

	for _, entity := range entities {
		if _, ok := entity.Properties["name"].(string); ok {
			// Check if any query term matches the entity name
			for _, term := range queryTerms {
				if contains(entity.Properties["name"].(string), term) {
					result := GraphSearchResult{
						NodeID:     entity.ID,
						NodeType:   entity.Type,
						Properties: entity.Properties,
						Score:      0.8, // Simple scoring
					}
					results = append(results, result)
					break // Don't add the same entity multiple times
				}
			}
		}
	}

	return results, nil
}

// Helper function for case-insensitive string matching
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// Supporting types for multi-view results

// RetrievalOptions configures multi-view retrieval
type RetrievalOptions struct {
	MaxResults   int                    `json:"max_results"`
	IncludeGraph bool                   `json:"include_graph"`
	Filters      map[string]interface{} `json:"filters"`
	MinScore     float64                `json:"min_score"`
}

// MultiViewResults contains results from all storage backends
type MultiViewResults struct {
	Query         string              `json:"query"`
	VectorResults []VectorResult      `json:"vector_results"`
	SearchResults []SearchResult      `json:"search_results"`
	GraphResults  []GraphSearchResult `json:"graph_results"`
	Errors        []error             `json:"errors,omitempty"`
	Timestamp     time.Time           `json:"timestamp"`
}

// GraphSearchResult represents a result from graph search
type GraphSearchResult struct {
	NodeID     string                 `json:"node_id"`
	NodeType   NodeType               `json:"node_type"`
	Properties map[string]interface{} `json:"properties"`
	Score      float64                `json:"score"`
	Path       []string               `json:"path,omitempty"`
}
