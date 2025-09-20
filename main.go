package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("Agentic Memory System - Core Infrastructure")
	
	// Load default configuration
	config := DefaultServerConfig()
	
	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}
	
	fmt.Printf("Server: %s v%s\n", config.Server.Name, config.Server.Version)
	fmt.Printf("Vector Store: %s (%d dimensions)\n", 
		config.Storage.VectorStore.Provider, 
		config.Storage.VectorStore.Dimensions)
	fmt.Printf("Graph Store: %s\n", config.Storage.GraphStore.Provider)
	fmt.Printf("Search Index: %s\n", config.Storage.SearchIndex.Provider)
	
	// Demonstrate core data structures
	fmt.Println("\nDemonstrating core data structures:")
	
	// Create a chunk
	chunk := NewChunk("chunk-1", "This is sample content for the memory system", "demo")
	if err := chunk.Validate(); err != nil {
		log.Fatalf("Chunk validation failed: %v", err)
	}
	fmt.Printf("Created chunk: %s\n", chunk.ID)
	
	// Create an entity
	entity := NewEntity("entity-1", "Sample Entity", "CONCEPT", "demo")
	if err := entity.Validate(); err != nil {
		log.Fatalf("Entity validation failed: %v", err)
	}
	fmt.Printf("Created entity: %s\n", entity.String())
	
	// Create a claim
	claim := NewClaim("claim-1", "Sample Entity", "is a", "concept", "demo")
	if err := claim.Validate(); err != nil {
		log.Fatalf("Claim validation failed: %v", err)
	}
	fmt.Printf("Created claim: %s\n", claim.String())
	
	// Create graph nodes and edges
	node := NewNode("node-1", EntityNode)
	if err := node.Validate(); err != nil {
		log.Fatalf("Node validation failed: %v", err)
	}
	fmt.Printf("Created node: %s (type: %s)\n", node.ID, node.Type)
	
	edge := NewEdge("edge-1", "node-1", "node-2", RelatedTo, 0.8)
	if err := edge.Validate(); err != nil {
		log.Fatalf("Edge validation failed: %v", err)
	}
	fmt.Printf("Created edge: %s (%s -> %s, weight: %.2f)\n", 
		edge.ID, edge.From, edge.To, edge.Weight)
	
	// Add entity and claim to chunk
	chunk.AddEntity(*entity)
	chunk.AddClaim(*claim)
	
	fmt.Printf("Chunk now contains %d entities and %d claims\n", 
		len(chunk.Entities), len(chunk.Claims))
	
	fmt.Println("\nCore infrastructure setup complete!")
	fmt.Println("Ready for MCP server implementation and storage backends.")
}