package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ContentProcessor orchestrates the content processing pipeline
type ContentProcessor struct {
	textChunker     *TextChunker
	entityExtractor *EntityExtractor
	claimExtractor  *ClaimExtractor
	config          *ContentProcessingConfig
}

// ContentProcessingConfig holds configuration for content processing pipeline
type ContentProcessingConfig struct {
	MaxChunkSize       int     `json:"max_chunk_size"`
	ChunkOverlap       int     `json:"chunk_overlap"`
	ChunkStrategy      string  `json:"chunk_strategy"`
	MinEntityConfidence float64 `json:"min_entity_confidence"`
	MinClaimConfidence  float64 `json:"min_claim_confidence"`
	EnablePreprocessing bool    `json:"enable_preprocessing"`
}

// ProcessingResult represents the result of content processing
type ProcessingResult struct {
	Chunks   []*Chunk  `json:"chunks"`
	Entities []*Entity `json:"entities"`
	Claims   []*Claim  `json:"claims"`
	Stats    ProcessingStats `json:"stats"`
}

// ProcessingStats provides statistics about the processing operation
type ProcessingStats struct {
	OriginalLength   int           `json:"original_length"`
	ChunkCount       int           `json:"chunk_count"`
	EntityCount      int           `json:"entity_count"`
	ClaimCount       int           `json:"claim_count"`
	ProcessingTime   time.Duration `json:"processing_time"`
	ChunkingTime     time.Duration `json:"chunking_time"`
	EntityExtractionTime time.Duration `json:"entity_extraction_time"`
	ClaimExtractionTime  time.Duration `json:"claim_extraction_time"`
}

// NewContentProcessor creates a new ContentProcessor with default configuration
func NewContentProcessor() *ContentProcessor {
	config := &ContentProcessingConfig{
		MaxChunkSize:        1000,
		ChunkOverlap:        100,
		ChunkStrategy:       "sentence",
		MinEntityConfidence: 0.5,
		MinClaimConfidence:  0.6,
		EnablePreprocessing: true,
	}
	
	return &ContentProcessor{
		textChunker:     NewTextChunker(config.MaxChunkSize, config.ChunkOverlap),
		entityExtractor: NewEntityExtractor(),
		claimExtractor:  NewClaimExtractor(),
		config:          config,
	}
}

// NewContentProcessorWithConfig creates a new ContentProcessor with custom configuration
func NewContentProcessorWithConfig(config *ContentProcessingConfig) *ContentProcessor {
	processor := &ContentProcessor{
		textChunker:     NewTextChunker(config.MaxChunkSize, config.ChunkOverlap),
		entityExtractor: NewEntityExtractor(),
		claimExtractor:  NewClaimExtractor(),
		config:          config,
	}
	
	// Configure extractors based on config
	processor.entityExtractor.SetMinConfidence(config.MinEntityConfidence)
	processor.claimExtractor.SetMinConfidence(config.MinClaimConfidence)
	
	return processor
}

