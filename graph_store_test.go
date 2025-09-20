package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGraphStoreInterface(t *testing.T) {
	Convey("GraphStore Interface Contract", t, func() {
		ctx := context.Background()
		store := NewMockGraphStore()
		
		Convey("Node operations", func() {
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
			
			Convey("Create node", func() {
				err := store.CreateNode(ctx, node)
				So(err, ShouldBeNil)
				
				Convey("Should be retrievable", func() {
					retrieved, err := store.GetNode(ctx, "test-node")
					So(err, ShouldBeNil)
					So(retrieved.ID, ShouldEqual, "test-node")
					So(retrieved.Type, ShouldEqual, EntityNode)
					So(retrieved.Properties["name"], ShouldEqual, "Test Entity")
				})
				
				Convey("Should not allow duplicate creation", func() {
					err := store.CreateNode(ctx, node)
					So(err, ShouldNotBeNil)
				})
			})
			
			Convey("Update node", func() {
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
			
			Convey("Delete node", func() {
				store.CreateNode(ctx, node)
				
				err := store.DeleteNode(ctx, "test-node")
				So(err, ShouldBeNil)
				
				_, err = store.GetNode(ctx, "test-node")
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("Edge operations", func() {
			// Create nodes first
			node1 := &Node{ID: "node1", Type: EntityNode, Properties: map[string]interface{}{}}
			node2 := &Node{ID: "node2", Type: EntityNode, Properties: map[string]interface{}{}}
			store.CreateNode(ctx, node1)
			store.CreateNode(ctx, node2)
			
			edge := &Edge{
				ID:     "test-edge",
				From:   "node1",
				To:     "node2",
				Type:   RelatedTo,
				Weight: 0.8,
				Properties: map[string]interface{}{
					"relationship": "knows",
				},
				CreatedAt: time.Now(),
			}
			
			Convey("Create edge", func() {
				err := store.CreateEdge(ctx, edge)
				So(err, ShouldBeNil)
				
				Convey("Should be retrievable", func() {
					retrieved, err := store.GetEdge(ctx, "test-edge")
					So(err, ShouldBeNil)
					So(retrieved.From, ShouldEqual, "node1")
					So(retrieved.To, ShouldEqual, "node2")
					So(retrieved.Type, ShouldEqual, RelatedTo)
					So(retrieved.Weight, ShouldEqual, 0.8)
				})
			})
			
			Convey("Should not create edge with non-existent nodes", func() {
				badEdge := &Edge{
					ID:   "bad-edge",
					From: "non-existent",
					To:   "node2",
					Type: RelatedTo,
				}
				
				err := store.CreateEdge(ctx, badEdge)
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("Graph traversal operations", func() {
			// Create a simple graph: A -> B -> C
			nodes := []*Node{
				{ID: "A", Type: EntityNode, Properties: map[string]interface{}{"name": "A"}},
				{ID: "B", Type: EntityNode, Properties: map[string]interface{}{"name": "B"}},
				{ID: "C", Type: EntityNode, Properties: map[string]interface{}{"name": "C"}},
			}
			
			for _, node := range nodes {
				store.CreateNode(ctx, node)
			}
			
			edges := []*Edge{
				{ID: "AB", From: "A", To: "B", Type: RelatedTo, Weight: 1.0},
				{ID: "BC", From: "B", To: "C", Type: RelatedTo, Weight: 1.0},
			}
			
			for _, edge := range edges {
				store.CreateEdge(ctx, edge)
			}
			
			Convey("Find paths", func() {
				options := GraphTraversalOptions{
					MaxDepth:   3,
					MaxResults: 10,
				}
				
				paths, err := store.FindPaths(ctx, "A", "C", options)
				So(err, ShouldBeNil)
				So(len(paths), ShouldBeGreaterThan, 0)
				
				// Should find path A -> B -> C
				path := paths[0]
				So(path.Nodes, ShouldResemble, []string{"A", "B", "C"})
				So(path.Edges, ShouldResemble, []string{"AB", "BC"})
			})
			
			Convey("Get neighbors", func() {
				options := GraphTraversalOptions{MaxDepth: 1}
				neighbors, err := store.GetNeighbors(ctx, "A", options)
				So(err, ShouldBeNil)
				So(len(neighbors), ShouldEqual, 1)
				So(neighbors[0].ID, ShouldEqual, "B")
			})
			
			Convey("Shortest path", func() {
				path, err := store.ShortestPath(ctx, "A", "C")
				So(err, ShouldBeNil)
				So(path.Nodes, ShouldResemble, []string{"A", "B", "C"})
			})
		})
		
		Convey("Graph algorithms", func() {
			// Create a more complex graph for algorithms
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
			}
			
			for _, edge := range edges {
				store.CreateEdge(ctx, edge)
			}
			
			Convey("PageRank", func() {
				options := PageRankOptions{
					Alpha:     0.85,
					MaxIter:   100,
					Tolerance: 0.001,
				}
				
				scores, err := store.PageRank(ctx, options)
				So(err, ShouldBeNil)
				So(len(scores), ShouldEqual, 4)
				
				// All nodes should have similar scores in this symmetric graph
				for nodeID, score := range scores {
					So(score, ShouldBeGreaterThan, 0)
					So(nodeID, ShouldBeIn, []string{"1", "2", "3", "4"})
				}
			})
			
			Convey("Community detection", func() {
				communities, err := store.CommunityDetection(ctx)
				So(err, ShouldBeNil)
				So(len(communities), ShouldBeGreaterThan, 0)
				
				// Should find at least one community containing all connected nodes
				totalNodes := 0
				for _, community := range communities {
					totalNodes += len(community.Nodes)
				}
				So(totalNodes, ShouldEqual, 4)
			})
		})
		
		Convey("Batch operations", func() {
			nodes := []*Node{
				{ID: "batch1", Type: EntityNode, Properties: map[string]interface{}{"batch": true}},
				{ID: "batch2", Type: EntityNode, Properties: map[string]interface{}{"batch": true}},
			}
			
			err := store.BatchCreateNodes(ctx, nodes)
			So(err, ShouldBeNil)
			
			for _, node := range nodes {
				retrieved, err := store.GetNode(ctx, node.ID)
				So(err, ShouldBeNil)
				So(retrieved.Properties["batch"], ShouldEqual, true)
			}
			
			edges := []*Edge{
				{ID: "batch-edge1", From: "batch1", To: "batch2", Type: RelatedTo},
			}
			
			err = store.BatchCreateEdges(ctx, edges)
			So(err, ShouldBeNil)
			
			retrieved, err := store.GetEdge(ctx, "batch-edge1")
			So(err, ShouldBeNil)
			So(retrieved.From, ShouldEqual, "batch1")
		})
		
		Convey("Query operations", func() {
			// Create test nodes of different types
			entityNode := &Node{ID: "entity1", Type: EntityNode, Properties: map[string]interface{}{"category": "person"}}
			claimNode := &Node{ID: "claim1", Type: ClaimNode, Properties: map[string]interface{}{"category": "fact"}}
			
			store.CreateNode(ctx, entityNode)
			store.CreateNode(ctx, claimNode)
			
			Convey("Find nodes by type", func() {
				entities, err := store.FindNodesByType(ctx, EntityNode, nil)
				So(err, ShouldBeNil)
				So(len(entities), ShouldBeGreaterThan, 0)
				
				// All results should be EntityNode type
				for _, node := range entities {
					So(node.Type, ShouldEqual, EntityNode)
				}
			})
			
			Convey("Find nodes by type with filters", func() {
				filters := map[string]interface{}{"category": "person"}
				entities, err := store.FindNodesByType(ctx, EntityNode, filters)
				So(err, ShouldBeNil)
				So(len(entities), ShouldEqual, 1)
				So(entities[0].ID, ShouldEqual, "entity1")
			})
		})
		
		Convey("Statistics", func() {
			nodeCount, err := store.NodeCount(ctx)
			So(err, ShouldBeNil)
			So(nodeCount, ShouldBeGreaterThanOrEqualTo, 0)
			
			edgeCount, err := store.EdgeCount(ctx)
			So(err, ShouldBeNil)
			So(edgeCount, ShouldBeGreaterThanOrEqualTo, 0)
		})
		
		Convey("Health and lifecycle", func() {
			err := store.Health(ctx)
			So(err, ShouldBeNil)
			
			err = store.Close()
			So(err, ShouldBeNil)
			
			// Operations should fail after close
			node := &Node{ID: "after-close", Type: EntityNode, Properties: map[string]interface{}{}}
			err = store.CreateNode(ctx, node)
			So(err, ShouldNotBeNil)
		})
	})
}

func BenchmarkGraphStore(b *testing.B) {
	ctx := context.Background()
	store := NewMockGraphStore()
	
	// Prepare test data
	_ = &Node{
		ID:         "bench-node",
		Type:       EntityNode,
		Properties: map[string]interface{}{"benchmark": true},
	}
	
	b.Run("CreateNode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			testNode := &Node{
				ID:         fmt.Sprintf("bench-node-%d", i),
				Type:       EntityNode,
				Properties: map[string]interface{}{"benchmark": true},
			}
			store.CreateNode(ctx, testNode)
		}
	})
	
	// Create nodes for edge benchmarks
	for i := 0; i < 100; i++ {
		testNode := &Node{
			ID:         fmt.Sprintf("edge-bench-node-%d", i),
			Type:       EntityNode,
			Properties: map[string]interface{}{},
		}
		store.CreateNode(ctx, testNode)
	}
	
	b.Run("CreateEdge", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			edge := &Edge{
				ID:     fmt.Sprintf("bench-edge-%d", i),
				From:   fmt.Sprintf("edge-bench-node-%d", i%100),
				To:     fmt.Sprintf("edge-bench-node-%d", (i+1)%100),
				Type:   RelatedTo,
				Weight: 1.0,
			}
			store.CreateEdge(ctx, edge)
		}
	})
	
	b.Run("GetNode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			store.GetNode(ctx, fmt.Sprintf("edge-bench-node-%d", i%100))
		}
	})
}