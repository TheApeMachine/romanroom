package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// FileVectorStore implements VectorStore interface with JSON file persistence
type FileVectorStore struct {
	mu       sync.RWMutex
	filePath string
	vectors  map[string]VectorStoreRecord
	closed   bool
}

// VectorStoreRecord represents a stored vector record
type VectorStoreRecord struct {
	ID        string                 `json:"id"`
	Embedding []float32              `json:"embedding"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// NewFileVectorStore creates a new file-based vector store
func NewFileVectorStore(filePath string) *FileVectorStore {
	return &FileVectorStore{
		filePath: filePath,
		vectors:  make(map[string]VectorStoreRecord),
	}
}

// Store saves a vector embedding with associated metadata
func (f *FileVectorStore) Store(ctx context.Context, id string, embedding []float32, metadata map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return fmt.Errorf("vector store is closed")
	}
	
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	
	f.vectors[id] = VectorStoreRecord{
		ID:        id,
		Embedding: embedding,
		Metadata:  metadata,
	}
	
	return f.save()
}

// Search performs similarity search and returns top k results
func (f *FileVectorStore) Search(ctx context.Context, query []float32, k int, filters map[string]interface{}) ([]VectorResult, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return nil, fmt.Errorf("vector store is closed")
	}
	
	var results []VectorResult
	
	for _, record := range f.vectors {
		// Apply filters
		if !f.matchesFilters(record.Metadata, filters) {
			continue
		}
		
		// Calculate cosine similarity
		score := f.cosineSimilarity(query, record.Embedding)
		
		result := VectorResult{
			ID:        record.ID,
			Score:     score,
			Embedding: record.Embedding,
			Metadata:  record.Metadata,
		}
		
		results = append(results, result)
	}
	
	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	// Return top k results
	if k > 0 && len(results) > k {
		results = results[:k]
	}
	
	return results, nil
}

// Delete removes a vector by ID
func (f *FileVectorStore) Delete(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return fmt.Errorf("vector store is closed")
	}
	
	if _, exists := f.vectors[id]; !exists {
		return fmt.Errorf("vector with ID %s not found", id)
	}
	
	delete(f.vectors, id)
	return f.save()
}

// BatchStore stores multiple vectors in a single operation
func (f *FileVectorStore) BatchStore(ctx context.Context, items []VectorStoreItem) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return fmt.Errorf("vector store is closed")
	}
	
	for _, item := range items {
		if item.Metadata == nil {
			item.Metadata = make(map[string]interface{})
		}
		
		f.vectors[item.ID] = VectorStoreRecord{
			ID:        item.ID,
			Embedding: item.Embedding,
			Metadata:  item.Metadata,
		}
	}
	
	return f.save()
}

// Update updates the metadata for an existing vector
func (f *FileVectorStore) Update(ctx context.Context, id string, metadata map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return fmt.Errorf("vector store is closed")
	}
	
	record, exists := f.vectors[id]
	if !exists {
		return fmt.Errorf("vector with ID %s not found", id)
	}
	
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	
	record.Metadata = metadata
	f.vectors[id] = record
	
	return f.save()
}

// GetByID retrieves a vector by ID
func (f *FileVectorStore) GetByID(ctx context.Context, id string) (*VectorResult, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return nil, fmt.Errorf("vector store is closed")
	}
	
	record, exists := f.vectors[id]
	if !exists {
		return nil, fmt.Errorf("vector with ID %s not found", id)
	}
	
	result := &VectorResult{
		ID:        record.ID,
		Score:     1.0, // Perfect match
		Embedding: record.Embedding,
		Metadata:  record.Metadata,
	}
	
	return result, nil
}

// Count returns the total number of vectors stored
func (f *FileVectorStore) Count(ctx context.Context) (int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return 0, fmt.Errorf("vector store is closed")
	}
	
	return int64(len(f.vectors)), nil
}

// Close closes the vector store connection
func (f *FileVectorStore) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.closed = true
	return nil
}

// Health checks if the vector store is healthy
func (f *FileVectorStore) Health(ctx context.Context) error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return fmt.Errorf("vector store is closed")
	}
	
	// Check if file is accessible
	if _, err := os.Stat(f.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("file access error: %w", err)
	}
	
	return nil
}

// Load loads vectors from the JSON file
func (f *FileVectorStore) Load() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if _, err := os.Stat(f.filePath); os.IsNotExist(err) {
		// File doesn't exist, start with empty store
		f.vectors = make(map[string]VectorStoreRecord)
		return nil
	}
	
	data, err := os.ReadFile(f.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	if len(data) == 0 {
		f.vectors = make(map[string]VectorStoreRecord)
		return nil
	}
	
	if err := json.Unmarshal(data, &f.vectors); err != nil {
		return fmt.Errorf("failed to unmarshal vectors: %w", err)
	}
	
	return nil
}

// Save saves vectors to the JSON file
func (f *FileVectorStore) Save() error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	return f.save()
}

// save is the internal save method (assumes lock is held)
func (f *FileVectorStore) save() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(f.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	data, err := json.MarshalIndent(f.vectors, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vectors: %w", err)
	}
	
	if err := os.WriteFile(f.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// matchesFilters checks if metadata matches the given filters
func (f *FileVectorStore) matchesFilters(metadata map[string]interface{}, filters map[string]interface{}) bool {
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

// cosineSimilarity calculates cosine similarity between two vectors
func (f *FileVectorStore) cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}
	
	var dotProduct, normA, normB float64
	
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	
	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}
	
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}