// Process processes the given content through the complete pipeline
func (cp *ContentProcessor) Process(content, source string) (*ProcessingResult, error) {
	startTime := time.Now()
	
	if content == "" {
		return &ProcessingResult{
			Chunks:   []*Chunk{},
			Entities: []*Entity{},
			Claims:   []*Claim{},
			Stats: ProcessingStats{
				OriginalLength: 0,
				ProcessingTime: time.Since(startTime),
			},
		}, nil
	}
	
	// Preprocess content if enabled
	processedContent := content
	if cp.config.EnablePreprocessing {
		processedContent = cp.Preprocess(content)
	}
	
	// Step 1: Chunk the content
	chunkStart := time.Now()
	chunkResults := cp.Chunk(processedContent)
	chunkingTime := time.Since(chunkStart)
	
	// Step 2: Create chunks and extract entities/claims
	var chunks []*Chunk
	var allEntities []*Entity
	var allClaims []*Claim
	
	entityStart := time.Now()
	claimStart := time.Now()
	
	for i, chunkResult := range chunkResults {
		// Create chunk
		chunkID := fmt.Sprintf("%s_chunk_%d", source, i)
		chunk := NewChunk(chunkID, chunkResult.Text, source)
		
		// Extract entities from chunk
		entities, err := cp.entityExtractor.Extract(chunkResult.Text, source)
		if err != nil {
			return nil, fmt.Errorf("entity extraction failed: %v", err)
		}
		
		// Extract claims from chunk
		claims, err := cp.claimExtractor.ExtractClaims(chunkResult.Text, source)
		if err != nil {
			return nil, fmt.Errorf("claim extraction failed: %v", err)
		}
		
		// Add entities and claims to chunk
		for _, entity := range entities {
			chunk.AddEntity(*entity)
		}
		for _, claim := range claims {
			chunk.AddClaim(*claim)
		}
		
		// Set chunk metadata
		chunk.SetMetadata("chunk_strategy", chunkResult.Strategy)
		chunk.SetMetadata("chunk_index", i)
		chunk.SetMetadata("original_start", chunkResult.Start)
		chunk.SetMetadata("original_end", chunkResult.End)
		chunk.SetMetadata("entity_count", len(entities))
		chunk.SetMetadata("claim_count", len(claims))
		
		chunks = append(chunks, chunk)
		allEntities = append(allEntities, entities...)
		allClaims = append(allClaims, claims...)
	}
	
	entityExtractionTime := time.Since(entityStart)
	claimExtractionTime := time.Since(claimStart)
	
	// Create processing stats
	stats := ProcessingStats{
		OriginalLength:       len(content),
		ChunkCount:           len(chunks),
		EntityCount:          len(allEntities),
		ClaimCount:           len(allClaims),
		ProcessingTime:       time.Since(startTime),
		ChunkingTime:         chunkingTime,
		EntityExtractionTime: entityExtractionTime,
		ClaimExtractionTime:  claimExtractionTime,
	}
	
	return &ProcessingResult{
		Chunks:   chunks,
		Entities: allEntities,
		Claims:   allClaims,
		Stats:    stats,
	}, nil
}

// Chunk splits content into chunks using the configured strategy
func (cp *ContentProcessor) Chunk(content string) []ChunkResult {
	if content == "" {
		return []ChunkResult{}
	}
	
	switch cp.config.ChunkStrategy {
	case "size":
		return cp.textChunker.ChunkBySize(content)
	case "sentence":
		return cp.textChunker.ChunkBySentence(content)
	case "paragraph":
		return cp.textChunker.ChunkByParagraph(content)
	default:
		// Default to sentence-based chunking
		return cp.textChunker.ChunkBySentence(content)
	}
}

// Preprocess cleans and normalizes content before processing
func (cp *ContentProcessor) Preprocess(content string) string {
	if content == "" {
		return content
	}
	
	// Step 1: Normalize whitespace
	content = cp.normalizeWhitespace(content)
	
	// Step 2: Fix common encoding issues
	content = cp.fixEncodingIssues(content)
	
	// Step 3: Remove or normalize special characters
	content = cp.normalizeSpecialCharacters(content)
	
	// Step 4: Fix common punctuation issues
	content = cp.fixPunctuation(content)
	
	return content
}

// SetChunkStrategy sets the chunking strategy
func (cp *ContentProcessor) SetChunkStrategy(strategy string) {
	cp.config.ChunkStrategy = strategy
}

// SetMaxChunkSize sets the maximum chunk size
func (cp *ContentProcessor) SetMaxChunkSize(size int) {
	cp.config.MaxChunkSize = size
	cp.textChunker = NewTextChunker(size, cp.config.ChunkOverlap)
}

// SetChunkOverlap sets the chunk overlap size
func (cp *ContentProcessor) SetChunkOverlap(overlap int) {
	cp.config.ChunkOverlap = overlap
	cp.textChunker = NewTextChunker(cp.config.MaxChunkSize, overlap)
}

// SetMinEntityConfidence sets the minimum confidence for entity extraction
func (cp *ContentProcessor) SetMinEntityConfidence(confidence float64) {
	cp.config.MinEntityConfidence = confidence
	cp.entityExtractor.SetMinConfidence(confidence)
}

// SetMinClaimConfidence sets the minimum confidence for claim extraction
func (cp *ContentProcessor) SetMinClaimConfidence(confidence float64) {
	cp.config.MinClaimConfidence = confidence
	cp.claimExtractor.SetMinConfidence(confidence)
}

// EnablePreprocessing enables or disables content preprocessing
func (cp *ContentProcessor) EnablePreprocessing(enable bool) {
	cp.config.EnablePreprocessing = enable
}

