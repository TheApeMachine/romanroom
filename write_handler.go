package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WriteHandler handles memory write operations
type WriteHandler struct {
	memoryWriter     *MemoryWriter
	contentProcessor *ContentProcessor
	validator        *WriteArgsValidator
	formatter        *WriteResponseFormatter
	config           *WriteHandlerConfig
}

// WriteHandlerConfig holds configuration for the write handler
type WriteHandlerConfig struct {
	MaxContentLength        int           `json:"max_content_length"`
	DefaultConfidence       float64       `json:"default_confidence"`
	RequireSource           bool          `json:"require_source"`
	EnableConflictDetection bool          `json:"enable_conflict_detection"`
	ProcessingTimeout       time.Duration `json:"processing_timeout"`
	EnableDeduplication     bool          `json:"enable_deduplication"`
}

// NewWriteHandler creates a new WriteHandler instance
func NewWriteHandler(memoryWriter *MemoryWriter, contentProcessor *ContentProcessor) *WriteHandler {
	config := &WriteHandlerConfig{
		MaxContentLength:        10000,
		DefaultConfidence:       1.0,
		RequireSource:           true,
		EnableConflictDetection: true,
		ProcessingTimeout:       30 * time.Second,
		EnableDeduplication:     true,
	}

	return &WriteHandler{
		memoryWriter:     memoryWriter,
		contentProcessor: contentProcessor,
		validator:        NewWriteArgsValidator(),
		formatter:        NewWriteResponseFormatter(),
		config:           config,
	}
}

// NewWriteHandlerWithConfig creates a WriteHandler with custom configuration
func NewWriteHandlerWithConfig(memoryWriter *MemoryWriter, contentProcessor *ContentProcessor, config *WriteHandlerConfig) *WriteHandler {
	if config == nil {
		return NewWriteHandler(memoryWriter, contentProcessor)
	}

	return &WriteHandler{
		memoryWriter:     memoryWriter,
		contentProcessor: contentProcessor,
		validator:        NewWriteArgsValidator(),
		formatter:        NewWriteResponseFormatter(),
		config:           config,
	}
}

// HandleWrite processes a memory write request
func (wh *WriteHandler) HandleWrite(ctx context.Context, req *mcp.CallToolRequest, args WriteArgs) (*mcp.CallToolResult, WriteResult, error) {
	startTime := time.Now()

	log.Printf("Handling write request: content length=%d, source=%s", len(args.Content), args.Source)

	// Validate arguments
	if err := wh.validator.Validate(args); err != nil {
		return nil, WriteResult{}, fmt.Errorf("argument validation failed: %w", err)
	}

	// Sanitize and prepare arguments
	sanitizedArgs, err := wh.validator.SanitizeInput(args)
	if err != nil {
		return nil, WriteResult{}, fmt.Errorf("input sanitization failed: %w", err)
	}

	// Set timeout context
	writeCtx, cancel := context.WithTimeout(ctx, wh.config.ProcessingTimeout)
	defer cancel()

	// Process the content
	processedContent, err := wh.processContent(writeCtx, sanitizedArgs.Content, sanitizedArgs.Source)
	if err != nil {
		return nil, WriteResult{}, fmt.Errorf("content processing failed: %w", err)
	}

	// Convert args to write metadata
	metadata := wh.convertArgsToMetadata(sanitizedArgs)

	// Write to memory
	writeResponse, err := wh.memoryWriter.Write(writeCtx, sanitizedArgs.Content, metadata)
	if err != nil {
		return nil, WriteResult{}, fmt.Errorf("memory write failed: %w", err)
	}

	// Detect conflicts if enabled
	var conflicts []ConflictInfo
	if wh.config.EnableConflictDetection {
		conflicts, err = wh.detectConflicts(writeCtx, processedContent, writeResponse)
		if err != nil {
			log.Printf("Conflict detection failed: %v", err)
			// Don't fail the write operation, just log the error
		}
	}

	// Create enhanced write response
	enhancedResponse := &WriteResponse{
		MemoryID:       writeResponse.MemoryID,
		CandidateCount: writeResponse.CandidateCount,
		ConflictsFound: conflicts,
		EntitiesLinked: writeResponse.EntitiesLinked,
		ProvenanceID:   writeResponse.ProvenanceID,
		ChunksCreated:  processedContent.Stats.ChunkCount,
		GraphUpdates: GraphUpdates{
			NodesCreated: processedContent.Stats.EntityCount + processedContent.Stats.ClaimCount,
			EdgesCreated: len(processedContent.Claims), // Simplified edge count
		},
		ProcessingTime: time.Since(startTime),
	}

	// Format the response
	result, err := wh.formatter.Format(enhancedResponse, sanitizedArgs)
	if err != nil {
		return nil, WriteResult{}, fmt.Errorf("response formatting failed: %w", err)
	}

	// Create MCP result
	mcpResult := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Stored memory with ID: %s, created %d chunks, linked %d entities",
					result.MemoryID, enhancedResponse.ChunksCreated, len(result.EntitiesLinked)),
			},
		},
	}

	log.Printf("Write completed in %v, memory ID: %s",
		time.Since(startTime), result.MemoryID)

	return mcpResult, result, nil
}

