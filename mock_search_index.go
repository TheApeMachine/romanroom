package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// MockSearchIndex provides an in-memory implementation of SearchIndex for testing
type MockSearchIndex struct {
	mu        sync.RWMutex
	documents map[string]IndexDocument
	index     map[string][]string // term -> document IDs
	closed    bool
	healthy   bool
	indexed   map[string]IndexDocument
}

// NewMockSearchIndex creates a new mock search index
func NewMockSearchIndex() *MockSearchIndex {
	return &MockSearchIndex{
		documents: make(map[string]IndexDocument),
		index:     make(map[string][]string),
		healthy:   true,
		indexed:   make(map[string]IndexDocument),
	}
}

// GetIndexed returns indexed documents for testing
func (m *MockSearchIndex) GetIndexed() map[string]IndexDocument {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.indexed
}

// Index adds a document to the search index
func (m *MockSearchIndex) Index(ctx context.Context, doc IndexDocument) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("search index is closed")
	}
	
	// Remove old document from index if it exists
	if oldDoc, exists := m.documents[doc.ID]; exists {
		m.removeFromIndex(doc.ID, oldDoc.Content)
	}
	
	// Store document
	m.documents[doc.ID] = doc
	
	// Add to inverted index
	m.addToIndex(doc.ID, doc.Content)
	
	// Also store in test helper
	m.indexed[doc.ID] = doc
	
	return nil
}

// BatchIndex adds multiple documents to the search index
func (m *MockSearchIndex) BatchIndex(ctx context.Context, docs []IndexDocument) error {
	for _, doc := range docs {
		if err := m.Index(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

// Update updates an existing document
func (m *MockSearchIndex) Update(ctx context.Context, id string, doc IndexDocument) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("search index is closed")
	}
	
	// Remove old document from index
	if oldDoc, exists := m.documents[id]; exists {
		m.removeFromIndex(id, oldDoc.Content)
	}
	
	// Update document ID to match parameter
	doc.ID = id
	m.documents[id] = doc
	
	// Add updated document to index
	m.addToIndex(id, doc.Content)
	
	return nil
}

// Delete removes a document from the index
func (m *MockSearchIndex) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("search index is closed")
	}
	
	if doc, exists := m.documents[id]; exists {
		m.removeFromIndex(id, doc.Content)
		delete(m.documents, id)
		return nil
	}
	
	return fmt.Errorf("document with ID %s not found", id)
}

// Search performs text search and returns matching documents
func (m *MockSearchIndex) Search(ctx context.Context, query string, options SearchIndexOptions) ([]SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("search index is closed")
	}
	
	if !m.healthy {
		return nil, fmt.Errorf("search index is unhealthy")
	}
	
	// Tokenize query
	queryTerms := m.tokenize(strings.ToLower(query))
	
	// Find matching documents
	docScores := make(map[string]float64)
	
	for _, term := range queryTerms {
		if docIDs, exists := m.index[term]; exists {
			for _, docID := range docIDs {
				docScores[docID] += 1.0 // Simple TF scoring
			}
		}
	}
	
	// Convert to results and apply filters
	var results []SearchResult
	for docID, score := range docScores {
		doc := m.documents[docID]
		
		// Apply filters
		if !m.matchesFilters(doc.Metadata, options.Filters) {
			continue
		}
		
		result := SearchResult{
			ID:       docID,
			Score:    score,
			Content:  doc.Content,
			Metadata: doc.Metadata,
		}
		
		// Add highlights if requested
		if options.Highlight {
			result.Highlights = m.generateHighlights(doc.Content, queryTerms)
		}
		
		results = append(results, result)
	}
	
	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	// Apply pagination
	start := options.Offset
	end := start + options.Limit
	
	if start >= len(results) {
		return []SearchResult{}, nil
	}
	
	if end > len(results) {
		end = len(results)
	}
	
	if options.Limit > 0 {
		results = results[start:end]
	}
	
	return results, nil
}

// MultiSearch performs multiple searches
func (m *MockSearchIndex) MultiSearch(ctx context.Context, queries []string, options SearchIndexOptions) ([][]SearchResult, error) {
	var results [][]SearchResult
	
	for _, query := range queries {
		queryResults, err := m.Search(ctx, query, options)
		if err != nil {
			return nil, err
		}
		results = append(results, queryResults)
	}
	
	return results, nil
}

// Suggest provides search suggestions
func (m *MockSearchIndex) Suggest(ctx context.Context, query string, field string, size int) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("search index is closed")
	}
	
	queryLower := strings.ToLower(query)
	var suggestions []string
	
	// Find terms that start with the query
	for term := range m.index {
		if strings.HasPrefix(term, queryLower) {
			suggestions = append(suggestions, term)
		}
	}
	
	// Sort and limit
	sort.Strings(suggestions)
	if size > 0 && len(suggestions) > size {
		suggestions = suggestions[:size]
	}
	
	return suggestions, nil
}

