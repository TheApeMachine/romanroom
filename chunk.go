package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// Chunk represents a piece of content stored in the memory system
type Chunk struct {
	ID          string                 `json:"id"`
	Content     string                 `json:"content"`
	Embedding   []float32              `json:"embedding"`
	Metadata    map[string]interface{} `json:"metadata"`
	Claims      []Claim                `json:"claims"`
	Entities    []Entity               `json:"entities"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source"`
	Confidence  float64                `json:"confidence"`
}

// NewChunk creates a new Chunk with the given content and source
func NewChunk(id, content, source string) *Chunk {
	return &Chunk{
		ID:        id,
		Content:   content,
		Source:    source,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
		Claims:    make([]Claim, 0),
		Entities:  make([]Entity, 0),
		Confidence: 1.0,
	}
}

// Validate checks if the chunk has all required fields
func (c *Chunk) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("chunk ID cannot be empty")
	}
	if c.Content == "" {
		return fmt.Errorf("chunk content cannot be empty")
	}
	if c.Source == "" {
		return fmt.Errorf("chunk source cannot be empty")
	}
	if c.Confidence < 0 || c.Confidence > 1 {
		return fmt.Errorf("chunk confidence must be between 0 and 1, got %f", c.Confidence)
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling for Chunk
func (c *Chunk) MarshalJSON() ([]byte, error) {
	type Alias Chunk
	return json.Marshal(&struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (*Alias)(c),
		Timestamp: c.Timestamp.Format(time.RFC3339),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Chunk
func (c *Chunk) UnmarshalJSON(data []byte) error {
	type Alias Chunk
	aux := &struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias: (*Alias)(c),
	}
	
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	
	if aux.Timestamp != "" {
		t, err := time.Parse(time.RFC3339, aux.Timestamp)
		if err != nil {
			return fmt.Errorf("invalid timestamp format: %v", err)
		}
		c.Timestamp = t
	}
	
	return nil
}

// AddClaim adds a claim to the chunk
func (c *Chunk) AddClaim(claim Claim) {
	c.Claims = append(c.Claims, claim)
}

// AddEntity adds an entity to the chunk
func (c *Chunk) AddEntity(entity Entity) {
	c.Entities = append(c.Entities, entity)
}

// SetEmbedding sets the embedding vector for the chunk
func (c *Chunk) SetEmbedding(embedding []float32) {
	c.Embedding = embedding
}

// GetMetadata returns a metadata value by key
func (c *Chunk) GetMetadata(key string) (interface{}, bool) {
	value, exists := c.Metadata[key]
	return value, exists
}

// SetMetadata sets a metadata key-value pair
func (c *Chunk) SetMetadata(key string, value interface{}) {
	c.Metadata[key] = value
}