// processContent handles the core content processing logic
func (wh *WriteHandler) processContent(_ context.Context, content, source string) (*ProcessingResult, error) {
	// Process content through the content processor
	processedContent, err := wh.contentProcessor.Process(content, source)
	if err != nil {
		return nil, fmt.Errorf("content processing failed: %w", err)
	}

	// Validate processing results
	if len(processedContent.Chunks) == 0 {
		return nil, fmt.Errorf("no valid chunks created from content")
	}

	// Apply additional processing filters
	processedContent = wh.applyProcessingFilters(processedContent)

	return processedContent, nil
}

// detectConflicts detects potential conflicts with existing memories
func (wh *WriteHandler) detectConflicts(_ context.Context, processedContent *ProcessingResult, writeResponse *WriteResult) ([]ConflictInfo, error) {
	conflicts := make([]ConflictInfo, 0)

	// Simple conflict detection based on entity overlap
	// This would be enhanced with actual similarity checking in a full implementation
	for _, chunk := range processedContent.Chunks {
		for _, entity := range chunk.Entities {
			// Check if entity already exists with different claims
			// This is a placeholder - real implementation would query the graph store
			if wh.hasConflictingClaims(entity) {
				conflicts = append(conflicts, ConflictInfo{
					ID:             fmt.Sprintf("entity_conflict_%s", entity.ID),
					Type:           "entity_claim_conflict",
					Description:    fmt.Sprintf("Entity '%s' has conflicting claims in existing memory", entity.Name),
					ConflictingIDs: []string{entity.ID},
					Severity:       "medium",
				})
			}
		}
	}

	// Check for duplicate content
	if wh.hasDuplicateContent(processedContent) {
		conflicts = append(conflicts, ConflictInfo{
			ID:             fmt.Sprintf("duplicate_content_%s", writeResponse.MemoryID),
			Type:           "duplicate_content",
			Description:    "Similar content already exists in memory",
			ConflictingIDs: []string{writeResponse.MemoryID},
			Severity:       "low",
		})
	}

	return conflicts, nil
}

// hasConflictingClaims checks if an entity has conflicting claims (placeholder)
func (wh *WriteHandler) hasConflictingClaims(_ Entity) bool {
	// Placeholder implementation
	// Real implementation would query the graph store for existing claims about this entity
	return false
}

// hasDuplicateContent checks for duplicate content (placeholder)
func (wh *WriteHandler) hasDuplicateContent(_ *ProcessingResult) bool {
	// Placeholder implementation
	// Real implementation would use similarity search to find duplicate content
	return false
}

