package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// FileSearchIndex implements SearchIndex interface with JSON file persistence
type FileSearchIndex struct {
	mu        sync.RWMutex
	filePath  string
	documents map[string]IndexDocument
	// Inverted index: term -> document IDs
	invertedIndex map[string][]string
	closed        bool
}

// SearchIndexData represents the JSON structure for persistence
type SearchIndexData struct {
	Documents     map[string]IndexDocument `json:"documents"`
	InvertedIndex map[string][]string      `json:"inverted_index"`
}

// NewFileSearchIndex creates a new file-based search index
func NewFileSearchIndex(filePath string) *FileSearchIndex {
	return &FileSearchIndex{
		filePath:      filePath,
		documents:     make(map[string]IndexDocument),
		invertedIndex: make(map[string][]string),
	}
}

// Index indexes a document
func (f *FileSearchIndex) Index(ctx context.Context, doc IndexDocument) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return fmt.Errorf("search index is closed")
	}
	
	// Remove old document from index if it exists
	if oldDoc, exists := f.documents[doc.ID]; exists {
		f.removeFromInvertedIndex(doc.ID, oldDoc.Content)
	}
	
	// Store document
	f.documents[doc.ID] = doc
	
	// Add to inverted index
	f.addToInvertedIndex(doc.ID, doc.Content)
	
	return f.save()
}

// Search performs text search with options
func (f *FileSearchIndex) Search(ctx context.Context, query string, options SearchIndexOptions) ([]SearchResult, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return nil, fmt.Errorf("search index is closed")
	}
	
	// Tokenize query
	queryTerms := f.tokenize(strings.ToLower(query))
	if len(queryTerms) == 0 {
		return []SearchResult{}, nil
	}
	
	// Find matching documents
	docScores := make(map[string]float64)
	
	for _, term := range queryTerms {
		if docIDs, exists := f.invertedIndex[term]; exists {
			for _, docID := range docIDs {
				docScores[docID] += 1.0 // Simple TF scoring
			}
		}
	}
	
	// Convert to results and apply filters
	var results []SearchResult
	for docID, score := range docScores {
		doc := f.documents[docID]
		
		// Apply filters
		if !f.matchesFilters(doc.Metadata, options.Filters) {
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
			result.Highlights = f.generateHighlights(doc.Content, queryTerms)
		}
		
		results = append(results, result)
	}
	
	// Sort results
	f.sortResults(results, options.SortBy, options.SortOrder)
	
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

// Delete removes a document from the index
func (f *FileSearchIndex) Delete(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return fmt.Errorf("search index is closed")
	}
	
	doc, exists := f.documents[id]
	if !exists {
		return fmt.Errorf("document with ID %s not found", id)
	}
	
	// Remove from inverted index
	f.removeFromInvertedIndex(id, doc.Content)
	
	// Remove document
	delete(f.documents, id)
	
	return f.save()
}

// BatchIndex indexes multiple documents in a single operation
func (f *FileSearchIndex) BatchIndex(ctx context.Context, docs []IndexDocument) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return fmt.Errorf("search index is closed")
	}
	
	for _, doc := range docs {
		// Remove old document from index if it exists
		if oldDoc, exists := f.documents[doc.ID]; exists {
			f.removeFromInvertedIndex(doc.ID, oldDoc.Content)
		}
		
		// Store document
		f.documents[doc.ID] = doc
		
		// Add to inverted index
		f.addToInvertedIndex(doc.ID, doc.Content)
	}
	
	return f.save()
}

// Update updates an existing document
func (f *FileSearchIndex) Update(ctx context.Context, id string, doc IndexDocument) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return fmt.Errorf("search index is closed")
	}
	
	// Remove old document from index if it exists
	if oldDoc, exists := f.documents[id]; exists {
		f.removeFromInvertedIndex(id, oldDoc.Content)
	}
	
	// Update document ID to match parameter
	doc.ID = id
	f.documents[id] = doc
	
	// Add updated document to index
	f.addToInvertedIndex(id, doc.Content)
	
	return f.save()
}

// GetDocument retrieves a document by ID
func (f *FileSearchIndex) GetDocument(ctx context.Context, id string) (*IndexDocument, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return nil, fmt.Errorf("search index is closed")
	}
	
	doc, exists := f.documents[id]
	if !exists {
		return nil, fmt.Errorf("document with ID %s not found", id)
	}
	
	return &doc, nil
}

// DocumentExists checks if a document exists
func (f *FileSearchIndex) DocumentExists(ctx context.Context, id string) (bool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return false, fmt.Errorf("search index is closed")
	}
	
	_, exists := f.documents[id]
	return exists, nil
}

// DocumentCount returns the total number of documents indexed
func (f *FileSearchIndex) DocumentCount(ctx context.Context) (int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return 0, fmt.Errorf("search index is closed")
	}
	
	return int64(len(f.documents)), nil
}

// IndexSize returns the index size
func (f *FileSearchIndex) IndexSize(ctx context.Context) (int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return 0, fmt.Errorf("search index is closed")
	}
	
	size := int64(0)
	for _, doc := range f.documents {
		size += int64(len(doc.Content))
	}
	
	return size, nil
}

