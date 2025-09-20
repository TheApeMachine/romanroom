package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ProvenanceTracker tracks the lineage and history of memories
type ProvenanceTracker struct {
	records map[string]*ProvenanceRecord
	mutex   sync.RWMutex
	config  *ProvenanceConfig
}

// ProvenanceConfig contains configuration for provenance tracking
type ProvenanceConfig struct {
	EnableVersioning     bool
	MaxVersionHistory   int
	TrackModifications  bool
	EnableIntegrityCheck bool
}

// ProvenanceRecord represents the complete lineage of a memory
type ProvenanceRecord struct {
	ID              string                 `json:"id"`
	MemoryID        string                 `json:"memory_id"`
	OriginalSource  string                 `json:"original_source"`
	CreatedAt       time.Time              `json:"created_at"`
	CreatedBy       string                 `json:"created_by"`
	LastModified    time.Time              `json:"last_modified"`
	ModifiedBy      string                 `json:"modified_by"`
	Version         int                    `json:"version"`
	ParentVersions  []string               `json:"parent_versions"`
	Transformations []Transformation       `json:"transformations"`
	Metadata        map[string]interface{} `json:"metadata"`
	IntegrityHash   string                 `json:"integrity_hash"`
}

// Transformation represents a modification or processing step
type Transformation struct {
	ID          string                 `json:"id"`
	Type        TransformationType     `json:"type"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Agent       string                 `json:"agent"`
	Parameters  map[string]interface{} `json:"parameters"`
	InputHash   string                 `json:"input_hash"`
	OutputHash  string                 `json:"output_hash"`
}

// TransformationType represents different types of transformations
type TransformationType string

const (
	TransformationChunking      TransformationType = "chunking"
	TransformationEmbedding     TransformationType = "embedding"
	TransformationEntityExtraction TransformationType = "entity_extraction"
	TransformationClaimExtraction  TransformationType = "claim_extraction"
	TransformationResolution    TransformationType = "entity_resolution"
	TransformationMerge         TransformationType = "merge"
	TransformationUpdate        TransformationType = "update"
	TransformationDelete        TransformationType = "delete"
)

// NewProvenanceTracker creates a new ProvenanceTracker instance
func NewProvenanceTracker() *ProvenanceTracker {
	config := &ProvenanceConfig{
		EnableVersioning:     true,
		MaxVersionHistory:   10,
		TrackModifications:  true,
		EnableIntegrityCheck: true,
	}

	return &ProvenanceTracker{
		records: make(map[string]*ProvenanceRecord),
		config:  config,
	}
}

// Track creates a new provenance record for a memory
func (pt *ProvenanceTracker) Track(memoryID string, metadata WriteMetadata) (string, error) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	provenanceID := pt.generateProvenanceID(memoryID, metadata)
	
	record := &ProvenanceRecord{
		ID:             provenanceID,
		MemoryID:       memoryID,
		OriginalSource: metadata.Source,
		CreatedAt:      metadata.Timestamp,
		CreatedBy:      metadata.UserID,
		LastModified:   metadata.Timestamp,
		ModifiedBy:     metadata.UserID,
		Version:        1,
		ParentVersions: []string{},
		Transformations: []Transformation{},
		Metadata: map[string]interface{}{
			"tags":       metadata.Tags,
			"confidence": metadata.Confidence,
			"source":     metadata.Source,
		},
	}

	// Calculate integrity hash if enabled
	if pt.config.EnableIntegrityCheck {
		hash, err := pt.calculateIntegrityHash(record)
		if err != nil {
			return "", fmt.Errorf("failed to calculate integrity hash: %w", err)
		}
		record.IntegrityHash = hash
	}

	pt.records[provenanceID] = record
	return provenanceID, nil
}

// GetProvenance retrieves the provenance record for a memory
func (pt *ProvenanceTracker) GetProvenance(provenanceID string) (*ProvenanceRecord, error) {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	record, exists := pt.records[provenanceID]
	if !exists {
		return nil, fmt.Errorf("provenance record not found: %s", provenanceID)
	}

	// Verify integrity if enabled
	if pt.config.EnableIntegrityCheck {
		if err := pt.verifyIntegrity(record); err != nil {
			return nil, fmt.Errorf("integrity check failed: %w", err)
		}
	}

	// Return a copy to prevent external modifications
	recordCopy := *record
	return &recordCopy, nil
}

// UpdateProvenance updates an existing provenance record
func (pt *ProvenanceTracker) UpdateProvenance(provenanceID string, transformation Transformation) error {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	record, exists := pt.records[provenanceID]
	if !exists {
		return fmt.Errorf("provenance record not found: %s", provenanceID)
	}

	// Create new version if versioning is enabled
	if pt.config.EnableVersioning {
		newVersion := record.Version + 1
		
		// Limit version history
		if len(record.ParentVersions) >= pt.config.MaxVersionHistory {
			record.ParentVersions = record.ParentVersions[1:]
		}
		record.ParentVersions = append(record.ParentVersions, fmt.Sprintf("%s_v%d", provenanceID, record.Version))
		record.Version = newVersion
	}

	// Add transformation
	record.Transformations = append(record.Transformations, transformation)
	record.LastModified = transformation.Timestamp
	record.ModifiedBy = transformation.Agent

	// Recalculate integrity hash
	if pt.config.EnableIntegrityCheck {
		hash, err := pt.calculateIntegrityHash(record)
		if err != nil {
			return fmt.Errorf("failed to update integrity hash: %w", err)
		}
		record.IntegrityHash = hash
	}

	return nil
}

// TrackTransformation adds a transformation to the provenance record
func (pt *ProvenanceTracker) TrackTransformation(provenanceID string, transformationType TransformationType, description string, agent string, parameters map[string]interface{}) error {
	transformation := Transformation{
		ID:          pt.generateTransformationID(provenanceID, transformationType),
		Type:        transformationType,
		Description: description,
		Timestamp:   time.Now(),
		Agent:       agent,
		Parameters:  parameters,
	}

	return pt.UpdateProvenance(provenanceID, transformation)
}

// GetMemoryLineage returns the complete lineage of a memory including all versions
func (pt *ProvenanceTracker) GetMemoryLineage(memoryID string) ([]*ProvenanceRecord, error) {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	var lineage []*ProvenanceRecord
	
	// Find all records for this memory ID
	for _, record := range pt.records {
		if record.MemoryID == memoryID {
			recordCopy := *record
			lineage = append(lineage, &recordCopy)
		}
	}

	if len(lineage) == 0 {
		return nil, fmt.Errorf("no provenance records found for memory: %s", memoryID)
	}

	return lineage, nil
}

// GetTransformationHistory returns all transformations for a memory
func (pt *ProvenanceTracker) GetTransformationHistory(provenanceID string) ([]Transformation, error) {
	record, err := pt.GetProvenance(provenanceID)
	if err != nil {
		return nil, err
	}

	return record.Transformations, nil
}

// VerifyIntegrity verifies the integrity of all provenance records
func (pt *ProvenanceTracker) VerifyIntegrity() error {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	if !pt.config.EnableIntegrityCheck {
		return nil
	}

	for id, record := range pt.records {
		if err := pt.verifyIntegrity(record); err != nil {
			return fmt.Errorf("integrity check failed for record %s: %w", id, err)
		}
	}

	return nil
}

// generateProvenanceID generates a unique ID for a provenance record
func (pt *ProvenanceTracker) generateProvenanceID(memoryID string, metadata WriteMetadata) string {
	data := fmt.Sprintf("%s_%s_%d", memoryID, metadata.Source, metadata.Timestamp.Unix())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("prov_%x", hash[:8])
}

// generateTransformationID generates a unique ID for a transformation
func (pt *ProvenanceTracker) generateTransformationID(provenanceID string, transformationType TransformationType) string {
	data := fmt.Sprintf("%s_%s_%d", provenanceID, transformationType, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("trans_%x", hash[:8])
}

// calculateIntegrityHash calculates a hash for integrity verification
func (pt *ProvenanceTracker) calculateIntegrityHash(record *ProvenanceRecord) (string, error) {
	// Create a copy without the integrity hash for calculation
	recordForHash := *record
	recordForHash.IntegrityHash = ""

	data, err := json.Marshal(recordForHash)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

// verifyIntegrity verifies the integrity hash of a record
func (pt *ProvenanceTracker) verifyIntegrity(record *ProvenanceRecord) error {
	if record.IntegrityHash == "" {
		return nil // No hash to verify
	}

	expectedHash, err := pt.calculateIntegrityHash(record)
	if err != nil {
		return err
	}

	if expectedHash != record.IntegrityHash {
		return fmt.Errorf("integrity hash mismatch: expected %s, got %s", expectedHash, record.IntegrityHash)
	}

	return nil
}

// GetStats returns statistics about provenance tracking
func (pt *ProvenanceTracker) GetStats() map[string]interface{} {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	transformationCounts := make(map[TransformationType]int)
	totalTransformations := 0

	for _, record := range pt.records {
		for _, transformation := range record.Transformations {
			transformationCounts[transformation.Type]++
			totalTransformations++
		}
	}

	return map[string]interface{}{
		"total_records":        len(pt.records),
		"total_transformations": totalTransformations,
		"transformation_counts": transformationCounts,
		"versioning_enabled":   pt.config.EnableVersioning,
		"integrity_enabled":    pt.config.EnableIntegrityCheck,
	}
}