// GetDocument retrieves a document by ID
func (m *MockSearchIndex) GetDocument(ctx context.Context, id string) (*IndexDocument, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("search index is closed")
	}
	
	if doc, exists := m.documents[id]; exists {
		return &doc, nil
	}
	
	return nil, fmt.Errorf("document with ID %s not found", id)
}

// DocumentExists checks if a document exists
func (m *MockSearchIndex) DocumentExists(ctx context.Context, id string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return false, fmt.Errorf("search index is closed")
	}
	
	_, exists := m.documents[id]
	return exists, nil
}

// CreateIndex creates a new index (no-op for mock)
func (m *MockSearchIndex) CreateIndex(ctx context.Context, indexName string, mapping map[string]interface{}) error {
	return nil // No-op for mock implementation
}

// DeleteIndex deletes an index (no-op for mock)
func (m *MockSearchIndex) DeleteIndex(ctx context.Context, indexName string) error {
	return nil // No-op for mock implementation
}

// RefreshIndex refreshes the index (no-op for mock)
func (m *MockSearchIndex) RefreshIndex(ctx context.Context) error {
	return nil // No-op for mock implementation
}

// DocumentCount returns the number of documents
func (m *MockSearchIndex) DocumentCount(ctx context.Context) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return 0, fmt.Errorf("search index is closed")
	}
	
	if !m.healthy {
		return 0, fmt.Errorf("search index is unhealthy")
	}
	
	return int64(len(m.documents)), nil
}

// IndexSize returns the index size (approximation)
func (m *MockSearchIndex) IndexSize(ctx context.Context) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return 0, fmt.Errorf("search index is closed")
	}
	
	size := int64(0)
	for _, doc := range m.documents {
		size += int64(len(doc.Content))
	}
	
	return size, nil
}

// Close closes the search index
func (m *MockSearchIndex) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.closed = true
	return nil
}

// Health checks if the search index is healthy
func (m *MockSearchIndex) Health(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return fmt.Errorf("search index is closed")
	}
	
	if !m.healthy {
		return fmt.Errorf("search index is unhealthy")
	}
	
	return nil
}

// SetHealthy sets the health status for testing
func (m *MockSearchIndex) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
}

// Helper methods

// tokenize splits text into terms
func (m *MockSearchIndex) tokenize(text string) []string {
	// Simple tokenization - split on whitespace and punctuation
	text = strings.ToLower(text)
	words := strings.FieldsFunc(text, func(c rune) bool {
		return !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'))
	})
	
	// Remove empty strings
	var result []string
	for _, word := range words {
		if len(word) > 0 {
			result = append(result, word)
		}
	}
	
	return result
}

// addToIndex adds a document to the inverted index
func (m *MockSearchIndex) addToIndex(docID, content string) {
	terms := m.tokenize(content)
	
	for _, term := range terms {
		if _, exists := m.index[term]; !exists {
			m.index[term] = []string{}
		}
		
		// Add document ID if not already present
		found := false
		for _, id := range m.index[term] {
			if id == docID {
				found = true
				break
			}
		}
		
		if !found {
			m.index[term] = append(m.index[term], docID)
		}
	}
}

// removeFromIndex removes a document from the inverted index
func (m *MockSearchIndex) removeFromIndex(docID, content string) {
	terms := m.tokenize(content)
	
	for _, term := range terms {
		if docIDs, exists := m.index[term]; exists {
			// Remove document ID from the list
			for i, id := range docIDs {
				if id == docID {
					m.index[term] = append(docIDs[:i], docIDs[i+1:]...)
					break
				}
			}
			
			// Remove term if no documents left
			if len(m.index[term]) == 0 {
				delete(m.index, term)
			}
		}
	}
}

// matchesFilters checks if document metadata matches filters
func (m *MockSearchIndex) matchesFilters(metadata map[string]interface{}, filters map[string]interface{}) bool {
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

// generateHighlights creates highlighted snippets
func (m *MockSearchIndex) generateHighlights(content string, queryTerms []string) []string {
	var highlights []string
	contentLower := strings.ToLower(content)
	
	for _, term := range queryTerms {
		if strings.Contains(contentLower, term) {
			// Find the term and create a snippet around it
			index := strings.Index(contentLower, term)
			start := index - 20
			if start < 0 {
				start = 0
			}
			end := index + len(term) + 20
			if end > len(content) {
				end = len(content)
			}
			
			snippet := content[start:end]
			// Highlight the term (simple approach)
			highlighted := strings.ReplaceAll(snippet, term, fmt.Sprintf("<em>%s</em>", term))
			highlights = append(highlights, highlighted)
		}
	}
	
	return highlights
}