// Suggest provides search suggestions
func (f *FileSearchIndex) Suggest(ctx context.Context, query string, field string, size int) ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return nil, fmt.Errorf("search index is closed")
	}
	
	queryLower := strings.ToLower(query)
	var suggestions []string
	
	// Find terms that start with the query
	for term := range f.invertedIndex {
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

// MultiSearch performs multiple searches
func (f *FileSearchIndex) MultiSearch(ctx context.Context, queries []string, options SearchIndexOptions) ([][]SearchResult, error) {
	var results [][]SearchResult
	
	for _, query := range queries {
		queryResults, err := f.Search(ctx, query, options)
		if err != nil {
			return nil, err
		}
		results = append(results, queryResults)
	}
	
	return results, nil
}

// CreateIndex creates a new index (no-op for file-based implementation)
func (f *FileSearchIndex) CreateIndex(ctx context.Context, indexName string, mapping map[string]interface{}) error {
	return nil // No-op for file-based implementation
}

// DeleteIndex deletes an index (no-op for file-based implementation)
func (f *FileSearchIndex) DeleteIndex(ctx context.Context, indexName string) error {
	return nil // No-op for file-based implementation
}

// RefreshIndex refreshes the index (no-op for file-based implementation)
func (f *FileSearchIndex) RefreshIndex(ctx context.Context) error {
	return nil // No-op for file-based implementation
}

// Close closes the search index connection
func (f *FileSearchIndex) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.closed = true
	return nil
}

// Health checks if the search index is healthy
func (f *FileSearchIndex) Health(ctx context.Context) error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return fmt.Errorf("search index is closed")
	}
	
	// Check if file is accessible
	if _, err := os.Stat(f.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("file access error: %w", err)
	}
	
	return nil
}

// Load loads the search index from the JSON file
func (f *FileSearchIndex) Load() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if _, err := os.Stat(f.filePath); os.IsNotExist(err) {
		// File doesn't exist, start with empty index
		f.documents = make(map[string]IndexDocument)
		f.invertedIndex = make(map[string][]string)
		return nil
	}
	
	data, err := os.ReadFile(f.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	if len(data) == 0 {
		f.documents = make(map[string]IndexDocument)
		f.invertedIndex = make(map[string][]string)
		return nil
	}
	
	var indexData SearchIndexData
	if err := json.Unmarshal(data, &indexData); err != nil {
		return fmt.Errorf("failed to unmarshal search index data: %w", err)
	}
	
	f.documents = indexData.Documents
	f.invertedIndex = indexData.InvertedIndex
	
	return nil
}

// Save saves the search index to the JSON file
func (f *FileSearchIndex) Save() error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	return f.save()
}

// save is the internal save method (assumes lock is held)
func (f *FileSearchIndex) save() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(f.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	indexData := SearchIndexData{
		Documents:     f.documents,
		InvertedIndex: f.invertedIndex,
	}
	
	data, err := json.MarshalIndent(indexData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal search index data: %w", err)
	}
	
	if err := os.WriteFile(f.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// Helper methods

// tokenize splits text into terms
func (f *FileSearchIndex) tokenize(text string) []string {
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

// addToInvertedIndex adds a document to the inverted index
func (f *FileSearchIndex) addToInvertedIndex(docID, content string) {
	terms := f.tokenize(content)
	
	for _, term := range terms {
		if _, exists := f.invertedIndex[term]; !exists {
			f.invertedIndex[term] = []string{}
		}
		
		// Add document ID if not already present
		found := false
		for _, id := range f.invertedIndex[term] {
			if id == docID {
				found = true
				break
			}
		}
		
		if !found {
			f.invertedIndex[term] = append(f.invertedIndex[term], docID)
		}
	}
}

// removeFromInvertedIndex removes a document from the inverted index
func (f *FileSearchIndex) removeFromInvertedIndex(docID, content string) {
	terms := f.tokenize(content)
	
	for _, term := range terms {
		if docIDs, exists := f.invertedIndex[term]; exists {
			// Remove document ID from the list
			for i, id := range docIDs {
				if id == docID {
					f.invertedIndex[term] = append(docIDs[:i], docIDs[i+1:]...)
					break
				}
			}
			
			// Remove term if no documents left
			if len(f.invertedIndex[term]) == 0 {
				delete(f.invertedIndex, term)
			}
		}
	}
}

// matchesFilters checks if document metadata matches filters
func (f *FileSearchIndex) matchesFilters(metadata map[string]interface{}, filters map[string]interface{}) bool {
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
func (f *FileSearchIndex) generateHighlights(content string, queryTerms []string) []string {
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

// sortResults sorts search results based on sort options
func (f *FileSearchIndex) sortResults(results []SearchResult, sortBy, sortOrder string) {
	if sortBy == "" {
		sortBy = "score"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}
	
	sort.Slice(results, func(i, j int) bool {
		var less bool
		
		switch sortBy {
		case "score":
			less = results[i].Score < results[j].Score
		case "id":
			less = results[i].ID < results[j].ID
		default:
			// Default to score
			less = results[i].Score < results[j].Score
		}
		
		if sortOrder == "asc" {
			return less
		}
		return !less
	})
}