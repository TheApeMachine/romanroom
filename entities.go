package main

import (
	"fmt"
	"strings"
	"time"
)

// Entity represents an entity extracted from content
type Entity struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Confidence float64                `json:"confidence"`
	Source     string                 `json:"source"`
	CreatedAt  time.Time              `json:"created_at"`
}

// Claim represents a factual claim extracted from content
type Claim struct {
	ID         string                 `json:"id"`
	Subject    string                 `json:"subject"`
	Predicate  string                 `json:"predicate"`
	Object     string                 `json:"object"`
	Confidence float64                `json:"confidence"`
	Evidence   []string               `json:"evidence"`
	Source     string                 `json:"source"`
	CreatedAt  time.Time              `json:"created_at"`
}

// NewEntity creates a new Entity with the given name and type
func NewEntity(id, name, entityType, source string) *Entity {
	return &Entity{
		ID:         id,
		Name:       name,
		Type:       entityType,
		Properties: make(map[string]interface{}),
		Confidence: 1.0,
		Source:     source,
		CreatedAt:  time.Now(),
	}
}

// NewClaim creates a new Claim with the given subject, predicate, and object
func NewClaim(id, subject, predicate, object, source string) *Claim {
	return &Claim{
		ID:        id,
		Subject:   subject,
		Predicate: predicate,
		Object:    object,
		Confidence: 1.0,
		Evidence:  make([]string, 0),
		Source:    source,
		CreatedAt: time.Now(),
	}
}

// Validate checks if the entity has all required fields
func (e *Entity) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("entity ID cannot be empty")
	}
	if e.Name == "" {
		return fmt.Errorf("entity name cannot be empty")
	}
	if e.Type == "" {
		return fmt.Errorf("entity type cannot be empty")
	}
	if e.Source == "" {
		return fmt.Errorf("entity source cannot be empty")
	}
	if e.Confidence < 0 || e.Confidence > 1 {
		return fmt.Errorf("entity confidence must be between 0 and 1, got %f", e.Confidence)
	}
	return nil
}

// Validate checks if the claim has all required fields
func (c *Claim) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("claim ID cannot be empty")
	}
	if c.Subject == "" {
		return fmt.Errorf("claim subject cannot be empty")
	}
	if c.Predicate == "" {
		return fmt.Errorf("claim predicate cannot be empty")
	}
	if c.Object == "" {
		return fmt.Errorf("claim object cannot be empty")
	}
	if c.Source == "" {
		return fmt.Errorf("claim source cannot be empty")
	}
	if c.Confidence < 0 || c.Confidence > 1 {
		return fmt.Errorf("claim confidence must be between 0 and 1, got %f", c.Confidence)
	}
	return nil
}

// SetProperty sets a property on the entity
func (e *Entity) SetProperty(key string, value interface{}) {
	e.Properties[key] = value
}

// GetProperty gets a property from the entity
func (e *Entity) GetProperty(key string) (interface{}, bool) {
	value, exists := e.Properties[key]
	return value, exists
}

// AddEvidence adds evidence to support the claim
func (c *Claim) AddEvidence(evidence string) {
	c.Evidence = append(c.Evidence, evidence)
}

// String returns a string representation of the entity
func (e *Entity) String() string {
	return fmt.Sprintf("Entity{ID: %s, Name: %s, Type: %s, Confidence: %.2f}", 
		e.ID, e.Name, e.Type, e.Confidence)
}

// String returns a string representation of the claim
func (c *Claim) String() string {
	return fmt.Sprintf("Claim{ID: %s, Subject: %s, Predicate: %s, Object: %s, Confidence: %.2f}", 
		c.ID, c.Subject, c.Predicate, c.Object, c.Confidence)
}

// NormalizedName returns a normalized version of the entity name for comparison
func (e *Entity) NormalizedName() string {
	return strings.ToLower(strings.TrimSpace(e.Name))
}

// Triple returns the claim as a subject-predicate-object triple
func (c *Claim) Triple() string {
	return fmt.Sprintf("%s %s %s", c.Subject, c.Predicate, c.Object)
}

// HasEvidence returns true if the claim has supporting evidence
func (c *Claim) HasEvidence() bool {
	return len(c.Evidence) > 0
}

// EvidenceCount returns the number of evidence items for the claim
func (c *Claim) EvidenceCount() int {
	return len(c.Evidence)
}