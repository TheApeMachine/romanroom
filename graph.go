package main

import (
	"fmt"
	"time"
)

// NodeType represents the type of a graph node
type NodeType string

const (
	EntityNode       NodeType = "Entity"
	ClaimNode        NodeType = "Claim"
	EventNode        NodeType = "Event"
	TaskNode         NodeType = "Task"
	ConversationNode NodeType = "ConversationTurn"
	SourceNode       NodeType = "Source"
)

// EdgeType represents the type of a graph edge
type EdgeType string

const (
	RelatedTo    EdgeType = "RELATED_TO"
	PartOf       EdgeType = "PART_OF"
	Supports     EdgeType = "SUPPORTS"
	Refutes      EdgeType = "REFUTES"
	TemporalNext EdgeType = "TEMPORAL_NEXT"
	CausedBy     EdgeType = "CAUSED_BY"
)

// Node represents a node in the memory graph
type Node struct {
	ID         string                 `json:"id"`
	Type       NodeType               `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Embedding  []float32              `json:"embedding,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// Edge represents an edge in the memory graph
type Edge struct {
	ID         string                 `json:"id"`
	From       string                 `json:"from"`
	To         string                 `json:"to"`
	Type       EdgeType               `json:"type"`
	Weight     float64                `json:"weight"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"created_at"`
}

// NewNode creates a new Node with the given ID and type
func NewNode(id string, nodeType NodeType) *Node {
	now := time.Now()
	return &Node{
		ID:         id,
		Type:       nodeType,
		Properties: make(map[string]interface{}),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// NewEdge creates a new Edge between two nodes
func NewEdge(id, from, to string, edgeType EdgeType, weight float64) *Edge {
	return &Edge{
		ID:         id,
		From:       from,
		To:         to,
		Type:       edgeType,
		Weight:     weight,
		Properties: make(map[string]interface{}),
		CreatedAt:  time.Now(),
	}
}

// Validate checks if the node has all required fields
func (n *Node) Validate() error {
	if n.ID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}
	if n.Type == "" {
		return fmt.Errorf("node type cannot be empty")
	}
	if !isValidNodeType(n.Type) {
		return fmt.Errorf("invalid node type: %s", n.Type)
	}
	return nil
}

// Validate checks if the edge has all required fields
func (e *Edge) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("edge ID cannot be empty")
	}
	if e.From == "" {
		return fmt.Errorf("edge 'from' node cannot be empty")
	}
	if e.To == "" {
		return fmt.Errorf("edge 'to' node cannot be empty")
	}
	if e.Type == "" {
		return fmt.Errorf("edge type cannot be empty")
	}
	if !isValidEdgeType(e.Type) {
		return fmt.Errorf("invalid edge type: %s", e.Type)
	}
	if e.Weight < 0 {
		return fmt.Errorf("edge weight cannot be negative, got %f", e.Weight)
	}
	return nil
}

// SetProperty sets a property on the node
func (n *Node) SetProperty(key string, value interface{}) {
	n.Properties[key] = value
	n.UpdatedAt = time.Now()
}

// GetProperty gets a property from the node
func (n *Node) GetProperty(key string) (interface{}, bool) {
	value, exists := n.Properties[key]
	return value, exists
}

// SetProperty sets a property on the edge
func (e *Edge) SetProperty(key string, value interface{}) {
	e.Properties[key] = value
}

// GetProperty gets a property from the edge
func (e *Edge) GetProperty(key string) (interface{}, bool) {
	value, exists := e.Properties[key]
	return value, exists
}

// SetEmbedding sets the embedding vector for the node
func (n *Node) SetEmbedding(embedding []float32) {
	n.Embedding = embedding
	n.UpdatedAt = time.Now()
}

// isValidNodeType checks if the given node type is valid
func isValidNodeType(nodeType NodeType) bool {
	validTypes := []NodeType{
		EntityNode, ClaimNode, EventNode, TaskNode, ConversationNode, SourceNode,
	}
	for _, validType := range validTypes {
		if nodeType == validType {
			return true
		}
	}
	return false
}

// isValidEdgeType checks if the given edge type is valid
func isValidEdgeType(edgeType EdgeType) bool {
	validTypes := []EdgeType{
		RelatedTo, PartOf, Supports, Refutes, TemporalNext, CausedBy,
	}
	for _, validType := range validTypes {
		if edgeType == validType {
			return true
		}
	}
	return false
}