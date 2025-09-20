package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMockGraphStore(t *testing.T) {
	Convey("MockGraphStore Implementation", t, func() {
		ctx := context.Background()
		store := NewMockGraphStore()
		
		Convey("Should implement GraphStore interface", func() {
			var _ GraphStore = store
		})
		
		Convey("Node lifecycle operations", func() {
			node := &Node{
				ID:   "test-node",
				Type: EntityNode,
				Properties: map[string]interface{}{
					"name": "Test Entity",
					"type": "person",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			
			Convey("Create and retrieve node", func() {
				err := store.CreateNode(ctx, node)
				So(err, ShouldBeNil)
				
				retrieved, err := store.GetNode(ctx, "test-node")
				So(err, ShouldBeNil)
				So(retrieved.ID, ShouldEqual, "test-node")
				So(retrieved.Type, ShouldEqual, EntityNode)
				So(retrieved.Properties["name"], ShouldEqual, "Test Entity")
			})
			
			Convey("Prevent duplicate node creation", func() {
				store.CreateNode(ctx, node)
				err := store.CreateNode(ctx, node)
				So(err, ShouldNotBeNil)
			})
			
			Convey("Update existing node", func() {
				store.CreateNode(ctx, node)
				
				updatedNode := &Node{
					ID:   "test-node",
					Type: EntityNode,
					Properties: map[string]interface{}{
						"name":    "Updated Entity",
						"version": 2,
					},
					UpdatedAt: time.Now(),
				}
				
				err := store.UpdateNode(ctx, updatedNode)
				So(err, ShouldBeNil)
				
				retrieved, err := store.GetNode(ctx, "test-node")
				So(err, ShouldBeNil)
				So(retrieved.Properties["name"], ShouldEqual, "Updated Entity")
				So(retrieved.Properties["version"], ShouldEqual, 2)
			})
			
			Convey("Delete node and cleanup edges", func() {
				// Create nodes and edges
				node1 := &Node{ID: "node1", Type: EntityNode, Properties: map[string]interface{}{}}
				node2 := &Node{ID: "node2", Type: EntityNode, Properties: map[string]interface{}{}}
				store.CreateNode(ctx, node1)
				store.CreateNode(ctx, node2)
				
				edge := &Edge{ID: "edge1", From: "node1", To: "node2", Type: RelatedTo}
				store.CreateEdge(ctx, edge)
				
				// Delete node1 - should also remove the edge
				err := store.DeleteNode(ctx, "node1")
				So(err, ShouldBeNil)
				
				// Node should be gone
				_, err = store.GetNode(ctx, "node1")
				So(err, ShouldNotBeNil)
				
				// Edge should be gone
				_, err = store.GetEdge(ctx, "edge1")
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("Edge lifecycle operations", func() {
			// Setup nodes
			node1 := &Node{ID: "n1", Type: EntityNode, Properties: map[string]interface{}{}}
			node2 := &Node{ID: "n2", Type: EntityNode, Properties: map[string]interface{}{}}
			store.CreateNode(ctx, node1)
			store.CreateNode(ctx, node2)
			
			edge := &Edge{
				ID:     "test-edge",
				From:   "n1",
				To:     "n2",
				Type:   RelatedTo,
				Weight: 0.8,
				Properties: map[string]interface{}{
					"relationship": "knows",
				},
				CreatedAt: time.Now(),
			}
			
			Convey("Create and retrieve edge", func() {
				err := store.CreateEdge(ctx, edge)
				So(err, ShouldBeNil)
				
				retrieved, err := store.GetEdge(ctx, "test-edge")
				So(err, ShouldBeNil)
				So(retrieved.From, ShouldEqual, "n1")
				So(retrieved.To, ShouldEqual, "n2")
				So(retrieved.Type, ShouldEqual, RelatedTo)
				So(retrieved.Weight, ShouldEqual, 0.8)
			})
			
			Convey("Prevent edge creation with non-existent nodes", func() {
				badEdge := &Edge{
					ID:   "bad-edge",
					From: "non-existent",
					To:   "n2",
					Type: RelatedTo,
				}
				
				err := store.CreateEdge(ctx, badEdge)
				So(err, ShouldNotBeNil)
			})
			
			Convey("Update existing edge", func() {
				store.CreateEdge(ctx, edge)
				
				updatedEdge := &Edge{
					ID:     "test-edge",
					From:   "n1",
					To:     "n2",
					Type:   Supports,
					Weight: 0.9,
					Properties: map[string]interface{}{
						"relationship": "supports",
						"strength":     "high",
					},
				}
				
				err := store.UpdateEdge(ctx, updatedEdge)
				So(err, ShouldBeNil)
				
				retrieved, err := store.GetEdge(ctx, "test-edge")
				So(err, ShouldBeNil)
				So(retrieved.Type, ShouldEqual, Supports)
				So(retrieved.Weight, ShouldEqual, 0.9)
				So(retrieved.Properties["strength"], ShouldEqual, "high")
			})
			
			Convey("Delete edge", func() {
				store.CreateEdge(ctx, edge)
				
				err := store.DeleteEdge(ctx, "test-edge")
				So(err, ShouldBeNil)
				
				_, err = store.GetEdge(ctx, "test-edge")
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("Graph traversal operations", func() {
			// Create a test graph: A -> B -> C -> D
			//                      A -> C (shortcut)
			nodes := []*Node{
				{ID: "A", Type: EntityNode, Properties: map[string]interface{}{"name": "A"}},
				{ID: "B", Type: EntityNode, Properties: map[string]interface{}{"name": "B"}},
				{ID: "C", Type: EntityNode, Properties: map[string]interface{}{"name": "C"}},
				{ID: "D", Type: EntityNode, Properties: map[string]interface{}{"name": "D"}},
			}
			
			for _, node := range nodes {
				store.CreateNode(ctx, node)
			}
			
			edges := []*Edge{
				{ID: "AB", From: "A", To: "B", Type: RelatedTo, Weight: 1.0},
				{ID: "BC", From: "B", To: "C", Type: RelatedTo, Weight: 1.0},
				{ID: "CD", From: "C", To: "D", Type: RelatedTo, Weight: 1.0},
				{ID: "AC", From: "A", To: "C", Type: RelatedTo, Weight: 0.5}, // Shortcut
			}
			
			for _, edge := range edges {
				store.CreateEdge(ctx, edge)
			}
			
			Convey("Find paths between nodes", func() {
				options := GraphTraversalOptions{
					MaxDepth:   5,
					MaxResults: 10,
				}
				
				paths, err := store.FindPaths(ctx, "A", "D", options)
				So(err, ShouldBeNil)
				So(len(paths), ShouldBeGreaterThan, 0)
				
				// Should find at least the path A -> B -> C -> D
				foundLongPath := false
				for _, path := range paths {
					if len(path.Nodes) == 4 && path.Nodes[0] == "A" && path.Nodes[3] == "D" {
						foundLongPath = true
						So(path.Edges, ShouldResemble, []string{"AB", "BC", "CD"})
						break
					}
				}
				So(foundLongPath, ShouldBeTrue)
			})
			
			Convey("Get neighbors", func() {
				options := GraphTraversalOptions{MaxDepth: 1}
				neighbors, err := store.GetNeighbors(ctx, "A", options)
				So(err, ShouldBeNil)
				So(len(neighbors), ShouldEqual, 2) // B and C
				
				neighborIDs := make([]string, len(neighbors))
				for i, neighbor := range neighbors {
					neighborIDs[i] = neighbor.ID
				}
				So(neighborIDs, ShouldContain, "B")
				So(neighborIDs, ShouldContain, "C")
			})
			
			Convey("Find shortest path", func() {
				path, err := store.ShortestPath(ctx, "A", "D")
				So(err, ShouldBeNil)
				So(path, ShouldNotBeNil)
				So(path.Nodes[0], ShouldEqual, "A")
				So(path.Nodes[len(path.Nodes)-1], ShouldEqual, "D")
			})
			
			Convey("Handle path not found", func() {
				// Create isolated node
				isolatedNode := &Node{ID: "isolated", Type: EntityNode, Properties: map[string]interface{}{}}
				store.CreateNode(ctx, isolatedNode)
				
				_, err := store.ShortestPath(ctx, "A", "isolated")
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("Graph algorithms", func() {
			// Create a more complex graph for algorithm testing
			nodes := []*Node{
				{ID: "1", Type: EntityNode, Properties: map[string]interface{}{}},
				{ID: "2", Type: EntityNode, Properties: map[string]interface{}{}},
				{ID: "3", Type: EntityNode, Properties: map[string]interface{}{}},
				{ID: "4", Type: EntityNode, Properties: map[string]interface{}{}},
			}
			
			for _, node := range nodes {
				store.CreateNode(ctx, node)
			}
			
			edges := []*Edge{
				{ID: "12", From: "1", To: "2", Type: RelatedTo, Weight: 1.0},
				{ID: "23", From: "2", To: "3", Type: RelatedTo, Weight: 1.0},
				{ID: "34", From: "3", To: "4", Type: RelatedTo, Weight: 1.0},
				{ID: "41", From: "4", To: "1", Type: RelatedTo, Weight: 1.0},
				{ID: "13", From: "1", To: "3", Type: RelatedTo, Weight: 0.5},
			}
			
			for _, edge := range edges {
				store.CreateEdge(ctx, edge)
			}
			
			Convey("PageRank algorithm", func() {
				options := PageRankOptions{
					Alpha:     0.85,
					MaxIter:   100,
					Tolerance: 0.001,
				}
				
				scores, err := store.PageRank(ctx, options)
				So(err, ShouldBeNil)
				So(len(scores), ShouldEqual, 4)
				
				// All nodes should have positive scores
				for nodeID, score := range scores {
					So(score, ShouldBeGreaterThan, 0)
					So(nodeID, ShouldBeIn, []string{"1", "2", "3", "4"})
				}
				
				// In this symmetric graph, scores should be relatively similar
				totalScore := 0.0
				for _, score := range scores {
					totalScore += score
				}
				avgScore := totalScore / 4
				
				for _, score := range scores {
					So(score, ShouldBeBetween, avgScore*0.5, avgScore*1.5)
				}
			})
			
			Convey("Community detection", func() {
				communities, err := store.CommunityDetection(ctx)
				So(err, ShouldBeNil)
				So(len(communities), ShouldBeGreaterThan, 0)
				
				// All nodes should be in some community
				allNodes := make(map[string]bool)
				for _, community := range communities {
					So(len(community.Nodes), ShouldBeGreaterThan, 0)
					So(community.Score, ShouldBeGreaterThan, 0)
					
					for _, nodeID := range community.Nodes {
						allNodes[nodeID] = true
					}
				}
				
				So(len(allNodes), ShouldEqual, 4)
				So(allNodes["1"], ShouldBeTrue)
				So(allNodes["2"], ShouldBeTrue)
				So(allNodes["3"], ShouldBeTrue)
				So(allNodes["4"], ShouldBeTrue)
			})
		})
		
		Convey("Batch operations", func() {
			nodes := []*Node{
				{ID: "batch1", Type: EntityNode, Properties: map[string]interface{}{"batch": true}},
				{ID: "batch2", Type: EntityNode, Properties: map[string]interface{}{"batch": true}},
				{ID: "batch3", Type: ClaimNode, Properties: map[string]interface{}{"batch": true}},
			}
			
			err := store.BatchCreateNodes(ctx, nodes)
			So(err, ShouldBeNil)
			
			// Verify all nodes were created
			for _, node := range nodes {
				retrieved, err := store.GetNode(ctx, node.ID)
				So(err, ShouldBeNil)
				So(retrieved.Type, ShouldEqual, node.Type)
				So(retrieved.Properties["batch"], ShouldEqual, true)
			}
			
			edges := []*Edge{
				{ID: "batch-edge1", From: "batch1", To: "batch2", Type: RelatedTo},
				{ID: "batch-edge2", From: "batch2", To: "batch3", Type: Supports},
			}
			
			err = store.BatchCreateEdges(ctx, edges)
			So(err, ShouldBeNil)
			
			// Verify all edges were created
			for _, edge := range edges {
				retrieved, err := store.GetEdge(ctx, edge.ID)
				So(err, ShouldBeNil)
				So(retrieved.From, ShouldEqual, edge.From)
				So(retrieved.To, ShouldEqual, edge.To)
				So(retrieved.Type, ShouldEqual, edge.Type)
			}
		})
		
		// Create test nodes and edges for query operations
		nodes := []*Node{
			{ID: "entity1", Type: EntityNode, Properties: map[string]interface{}{"category": "person", "name": "Alice"}},
			{ID: "entity2", Type: EntityNode, Properties: map[string]interface{}{"category": "place", "name": "Paris"}},
			{ID: "claim1", Type: ClaimNode, Properties: map[string]interface{}{"category": "fact", "statement": "Alice lives in Paris"}},
		}
		
		for _, node := range nodes {
			store.CreateNode(ctx, node)
		}
		
		edges := []*Edge{
			{ID: "rel1", From: "entity1", To: "entity2", Type: RelatedTo, Properties: map[string]interface{}{"type": "lives_in"}},
			{ID: "sup1", From: "claim1", To: "entity1", Type: Supports, Properties: map[string]interface{}{"confidence": 0.9}},
		}
		
		for _, edge := range edges {
			store.CreateEdge(ctx, edge)
		}
		
		Convey("Query operations", func() {
			
			Convey("Find nodes by type", func() {
				entities, err := store.FindNodesByType(ctx, EntityNode, nil)
				So(err, ShouldBeNil)
				So(len(entities), ShouldEqual, 2)
				
				for _, entity := range entities {
					So(entity.Type, ShouldEqual, EntityNode)
				}
			})
			
			Convey("Find nodes by type with filters", func() {
				filters := map[string]interface{}{"category": "person"}
				entities, err := store.FindNodesByType(ctx, EntityNode, filters)
				So(err, ShouldBeNil)
				So(len(entities), ShouldEqual, 1)
				So(entities[0].ID, ShouldEqual, "entity1")
				So(entities[0].Properties["name"], ShouldEqual, "Alice")
			})
			
			Convey("Find edges by type", func() {
				relatedEdges, err := store.FindEdgesByType(ctx, RelatedTo, nil)
				So(err, ShouldBeNil)
				So(len(relatedEdges), ShouldEqual, 1)
				So(relatedEdges[0].ID, ShouldEqual, "rel1")
			})
			
			Convey("Find edges by type with filters", func() {
				// Debug: check if the edge exists by ID
				supEdge, err := store.GetEdge(ctx, "sup1")
				So(err, ShouldBeNil)
				So(supEdge.Type, ShouldEqual, Supports)
				
				// Debug: check all edges
				allEdges, err := store.FindEdgesByType(ctx, RelatedTo, nil)
				So(err, ShouldBeNil)
				So(len(allEdges), ShouldEqual, 1)
				
				// First verify the edge exists
				allSupportEdges, err := store.FindEdgesByType(ctx, Supports, nil)
				So(err, ShouldBeNil)
				So(len(allSupportEdges), ShouldEqual, 1)
				So(allSupportEdges[0].Properties["confidence"], ShouldEqual, 0.9)
				
				filters := map[string]interface{}{"confidence": 0.9}
				supportEdges, err := store.FindEdgesByType(ctx, Supports, filters)
				So(err, ShouldBeNil)
				So(len(supportEdges), ShouldEqual, 1)
				So(supportEdges[0].ID, ShouldEqual, "sup1")
			})
		})
		
		Convey("Statistics and health", func() {
			// Add some test data
			for i := 0; i < 5; i++ {
				node := &Node{
					ID:         fmt.Sprintf("stats-node-%d", i),
					Type:       EntityNode,
					Properties: map[string]interface{}{},
				}
				store.CreateNode(ctx, node)
			}
			
			nodeCount, err := store.NodeCount(ctx)
			So(err, ShouldBeNil)
			So(nodeCount, ShouldBeGreaterThanOrEqualTo, 5)
			
			edgeCount, err := store.EdgeCount(ctx)
			So(err, ShouldBeNil)
			So(edgeCount, ShouldBeGreaterThanOrEqualTo, 0)
			
			err = store.Health(ctx)
			So(err, ShouldBeNil)
		})
		
		Convey("Health status management", func() {
			// Initially healthy
			err := store.Health(ctx)
			So(err, ShouldBeNil)
			
			// Set unhealthy
			store.SetHealthy(false)
			err = store.Health(ctx)
			So(err, ShouldNotBeNil)
			
			// Set healthy again
			store.SetHealthy(true)
			err = store.Health(ctx)
			So(err, ShouldBeNil)
		})
		
		Convey("Close functionality", func() {
			err := store.Close()
			So(err, ShouldBeNil)
			
			// All operations should fail after close
			node := &Node{ID: "after-close", Type: EntityNode, Properties: map[string]interface{}{}}
			err = store.CreateNode(ctx, node)
			So(err, ShouldNotBeNil)
			
			_, err = store.GetNode(ctx, "any-id")
			So(err, ShouldNotBeNil)
			
			err = store.Health(ctx)
			So(err, ShouldNotBeNil)
		})
	})
}

func BenchmarkMockGraphStore(b *testing.B) {
	ctx := context.Background()
	store := NewMockGraphStore()
	
	// Pre-populate for benchmarks
	for i := 0; i < 100; i++ {
		node := &Node{
			ID:         fmt.Sprintf("bench-node-%d", i),
			Type:       EntityNode,
			Properties: map[string]interface{}{"benchmark": true},
		}
		store.CreateNode(ctx, node)
	}
	
	b.Run("CreateNode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			node := &Node{
				ID:         fmt.Sprintf("create-bench-%d", i),
				Type:       EntityNode,
				Properties: map[string]interface{}{},
			}
			store.CreateNode(ctx, node)
		}
	})
	
	b.Run("GetNode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			store.GetNode(ctx, fmt.Sprintf("bench-node-%d", i%100))
		}
	})
	
	// Create edges for edge benchmarks
	for i := 0; i < 99; i++ {
		edge := &Edge{
			ID:     fmt.Sprintf("bench-edge-%d", i),
			From:   fmt.Sprintf("bench-node-%d", i),
			To:     fmt.Sprintf("bench-node-%d", i+1),
			Type:   RelatedTo,
			Weight: 1.0,
		}
		store.CreateEdge(ctx, edge)
	}
	
	b.Run("CreateEdge", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			edge := &Edge{
				ID:     fmt.Sprintf("edge-bench-%d", i),
				From:   fmt.Sprintf("bench-node-%d", i%100),
				To:     fmt.Sprintf("bench-node-%d", (i+1)%100),
				Type:   RelatedTo,
				Weight: 1.0,
			}
			store.CreateEdge(ctx, edge)
		}
	})
	
	b.Run("FindPaths", func(b *testing.B) {
		options := GraphTraversalOptions{MaxDepth: 3, MaxResults: 5}
		for i := 0; i < b.N; i++ {
			store.FindPaths(ctx, "bench-node-0", "bench-node-10", options)
		}
	})
}