// GetConfig returns the current processing configuration
func (cp *ContentProcessor) GetConfig() *ContentProcessingConfig {
	return cp.config
}

// normalizeWhitespace normalizes whitespace in the content
func (cp *ContentProcessor) normalizeWhitespace(content string) string {
	// Replace tabs with spaces first
	content = strings.ReplaceAll(content, "\t", " ")
	
	// Replace multiple spaces with single space (loop until no more changes)
	for strings.Contains(content, "  ") {
		content = strings.ReplaceAll(content, "  ", " ")
	}
	
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	
	// Remove excessive newlines (more than 2 consecutive)
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}
	
	return strings.TrimSpace(content)
}

// fixEncodingIssues fixes common encoding problems
func (cp *ContentProcessor) fixEncodingIssues(content string) string {
	// Fix common UTF-8 encoding issues based on actual byte sequences observed
	// â€™ = [195 162 226 130 172 226 132 162] -> '
	content = strings.ReplaceAll(content, string([]byte{195, 162, 226, 130, 172, 226, 132, 162}), "'")
	// â€œ = [195 162 226 130 172 197 147] -> "
	content = strings.ReplaceAll(content, string([]byte{195, 162, 226, 130, 172, 197, 147}), "\"")
	// â€ = [195 162 226 130 172] -> "
	content = strings.ReplaceAll(content, string([]byte{195, 162, 226, 130, 172}), "\"")
	
	// Also handle the standard UTF-8 sequences
	content = strings.ReplaceAll(content, "\xe2\x80\x99", "'")   // Right single quotation mark
	content = strings.ReplaceAll(content, "\xe2\x80\x9c", "\"")  // Left double quotation mark
	content = strings.ReplaceAll(content, "\xe2\x80\x9d", "\"")  // Right double quotation mark
	content = strings.ReplaceAll(content, "\xe2\x80\x94", "-")   // Em dash
	content = strings.ReplaceAll(content, "\xe2\x80\x93", "-")   // En dash
	content = strings.ReplaceAll(content, "\xe2\x80\xa2", "•")   // Bullet point
	content = strings.ReplaceAll(content, "\xe2\x80\xa6", "...") // Ellipsis
	
	return content
}

// normalizeSpecialCharacters normalizes special characters
func (cp *ContentProcessor) normalizeSpecialCharacters(content string) string {
	// Normalize different types of quotes
	content = strings.ReplaceAll(content, "\u201c", "\"") // Left double quotation mark
	content = strings.ReplaceAll(content, "\u201d", "\"") // Right double quotation mark
	content = strings.ReplaceAll(content, "\u2018", "'")  // Left single quotation mark
	content = strings.ReplaceAll(content, "\u2019", "'")  // Right single quotation mark
	
	// Normalize different types of dashes
	content = strings.ReplaceAll(content, "\u2014", " - ") // Em dash
	content = strings.ReplaceAll(content, "\u2013", " - ") // En dash
	
	// Normalize ellipsis
	content = strings.ReplaceAll(content, "\u2026", "...") // Horizontal ellipsis
	
	return content
}

// fixPunctuation fixes common punctuation issues
func (cp *ContentProcessor) fixPunctuation(content string) string {
	// Add space after periods if missing (but not for decimals)
	// Use regex to handle period followed by capital letter
	periodRegex := regexp.MustCompile(`\.([A-Z])`)
	content = periodRegex.ReplaceAllString(content, ". $1")
	
	// Fix spacing around commas
	content = strings.ReplaceAll(content, " ,", ",")
	content = strings.ReplaceAll(content, ",", ", ")
	content = strings.ReplaceAll(content, ",  ", ", ")
	
	// Fix spacing around colons and semicolons
	content = strings.ReplaceAll(content, " :", ":")
	content = strings.ReplaceAll(content, ":", ": ")
	content = strings.ReplaceAll(content, ":  ", ": ")
	
	content = strings.ReplaceAll(content, " ;", ";")
	content = strings.ReplaceAll(content, ";", "; ")
	content = strings.ReplaceAll(content, ";  ", "; ")
	
	// Fix spacing around parentheses
	content = strings.ReplaceAll(content, " (", " (")
	content = strings.ReplaceAll(content, "( ", "(")
	content = strings.ReplaceAll(content, " )", ")")
	content = strings.ReplaceAll(content, ") ", ") ")
	
	return content
}