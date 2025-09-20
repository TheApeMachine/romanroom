package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
)

// MockVectorStore provides an in-memory implementation of VectorStore for testing
type MockVectorStore struct {
	mu            sync.RWMutex
	vectors       map[string]VectorStoreItem
	metadata      map[string]map[string]interface{}
	closed        bool
	healthy       bool
	shouldFail    bool
	searchResults []VectorResult
	stored        map[string]VectorStoreItem
}

// NewMockVectorStore creates a new mock vector store
func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		vectors:       make(map[string]VectorStoreItem),
		metadata:      make(map[string]map[string]interface{}),
		healthy:       true,
		shouldFail:    false,
		searchResults: nil,
		stored:        make(map[string]VectorStoreItem),
	}
}

// Store saves a vector embedding with associated metadata
func (m *MockVectorStore) Store(ctx context.Context, id string, embedding []float32, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.shouldFail {
		return fmt.Errorf("mock store failure")
	}
	
	if m.closed {
		return fmt.Errorf("vector store is closed")
	}
	
	if !m.healthy {
		return fmt.Errorf("vector store is unhealthy")
	}
	
	item := VectorStoreItem{
		ID:        id,
		Embedding: embedding,
		Metadata:  metadata,
	}
	
	m.vectors[id] = item
	m.metadata[id] = metadata
	m.stored[id] = item
	
	return nil
}

// Search performs similarity search and returns top k results
func (m *MockVectorStore) Search(ctx context.Context, query []float32, k int, filters map[string]interface{}) ([]VectorResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("vector store is closed")
	}
	
	// Return predefined results if set for testing
	if m.searchResults != nil {
		return m.searchResults, nil
	}
	
	var results []VectorResult
	
	for id, vector := range m.vectors {
		// Apply filters
		if !m.matchesFilters(vector.Metadata, filters) {
			continue
		}
		
		// Calculate cosine similarity
		score := m.cosineSimilarity(query, vector.Embedding)
		
		result := VectorResult{
			ID:        id,
			Score:     score,
			Embedding: vector.Embedding,
			Metadata:  vector.Metadata,
		}
		
		if content, ok := vector.Metadata["content"].(string); ok {
			result.Content = content
		}
		
		results = append(results, result)
	}
	
	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	// Limit results
	if k > 0 && len(results) > k {
		results = results[:k]
	}
	
	return results, nil
}

// Delete removes a vector by ID
func (m *MockVectorStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("vector store is closed")
	}
	
	delete(m.vectors, id)
	delete(m.metadata, id)
	
	return nil
}

// Update modifies existing vector metadata
func (m *MockVectorStore) Update(ctx context.Context, id string, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("vector store is closed")
	}
	
	if vector, exists := m.vectors[id]; exists {
		vector.Metadata = metadata
		m.vectors[id] = vector
		m.metadata[id] = metadata
		return nil
	}
	
	return fmt.Errorf("vector with ID %s not found", id)
}

// BatchStore stores multiple vectors in a single operation
func (m *MockVectorStore) BatchStore(ctx context.Context, vectors []VectorStoreItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("vector store is closed")
	}
	
	for _, vector := range vectors {
		m.vectors[vector.ID] = vector
		m.metadata[vector.ID] = vector.Metadata
	}
	
	return nil
}

// GetByID retrieves a specific vector by ID
func (m *MockVectorStore) GetByID(ctx context.Context, id string) (*VectorResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("vector store is closed")
	}
	
	if vector, exists := m.vectors[id]; exists {
		result := &VectorResult{
			ID:        vector.ID,
			Score:     1.0, // Perfect match
			Embedding: vector.Embedding,
			Metadata:  vector.Metadata,
		}
		
		if content, ok := vector.Metadata["content"].(string); ok {
			result.Content = content
		}
		
		return result, nil
	}
	
	return nil, fmt.Errorf("vector with ID %s not found", id)
}

// Count returns the total number of vectors stored
func (m *MockVectorStore) Count(ctx context.Context) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return 0, fmt.Errorf("vector store is closed")
	}
	
	if !m.healthy {
		return 0, fmt.Errorf("vector store is unhealthy")
	}
	
	return int64(len(m.vectors)), nil
}

// Close closes the vector store connection
func (m *MockVectorStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("vector store is already closed")
	}
	
	m.closed = true
	return nil
}

// Health checks if the vector store is healthy
func (m *MockVectorStore) Health(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return fmt.Errorf("vector store is closed")
	}
	
	if !m.healthy {
		return fmt.Errorf("vector store is unhealthy")
	}
	
	return nil
}

// SetHealthy sets the health status for testing
func (m *MockVectorStore) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
}

// SetShouldFail makes the mock fail operations for testing
func (m *MockVectorStore) SetShouldFail(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
}

// SetSearchResults sets predefined search results for testing
func (m *MockVectorStore) SetSearchResults(results []VectorResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.searchResults = results
}

// GetStored returns stored items for testing
func (m *MockVectorStore) GetStored() map[string]VectorStoreItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stored
}

// cosineSimilarity calculates cosine similarity between two vectors
func (m *MockVectorStore) cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}
	
	var dotProduct, normA, normB float64
	
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	
	if normA == 0 || normB == 0 {
		return 0.0
	}
	
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// matchesFilters checks if metadata matches the given filters
func (m *MockVectorStore) matchesFilters(metadata map[string]interface{}, filters map[string]interface{}) bool {
	if filters == nil {
		return true
	}
	
	for key, expectedValue := range filters {
		if actualValue, exists := metadata[key]; !exists || actualValue != expectedValue {
			return false
		}
	}
	
	return true
}