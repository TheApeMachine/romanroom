package main

import (
	"context"
	"fmt"
	"time"
)

// MemoryWriter handles the creation and storage of memory chunks
type MemoryWriter struct {
	storage           *MultiViewStorage
	contentProcessor  *ContentProcessor
	entityResolver    *EntityResolver
	provenanceTracker *ProvenanceTracker
	config           *MemoryWriterConfig
}

// MemoryWriterConfig contains configuration for the memory writer
type MemoryWriterConfig struct {
	RequireEvidence    bool
	MaxChunkSize      int
	MinConfidence     float64
	EnableDeduplication bool
}

// NewMemoryWriter creates a new MemoryWriter instance
func NewMemoryWriter(storage *MultiViewStorage, contentProcessor *ContentProcessor, config *MemoryWriterConfig) *MemoryWriter {
	if config == nil {
		config = &MemoryWriterConfig{
			RequireEvidence:    false,
			MaxChunkSize:      1000,
			MinConfidence:     0.5,
			EnableDeduplication: true,
		}
	}

	return &MemoryWriter{
		storage:           storage,
		contentProcessor:  contentProcessor,
		entityResolver:    NewEntityResolver(storage),
		provenanceTracker: NewProvenanceTracker(),
		config:           config,
	}
}

// Write processes content and stores it as memory chunks
func (mw *MemoryWriter) Write(ctx context.Context, content string, metadata WriteMetadata) (*WriteResult, error) {
	// Process content to extract chunks, entities, and claims
	processedContent, err := mw.contentProcessor.Process(content, metadata.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to process content: %w", err)
	}

	// Create chunks from processed content
	chunks, err := mw.CreateChunk(processedContent, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunks: %w", err)
	}

	var storedChunks []string
	var entitiesLinked []string
	var conflictsFound []ConflictInfo

	// Store each chunk
	for _, chunk := range chunks {
		// Resolve entities if deduplication is enabled
		if mw.config.EnableDeduplication {
			resolvedEntities, err := mw.entityResolver.Resolve(ctx, chunk.Entities)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve entities: %w", err)
			}
			chunk.Entities = resolvedEntities
			for _, entity := range resolvedEntities {
				entitiesLinked = append(entitiesLinked, entity.ID)
			}
		}

		// Store the chunk
		chunkID, err := mw.StoreChunk(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to store chunk: %w", err)
		}
		storedChunks = append(storedChunks, chunkID)

		// Track provenance
		provenanceID, err := mw.provenanceTracker.Track(chunkID, metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to track provenance: %w", err)
		}
		chunk.Metadata["provenance_id"] = provenanceID
	}

	// Create write result
	result := &WriteResult{
		MemoryID:       storedChunks[0], // Primary chunk ID
		CandidateCount: len(chunks),
		ConflictsFound: conflictsFound,
		EntitiesLinked: entitiesLinked,
		ProvenanceID:   fmt.Sprintf("%v", chunks[0].Metadata["provenance_id"]),
	}

	return result, nil
}

// CreateChunk creates memory chunks from processed content
func (mw *MemoryWriter) CreateChunk(processedContent *ProcessingResult, metadata WriteMetadata) ([]*Chunk, error) {
	var chunks []*Chunk

	// Use the chunks from the processing result
	for _, chunk := range processedContent.Chunks {
		// Update chunk metadata with write metadata
		chunk.SetMetadata("user_id", metadata.UserID)
		chunk.SetMetadata("tags", metadata.Tags)
		chunk.SetMetadata("write_confidence", metadata.Confidence)
		
		// Override confidence if provided in metadata
		if metadata.Confidence > 0 {
			chunk.Confidence = metadata.Confidence
		}

		// Validate chunk meets minimum confidence threshold
		if chunk.Confidence < mw.config.MinConfidence {
			continue // Skip low-confidence chunks
		}

		// Validate evidence requirements for claims
		if mw.config.RequireEvidence && metadata.RequireEvidence {
			for _, claim := range chunk.Claims {
				if !claim.HasEvidence() {
					return nil, fmt.Errorf("claim requires evidence but none provided: %s", claim.Triple())
				}
			}
		}

		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("no valid chunks created from content")
	}

	return chunks, nil
}

// StoreChunk stores a chunk in the multi-view storage system
func (mw *MemoryWriter) StoreChunk(ctx context.Context, chunk *Chunk) (string, error) {
	// Store in vector database
	if err := mw.storage.vectorStore.Store(ctx, chunk.ID, chunk.Embedding, chunk.Metadata); err != nil {
		return "", fmt.Errorf("failed to store in vector database: %w", err)
	}

	// Store in search index
	doc := IndexDocument{
		ID:       chunk.ID,
		Content:  chunk.Content,
		Metadata: chunk.Metadata,
	}
	if err := mw.storage.searchIndex.Index(ctx, doc); err != nil {
		return "", fmt.Errorf("failed to index in search: %w", err)
	}

	// Store entities and claims in graph database
	for _, entity := range chunk.Entities {
		node := &Node{
			ID:   entity.ID,
			Type: EntityNode,
			Properties: map[string]interface{}{
				"name":        entity.Name,
				"type":        entity.Type,
				"confidence":  entity.Confidence,
				"chunk_id":    chunk.ID,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		if err := mw.storage.graphStore.CreateNode(ctx, node); err != nil {
			return "", fmt.Errorf("failed to create entity node: %w", err)
		}
	}

	for _, claim := range chunk.Claims {
		node := &Node{
			ID:   claim.ID,
			Type: ClaimNode,
			Properties: map[string]interface{}{
				"subject":     claim.Subject,
				"predicate":   claim.Predicate,
				"object":      claim.Object,
				"confidence":  claim.Confidence,
				"chunk_id":    chunk.ID,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		if err := mw.storage.graphStore.CreateNode(ctx, node); err != nil {
			return "", fmt.Errorf("failed to create claim node: %w", err)
		}
	}

	return chunk.ID, nil
}