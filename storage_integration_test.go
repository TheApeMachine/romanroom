package main

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestStorageIntegration(t *testing.T) {
	Convey("Given integrated file-based storage backends", t, func() {
		tempDir := t.TempDir()
		
		// Initialize all storage backends
		vectorStore := NewFileVectorStore(filepath.Join(tempDir, "vectors.json"))
		graphStore := NewFileGraphStore(filepath.Join(tempDir, "graph.json"))
		searchIndex := NewFileSearchIndex(filepath.Join(tempDir, "search.json"))
		
		ctx := context.Background()
		
		Convey("When storing related data across all backends", func() {
			// Store vector embeddings
			embeddings := []struct {
				id        string
				embedding []float32
				metadata  map[string]interface{}
			}{
				{"entity1", []float32{0.1, 0.2, 0.3}, map[string]interface{}{"type": "entity", "name": "Alice"}},
				{"entity2", []float32{0.4, 0.5, 0.6}, map[string]interface{}{"type": "entity", "name": "Bob"}},
				{"claim1", []float32{0.7, 0.8, 0.9}, map[string]interface{}{"type": "claim", "content": "Alice knows Bob"}},
			}
			
			for _, emb := range embeddings {
				err := vectorStore.Store(ctx, emb.id, emb.embedding, emb.metadata)
				So(err, ShouldBeNil)
			}
			
			// Create graph nodes and edges
			nodes := []*Node{
				NewNode("entity1", EntityNode),
				NewNode("entity2", EntityNode),
				NewNode("claim1", ClaimNode),
			}
			
			nodes[0].SetProperty("name", "Alice")
			nodes[1].SetProperty("name", "Bob")
			nodes[2].SetProperty("content", "Alice knows Bob")
			
			for _, node := range nodes {
				err := graphStore.CreateNode(ctx, node)
				So(err, ShouldBeNil)
			}
			
			edges := []*Edge{
				NewEdge("edge1", "entity1", "claim1", Supports, 0.9),
				NewEdge("edge2", "entity2", "claim1", Supports, 0.8),
				NewEdge("edge3", "entity1", "entity2", RelatedTo, 0.7),
			}
			
			for _, edge := range edges {
				err := graphStore.CreateEdge(ctx, edge)
				So(err, ShouldBeNil)
			}
			
			// Index documents for search
			documents := []IndexDocument{
				{
					ID:      "entity1",
					Content: "Alice is a person who works in artificial intelligence",
					Metadata: map[string]interface{}{"type": "entity", "name": "Alice"},
				},
				{
					ID:      "entity2",
					Content: "Bob is a researcher in machine learning and data science",
					Metadata: map[string]interface{}{"type": "entity", "name": "Bob"},
				},
				{
					ID:      "claim1",
					Content: "Alice knows Bob through their work in AI research",
					Metadata: map[string]interface{}{"type": "claim", "relationship": "knows"},
				},
			}
			
			for _, doc := range documents {
				err := searchIndex.Index(ctx, doc)
				So(err, ShouldBeNil)
			}
			
			Convey("Then cross-backend queries should work correctly", func() {
				// Test vector similarity search
				query := []float32{0.1, 0.2, 0.3}
				vectorResults, err := vectorStore.Search(ctx, query, 3, nil)
				So(err, ShouldBeNil)
				So(len(vectorResults), ShouldEqual, 3)
				So(vectorResults[0].ID, ShouldEqual, "entity1") // Most similar
				
				// Test graph traversal
				neighbors, err := graphStore.GetNeighbors(ctx, "entity1", GraphTraversalOptions{})
				So(err, ShouldBeNil)
				So(len(neighbors), ShouldEqual, 2) // claim1 and entity2
				
				// Test text search
				searchResults, err := searchIndex.Search(ctx, "artificial intelligence", SearchIndexOptions{})
				So(err, ShouldBeNil)
				So(len(searchResults), ShouldBeGreaterThan, 0)
				
				// Verify data consistency across backends
				for _, result := range vectorResults {
					// Check if corresponding node exists in graph
					node, err := graphStore.GetNode(ctx, result.ID)
					So(err, ShouldBeNil)
					So(node.ID, ShouldEqual, result.ID)
					
					// Check if corresponding document exists in search index
					doc, err := searchIndex.GetDocument(ctx, result.ID)
					So(err, ShouldBeNil)
					So(doc.ID, ShouldEqual, result.ID)
				}
			})
			
			Convey("And complex multi-backend operations should work", func() {
				// Find entities related to "Alice" through multiple backends
				
				// 1. Search for documents mentioning Alice
				aliceSearchResults, err := searchIndex.Search(ctx, "Alice", SearchIndexOptions{})
				So(err, ShouldBeNil)
				So(len(aliceSearchResults), ShouldBeGreaterThan, 0)
				
				// 2. Get Alice's neighbors in the graph
				aliceNeighbors, err := graphStore.GetNeighbors(ctx, "entity1", GraphTraversalOptions{})
				So(err, ShouldBeNil)
				So(len(aliceNeighbors), ShouldEqual, 2)
				
				// 3. Find similar vectors to Alice's embedding
				aliceVector, err := vectorStore.GetByID(ctx, "entity1")
				So(err, ShouldBeNil)
				
				similarVectors, err := vectorStore.Search(ctx, aliceVector.Embedding, 3, nil)
				So(err, ShouldBeNil)
				So(len(similarVectors), ShouldEqual, 3)
				
				// Verify that all backends return consistent information about Alice
				So(aliceVector.Metadata["name"], ShouldEqual, "Alice")
				
				aliceNode, err := graphStore.GetNode(ctx, "entity1")
				So(err, ShouldBeNil)
				aliceName, _ := aliceNode.GetProperty("name")
				So(aliceName, ShouldEqual, "Alice")
				
				aliceDoc, err := searchIndex.GetDocument(ctx, "entity1")
				So(err, ShouldBeNil)
				So(aliceDoc.Metadata["name"], ShouldEqual, "Alice")
			})
		})
		
		Convey("When performing batch operations across backends", func() {
			// Batch store vectors
			vectorItems := []VectorStoreItem{
				{"batch1", []float32{1.0, 0.0}, map[string]interface{}{"batch": true}},
				{"batch2", []float32{0.0, 1.0}, map[string]interface{}{"batch": true}},
			}
			
			err := vectorStore.BatchStore(ctx, vectorItems)
			So(err, ShouldBeNil)
			
			// Create corresponding nodes
			batchNodes := []*Node{
				NewNode("batch1", EntityNode),
				NewNode("batch2", EntityNode),
			}
			
			for _, node := range batchNodes {
				node.SetProperty("batch", true)
				err := graphStore.CreateNode(ctx, node)
				So(err, ShouldBeNil)
			}
			
			// Batch index documents
			batchDocs := []IndexDocument{
				{ID: "batch1", Content: "first batch document", Metadata: map[string]interface{}{"batch": true}},
				{ID: "batch2", Content: "second batch document", Metadata: map[string]interface{}{"batch": true}},
			}
			
			err = searchIndex.BatchIndex(ctx, batchDocs)
			So(err, ShouldBeNil)
			
			Convey("Then all backends should have the batch data", func() {
				// Verify vector store
				vectorCount, err := vectorStore.Count(ctx)
				So(err, ShouldBeNil)
				So(vectorCount, ShouldBeGreaterThanOrEqualTo, 2)
				
				// Verify graph store
				nodeCount, err := graphStore.NodeCount(ctx)
				So(err, ShouldBeNil)
				So(nodeCount, ShouldBeGreaterThanOrEqualTo, 2)
				
				// Verify search index
				docCount, err := searchIndex.DocumentCount(ctx)
				So(err, ShouldBeNil)
				So(docCount, ShouldBeGreaterThanOrEqualTo, 2)
				
				// Verify data consistency
				for _, item := range vectorItems {
					vector, err := vectorStore.GetByID(ctx, item.ID)
					So(err, ShouldBeNil)
					So(vector.Metadata["batch"], ShouldEqual, true)
					
					node, err := graphStore.GetNode(ctx, item.ID)
					So(err, ShouldBeNil)
					batch, _ := node.GetProperty("batch")
					So(batch, ShouldEqual, true)
					
					doc, err := searchIndex.GetDocument(ctx, item.ID)
					So(err, ShouldBeNil)
					So(doc.Metadata["batch"], ShouldEqual, true)
				}
			})
		})
		
		Convey("When testing persistence across all backends", func() {
			// Store data in all backends
			testID := "persist_test"
			
			// Vector store
			err := vectorStore.Store(ctx, testID, []float32{0.5, 0.5}, map[string]interface{}{"persistent": true})
			So(err, ShouldBeNil)
			err = vectorStore.Save()
			So(err, ShouldBeNil)
			
			// Graph store
			node := NewNode(testID, EntityNode)
			node.SetProperty("persistent", true)
			err = graphStore.CreateNode(ctx, node)
			So(err, ShouldBeNil)
			err = graphStore.Save()
			So(err, ShouldBeNil)
			
			// Search index
			doc := IndexDocument{
				ID:      testID,
				Content: "persistent test document",
				Metadata: map[string]interface{}{"persistent": true},
			}
			err = searchIndex.Index(ctx, doc)
			So(err, ShouldBeNil)
			err = searchIndex.Save()
			So(err, ShouldBeNil)
			
			// Create new instances and load
			newVectorStore := NewFileVectorStore(filepath.Join(tempDir, "vectors.json"))
			newGraphStore := NewFileGraphStore(filepath.Join(tempDir, "graph.json"))
			newSearchIndex := NewFileSearchIndex(filepath.Join(tempDir, "search.json"))
			
			err = newVectorStore.Load()
			So(err, ShouldBeNil)
			err = newGraphStore.Load()
			So(err, ShouldBeNil)
			err = newSearchIndex.Load()
			So(err, ShouldBeNil)
			
			Convey("Then all data should persist correctly", func() {
				// Verify vector store persistence
				vector, err := newVectorStore.GetByID(ctx, testID)
				So(err, ShouldBeNil)
				So(vector.Metadata["persistent"], ShouldEqual, true)
				
				// Verify graph store persistence
				persistedNode, err := newGraphStore.GetNode(ctx, testID)
				So(err, ShouldBeNil)
				persistent, _ := persistedNode.GetProperty("persistent")
				So(persistent, ShouldEqual, true)
				
				// Verify search index persistence
				persistedDoc, err := newSearchIndex.GetDocument(ctx, testID)
				So(err, ShouldBeNil)
				So(persistedDoc.Metadata["persistent"], ShouldEqual, true)
				So(persistedDoc.Content, ShouldEqual, "persistent test document")
			})
		})
		
		Convey("When testing health checks across all backends", func() {
			err := vectorStore.Health(ctx)
			So(err, ShouldBeNil)
			
			err = graphStore.Health(ctx)
			So(err, ShouldBeNil)
			
			err = searchIndex.Health(ctx)
			So(err, ShouldBeNil)
			
			Convey("Then all backends should be healthy", func() {
				// All health checks passed above
			})
		})
		
		Convey("When closing all backends", func() {
			err := vectorStore.Close()
			So(err, ShouldBeNil)
			
			err = graphStore.Close()
			So(err, ShouldBeNil)
			
			err = searchIndex.Close()
			So(err, ShouldBeNil)
			
			Convey("Then subsequent operations should fail", func() {
				err := vectorStore.Store(ctx, "test", []float32{1.0}, nil)
				So(err, ShouldNotBeNil)
				
				node := NewNode("test", EntityNode)
				err = graphStore.CreateNode(ctx, node)
				So(err, ShouldNotBeNil)
				
				doc := IndexDocument{ID: "test", Content: "test", Metadata: nil}
				err = searchIndex.Index(ctx, doc)
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestStorageIntegrationScenarios(t *testing.T) {
	Convey("Given a complete memory system scenario", t, func() {
		tempDir := t.TempDir()
		
		vectorStore := NewFileVectorStore(filepath.Join(tempDir, "vectors.json"))
		graphStore := NewFileGraphStore(filepath.Join(tempDir, "graph.json"))
		searchIndex := NewFileSearchIndex(filepath.Join(tempDir, "search.json"))
		
		ctx := context.Background()
		
		Convey("When simulating a conversation memory scenario", func() {
			// Simulate storing a conversation turn
			conversationID := "conv_001"
			turnID := "turn_001"
			
			// Store conversation embedding
			conversationEmbedding := []float32{0.2, 0.4, 0.6, 0.8}
			conversationMetadata := map[string]interface{}{
				"type":           "conversation",
				"conversation_id": conversationID,
				"turn_id":        turnID,
				"speaker":        "user",
				"timestamp":      "2024-01-01T10:00:00Z",
			}
			
			err := vectorStore.Store(ctx, turnID, conversationEmbedding, conversationMetadata)
			So(err, ShouldBeNil)
			
			// Create conversation node in graph
			conversationNode := NewNode(turnID, ConversationNode)
			conversationNode.SetProperty("conversation_id", conversationID)
			conversationNode.SetProperty("speaker", "user")
			conversationNode.SetProperty("content", "What is artificial intelligence?")
			
			err = graphStore.CreateNode(ctx, conversationNode)
			So(err, ShouldBeNil)
			
			// Index conversation content for search
			conversationDoc := IndexDocument{
				ID:      turnID,
				Content: "What is artificial intelligence? User question about AI concepts and definitions",
				Metadata: conversationMetadata,
			}
			
			err = searchIndex.Index(ctx, conversationDoc)
			So(err, ShouldBeNil)
			
			// Create related entities and claims
			aiEntityID := "entity_ai"
			aiClaimID := "claim_ai_definition"
			
			// AI entity
			aiEmbedding := []float32{0.1, 0.3, 0.7, 0.9}
			aiMetadata := map[string]interface{}{
				"type": "entity",
				"name": "Artificial Intelligence",
			}
			
			err = vectorStore.Store(ctx, aiEntityID, aiEmbedding, aiMetadata)
			So(err, ShouldBeNil)
			
			aiNode := NewNode(aiEntityID, EntityNode)
			aiNode.SetProperty("name", "Artificial Intelligence")
			aiNode.SetProperty("category", "technology")
			
			err = graphStore.CreateNode(ctx, aiNode)
			So(err, ShouldBeNil)
			
			aiDoc := IndexDocument{
				ID:      aiEntityID,
				Content: "Artificial Intelligence AI machine learning computer science technology",
				Metadata: aiMetadata,
			}
			
			err = searchIndex.Index(ctx, aiDoc)
			So(err, ShouldBeNil)
			
			// AI definition claim
			claimEmbedding := []float32{0.15, 0.35, 0.65, 0.85}
			claimMetadata := map[string]interface{}{
				"type":    "claim",
				"subject": "AI definition",
			}
			
			err = vectorStore.Store(ctx, aiClaimID, claimEmbedding, claimMetadata)
			So(err, ShouldBeNil)
			
			claimNode := NewNode(aiClaimID, ClaimNode)
			claimNode.SetProperty("content", "AI is the simulation of human intelligence in machines")
			claimNode.SetProperty("confidence", 0.9)
			
			err = graphStore.CreateNode(ctx, claimNode)
			So(err, ShouldBeNil)
			
			claimDoc := IndexDocument{
				ID:      aiClaimID,
				Content: "AI is the simulation of human intelligence in machines that are programmed to think and learn",
				Metadata: claimMetadata,
			}
			
			err = searchIndex.Index(ctx, claimDoc)
			So(err, ShouldBeNil)
			
			// Create relationships
			conversationToAI := NewEdge("edge_conv_ai", turnID, aiEntityID, RelatedTo, 0.8)
			conversationToAI.SetProperty("relationship_type", "asks_about")
			
			err = graphStore.CreateEdge(ctx, conversationToAI)
			So(err, ShouldBeNil)
			
			aiToClaim := NewEdge("edge_ai_claim", aiEntityID, aiClaimID, Supports, 0.9)
			aiToClaim.SetProperty("relationship_type", "defined_by")
			
			err = graphStore.CreateEdge(ctx, aiToClaim)
			So(err, ShouldBeNil)
			
			Convey("Then the memory system should support complex queries", func() {
				// Query 1: Find similar conversations
				similarConversations, err := vectorStore.Search(ctx, conversationEmbedding, 5, 
					map[string]interface{}{"type": "conversation"})
				So(err, ShouldBeNil)
				So(len(similarConversations), ShouldEqual, 1)
				So(similarConversations[0].ID, ShouldEqual, turnID)
				
				// Query 2: Find what the conversation is about through graph traversal
				conversationNeighbors, err := graphStore.GetNeighbors(ctx, turnID, GraphTraversalOptions{})
				So(err, ShouldBeNil)
				So(len(conversationNeighbors), ShouldEqual, 1)
				So(conversationNeighbors[0].ID, ShouldEqual, aiEntityID)
				
				// Query 3: Search for AI-related content
				aiSearchResults, err := searchIndex.Search(ctx, "artificial intelligence", SearchIndexOptions{})
				So(err, ShouldBeNil)
				So(len(aiSearchResults), ShouldBeGreaterThanOrEqualTo, 2) // AI entity and conversation
				
				// Query 4: Find claims about AI
				aiClaims, err := graphStore.GetNeighbors(ctx, aiEntityID, GraphTraversalOptions{
					NodeTypes: []NodeType{ClaimNode},
				})
				So(err, ShouldBeNil)
				So(len(aiClaims), ShouldEqual, 1)
				So(aiClaims[0].ID, ShouldEqual, aiClaimID)
				
				// Query 5: Multi-modal search combining vector similarity and text search
				// Find entities similar to AI
				similarEntities, err := vectorStore.Search(ctx, aiEmbedding, 3, 
					map[string]interface{}{"type": "entity"})
				So(err, ShouldBeNil)
				So(len(similarEntities), ShouldEqual, 1)
				
				// Search for content related to those entities
				for _, entity := range similarEntities {
					if entityName, exists := entity.Metadata["name"]; exists {
						searchResults, err := searchIndex.Search(ctx, entityName.(string), SearchIndexOptions{})
						So(err, ShouldBeNil)
						So(len(searchResults), ShouldBeGreaterThan, 0)
					}
				}
			})
			
			Convey("And the system should handle updates consistently", func() {
				// Update conversation with follow-up
				followUpID := "turn_002"
				followUpEmbedding := []float32{0.25, 0.45, 0.65, 0.85}
				followUpMetadata := map[string]interface{}{
					"type":           "conversation",
					"conversation_id": conversationID,
					"turn_id":        followUpID,
					"speaker":        "assistant",
					"timestamp":      "2024-01-01T10:01:00Z",
				}
				
				err := vectorStore.Store(ctx, followUpID, followUpEmbedding, followUpMetadata)
				So(err, ShouldBeNil)
				
				followUpNode := NewNode(followUpID, ConversationNode)
				followUpNode.SetProperty("conversation_id", conversationID)
				followUpNode.SetProperty("speaker", "assistant")
				followUpNode.SetProperty("content", "AI is a broad field of computer science...")
				
				err = graphStore.CreateNode(ctx, followUpNode)
				So(err, ShouldBeNil)
				
				followUpDoc := IndexDocument{
					ID:      followUpID,
					Content: "AI is a broad field of computer science focused on creating intelligent machines",
					Metadata: followUpMetadata,
				}
				
				err = searchIndex.Index(ctx, followUpDoc)
				So(err, ShouldBeNil)
				
				// Link follow-up to previous turn
				turnSequence := NewEdge("edge_turn_sequence", turnID, followUpID, TemporalNext, 1.0)
				turnSequence.SetProperty("sequence_order", 1)
				
				err = graphStore.CreateEdge(ctx, turnSequence)
				So(err, ShouldBeNil)
				
				// Verify conversation flow
				conversationTurns, err := vectorStore.Search(ctx, conversationEmbedding, 10, 
					map[string]interface{}{"conversation_id": conversationID})
				So(err, ShouldBeNil)
				So(len(conversationTurns), ShouldEqual, 2)
				
				// Verify temporal relationship by finding paths
				paths, err := graphStore.FindPaths(ctx, turnID, followUpID, GraphTraversalOptions{MaxDepth: 2})
				So(err, ShouldBeNil)
				So(len(paths), ShouldBeGreaterThan, 0)
				So(paths[0].Nodes[0], ShouldEqual, turnID)
				So(paths[0].Nodes[1], ShouldEqual, followUpID)
			})
		})
	})
}

func BenchmarkStorageIntegration(b *testing.B) {
	tempDir := b.TempDir()
	
	vectorStore := NewFileVectorStore(filepath.Join(tempDir, "bench_vectors.json"))
	graphStore := NewFileGraphStore(filepath.Join(tempDir, "bench_graph.json"))
	searchIndex := NewFileSearchIndex(filepath.Join(tempDir, "bench_search.json"))
	
	ctx := context.Background()
	
	// Prepare test data
	embedding := []float32{0.1, 0.2, 0.3, 0.4}
	metadata := map[string]interface{}{"benchmark": true}
	content := "This is benchmark content for testing storage integration performance"
	
	b.Run("IntegratedStore", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			id := fmt.Sprintf("bench_%d", i)
			
			// Store in all backends
			vectorStore.Store(ctx, id, embedding, metadata)
			
			node := NewNode(id, EntityNode)
			node.SetProperty("benchmark", true)
			graphStore.CreateNode(ctx, node)
			
			doc := IndexDocument{ID: id, Content: content, Metadata: metadata}
			searchIndex.Index(ctx, doc)
		}
	})
	
	// Prepare data for search benchmark
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("search_bench_%d", i)
		
		vectorStore.Store(ctx, id, embedding, metadata)
		
		node := NewNode(id, EntityNode)
		graphStore.CreateNode(ctx, node)
		
		doc := IndexDocument{ID: id, Content: content, Metadata: metadata}
		searchIndex.Index(ctx, doc)
	}
	
	b.Run("IntegratedQuery", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Query all backends
			vectorStore.Search(ctx, embedding, 10, nil)
			graphStore.FindNodesByType(ctx, EntityNode, nil)
			searchIndex.Search(ctx, "benchmark", SearchIndexOptions{Limit: 10})
		}
	})
}