// applyProcessingFilters applies additional filters to processed content
func (wh *WriteHandler) applyProcessingFilters(processedContent *ProcessingResult) *ProcessingResult {
	// Filter out low-confidence entities and claims
	filtered := &ProcessingResult{
		Chunks:   make([]*Chunk, 0),
		Entities: make([]*Entity, 0),
		Claims:   make([]*Claim, 0),
		Stats:    processedContent.Stats,
	}

	for _, chunk := range processedContent.Chunks {
		// Filter entities by confidence
		var filteredEntities []Entity
		for _, entity := range chunk.Entities {
			if entity.Confidence >= 0.5 { // Minimum confidence threshold
				filteredEntities = append(filteredEntities, entity)
			}
		}
		chunk.Entities = filteredEntities

		// Filter claims by confidence
		var filteredClaims []Claim
		for _, claim := range chunk.Claims {
			if claim.Confidence >= 0.6 { // Minimum confidence threshold for claims
				filteredClaims = append(filteredClaims, claim)
			}
		}
		chunk.Claims = filteredClaims

		// Only keep chunks that have entities or claims
		if len(chunk.Entities) > 0 || len(chunk.Claims) > 0 || len(chunk.Content) > 0 {
			filtered.Chunks = append(filtered.Chunks, chunk)
		}
	}

	// Update stats
	filtered.Stats.ChunkCount = len(filtered.Chunks)

	entityCount := 0
	claimCount := 0
	for _, chunk := range filtered.Chunks {
		entityCount += len(chunk.Entities)
		claimCount += len(chunk.Claims)
	}
	filtered.Stats.EntityCount = entityCount
	filtered.Stats.ClaimCount = claimCount

	return filtered
}

// convertArgsToMetadata converts WriteArgs to WriteMetadata
func (wh *WriteHandler) convertArgsToMetadata(args WriteArgs) WriteMetadata {
	metadata := WriteMetadata{
		Source:          args.Source,
		Timestamp:       time.Now(),
		Tags:            args.Tags,
		RequireEvidence: args.RequireEvidence,
		Confidence:      wh.config.DefaultConfidence,
		Language:        "en",
		ContentType:     "text/plain",
		Version:         "1.0",
		Metadata:        args.Metadata,
	}

	// Override confidence if provided in metadata
	if confidenceVal, ok := args.Metadata["confidence"]; ok {
		if confidence, ok := confidenceVal.(float64); ok && confidence > 0 && confidence <= 1 {
			metadata.Confidence = confidence
		}
	}

	// Set content type from metadata if provided
	if contentType, ok := args.Metadata["content_type"].(string); ok {
		metadata.ContentType = contentType
	}

	// Set language from metadata if provided
	if language, ok := args.Metadata["language"].(string); ok {
		metadata.Language = language
	}

	// Set user ID from metadata if provided
	if userID, ok := args.Metadata["user_id"].(string); ok {
		metadata.UserID = userID
	}

	return metadata
}

// GetConfig returns the current configuration
func (wh *WriteHandler) GetConfig() *WriteHandlerConfig {
	return wh.config
}

// UpdateConfig updates the configuration
func (wh *WriteHandler) UpdateConfig(config *WriteHandlerConfig) {
	if config != nil {
		wh.config = config
	}
}

// SetMemoryWriter sets the memory writer (for testing)
func (wh *WriteHandler) SetMemoryWriter(memoryWriter *MemoryWriter) {
	wh.memoryWriter = memoryWriter
}

// SetContentProcessor sets the content processor (for testing)
func (wh *WriteHandler) SetContentProcessor(contentProcessor *ContentProcessor) {
	wh.contentProcessor = contentProcessor
}

// GetStats returns handler statistics
func (wh *WriteHandler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"max_content_length":        wh.config.MaxContentLength,
		"default_confidence":        wh.config.DefaultConfidence,
		"require_source":            wh.config.RequireSource,
		"enable_conflict_detection": wh.config.EnableConflictDetection,
		"processing_timeout":        wh.config.ProcessingTimeout.String(),
		"enable_deduplication":      wh.config.EnableDeduplication,
	}
}
