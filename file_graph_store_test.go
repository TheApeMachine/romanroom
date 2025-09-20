package main

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFileGraphStore(t *testing.T) {
	Convey("Given a FileGraphStore", t, func() {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "graph.json")
		store := NewFileGraphStore(filePath)
		ctx := context.Background()
		
		Convey("When creating a node", func() {
			node := NewNode("node1", EntityNode)
			node.SetProperty("name", "Test Entity")
			
			err := store.CreateNode(ctx, node)
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
			
			Convey("And the node should be retrievable", func() {
				retrieved, err := store.GetNode(ctx, "node1")
				So(err, ShouldBeNil)
				So(retrieved, ShouldNotBeNil)
				So(retrieved.ID, ShouldEqual, "node1")
				So(retrieved.Type, ShouldEqual, EntityNode)
				name, exists := retrieved.GetProperty("name")
				So(exists, ShouldBeTrue)
				So(name, ShouldEqual, "Test Entity")
			})
		})
		
		Convey("When creating an edge", func() {
			// Create nodes first
			node1 := NewNode("node1", EntityNode)
			node2 := NewNode("node2", ClaimNode)
			
			err := store.CreateNode(ctx, node1)
			So(err, ShouldBeNil)
			err = store.CreateNode(ctx, node2)
			So(err, ShouldBeNil)
			
			edge := NewEdge("edge1", "node1", "node2", RelatedTo, 1.0)
			edge.SetProperty("strength", "high")
			
			err = store.CreateEdge(ctx, edge)
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
			
			Convey("And the edge should be retrievable", func() {
				retrieved, err := store.GetEdge(ctx, "edge1")
				So(err, ShouldBeNil)
				So(retrieved, ShouldNotBeNil)
				So(retrieved.ID, ShouldEqual, "edge1")
				So(retrieved.From, ShouldEqual, "node1")
				So(retrieved.To, ShouldEqual, "node2")
				So(retrieved.Type, ShouldEqual, RelatedTo)
				So(retrieved.Weight, ShouldEqual, 1.0)
				strength, exists := retrieved.GetProperty("strength")
				So(exists, ShouldBeTrue)
				So(strength, ShouldEqual, "high")
			})
		})
		
		Convey("When getting neighbors", func() {
			// Create a small graph
			nodes := []*Node{
				NewNode("center", EntityNode),
				NewNode("neighbor1", ClaimNode),
				NewNode("neighbor2", EventNode),
				NewNode("distant", TaskNode),
			}
			
			for _, node := range nodes {
				err := store.CreateNode(ctx, node)
				So(err, ShouldBeNil)
			}
			
			edges := []*Edge{
				NewEdge("e1", "center", "neighbor1", RelatedTo, 1.0),
				NewEdge("e2", "center", "neighbor2", Supports, 0.8),
				NewEdge("e3", "neighbor1", "distant", PartOf, 0.5),
			}
			
			for _, edge := range edges {
				err := store.CreateEdge(ctx, edge)
				So(err, ShouldBeNil)
			}
			
			neighbors, err := store.GetNeighbors(ctx, "center", GraphTraversalOptions{})
			
			Convey("Then it should return direct neighbors", func() {
				So(err, ShouldBeNil)
				So(len(neighbors), ShouldEqual, 2)
				
				neighborIDs := make([]string, len(neighbors))
				for i, n := range neighbors {
					neighborIDs[i] = n.ID
				}
				So(neighborIDs, ShouldContain, "neighbor1")
				So(neighborIDs, ShouldContain, "neighbor2")
			})
			
			Convey("And filtering by node type should work", func() {
				options := GraphTraversalOptions{
					NodeTypes: []NodeType{ClaimNode},
				}
				neighbors, err := store.GetNeighbors(ctx, "center", options)
				So(err, ShouldBeNil)
				So(len(neighbors), ShouldEqual, 1)
				So(neighbors[0].ID, ShouldEqual, "neighbor1")
				So(neighbors[0].Type, ShouldEqual, ClaimNode)
			})
		})
		
		Convey("When finding paths", func() {
			// Create a path: A -> B -> C
			nodes := []*Node{
				NewNode("A", EntityNode),
				NewNode("B", ClaimNode),
				NewNode("C", EventNode),
			}
			
			for _, node := range nodes {
				err := store.CreateNode(ctx, node)
				So(err, ShouldBeNil)
			}
			
			edges := []*Edge{
				NewEdge("AB", "A", "B", RelatedTo, 1.0),
				NewEdge("BC", "B", "C", Supports, 2.0),
			}
			
			for _, edge := range edges {
				err := store.CreateEdge(ctx, edge)
				So(err, ShouldBeNil)
			}
			
			paths, err := store.FindPaths(ctx, "A", "C", GraphTraversalOptions{MaxDepth: 5})
			
			Convey("Then it should find the path", func() {
				So(err, ShouldBeNil)
				So(len(paths), ShouldBeGreaterThan, 0)
				
				path := paths[0]
				So(path.Nodes, ShouldResemble, []string{"A", "B", "C"})
				So(path.Edges, ShouldResemble, []string{"AB", "BC"})
				So(path.Cost, ShouldEqual, 3.0) // 1.0 + 2.0
			})
		})
		
		Convey("When computing PageRank", func() {
			// Create a simple graph for PageRank
			nodes := []*Node{
				NewNode("page1", EntityNode),
				NewNode("page2", EntityNode),
				NewNode("page3", EntityNode),
			}
			
			for _, node := range nodes {
				err := store.CreateNode(ctx, node)
				So(err, ShouldBeNil)
			}
			
			edges := []*Edge{
				NewEdge("12", "page1", "page2", RelatedTo, 1.0),
				NewEdge("13", "page1", "page3", RelatedTo, 1.0),
				NewEdge("23", "page2", "page3", RelatedTo, 1.0),
			}
			
			for _, edge := range edges {
				err := store.CreateEdge(ctx, edge)
				So(err, ShouldBeNil)
			}
			
			options := PageRankOptions{
				Alpha:     0.85,
				MaxIter:   100,
				Tolerance: 0.001,
			}
			
			scores, err := store.PageRank(ctx, options)
			
			Convey("Then it should compute PageRank scores", func() {
				So(err, ShouldBeNil)
				So(len(scores), ShouldEqual, 3)
				
				// All scores should be positive
				for nodeID, score := range scores {
					So(score, ShouldBeGreaterThan, 0)
					So(nodeID, ShouldBeIn, "page1", "page2", "page3")
				}
				
				// Sum of scores should be positive and reasonable
				totalScore := 0.0
				for _, score := range scores {
					totalScore += score
				}
				So(totalScore, ShouldBeGreaterThan, 0.5)
				So(totalScore, ShouldBeLessThan, 10.0)
			})
		})
		
		Convey("When performing community detection", func() {
			// Create nodes of different types
			nodes := []*Node{
				NewNode("entity1", EntityNode),
				NewNode("entity2", EntityNode),
				NewNode("claim1", ClaimNode),
				NewNode("claim2", ClaimNode),
				NewNode("event1", EventNode),
			}
			
			for _, node := range nodes {
				err := store.CreateNode(ctx, node)
				So(err, ShouldBeNil)
			}
			
			communities, err := store.CommunityDetection(ctx)
			
			Convey("Then it should detect communities", func() {
				So(err, ShouldBeNil)
				So(len(communities), ShouldEqual, 3) // 3 different node types
				
				// Check that communities are grouped by type
				for _, community := range communities {
					So(len(community.Nodes), ShouldBeGreaterThan, 0)
					So(community.ID, ShouldStartWith, "community_")
					So(community.Score, ShouldEqual, 1.0)
				}
			})
		})
		
		Convey("When finding nodes by type", func() {
			// Create nodes of different types
			nodes := []*Node{
				NewNode("entity1", EntityNode),
				NewNode("entity2", EntityNode),
				NewNode("claim1", ClaimNode),
			}
			
			nodes[0].SetProperty("category", "A")
			nodes[1].SetProperty("category", "B")
			nodes[2].SetProperty("category", "A")
			
			for _, node := range nodes {
				err := store.CreateNode(ctx, node)
				So(err, ShouldBeNil)
			}
			
			entityNodes, err := store.FindNodesByType(ctx, EntityNode, nil)
			
			Convey("Then it should find nodes of the specified type", func() {
				So(err, ShouldBeNil)
				So(len(entityNodes), ShouldEqual, 2)
				
				for _, node := range entityNodes {
					So(node.Type, ShouldEqual, EntityNode)
				}
			})
			
			Convey("And filtering should work", func() {
				filters := map[string]interface{}{"category": "A"}
				filteredNodes, err := store.FindNodesByType(ctx, EntityNode, filters)
				So(err, ShouldBeNil)
				So(len(filteredNodes), ShouldEqual, 1)
				So(filteredNodes[0].ID, ShouldEqual, "entity1")
			})
		})
		
		Convey("When finding edges by type", func() {
			// Create nodes and edges
			node1 := NewNode("node1", EntityNode)
			node2 := NewNode("node2", ClaimNode)
			node3 := NewNode("node3", EventNode)
			
			for _, node := range []*Node{node1, node2, node3} {
				err := store.CreateNode(ctx, node)
				So(err, ShouldBeNil)
			}
			
			edges := []*Edge{
				NewEdge("edge1", "node1", "node2", RelatedTo, 1.0),
				NewEdge("edge2", "node2", "node3", Supports, 1.0),
				NewEdge("edge3", "node1", "node3", RelatedTo, 1.0),
			}
			
			edges[0].SetProperty("strength", "high")
			edges[2].SetProperty("strength", "low")
			
			for _, edge := range edges {
				err := store.CreateEdge(ctx, edge)
				So(err, ShouldBeNil)
			}
			
			relatedEdges, err := store.FindEdgesByType(ctx, RelatedTo, nil)
			
			Convey("Then it should find edges of the specified type", func() {
				So(err, ShouldBeNil)
				So(len(relatedEdges), ShouldEqual, 2)
				
				for _, edge := range relatedEdges {
					So(edge.Type, ShouldEqual, RelatedTo)
				}
			})
			
			Convey("And filtering should work", func() {
				filters := map[string]interface{}{"strength": "high"}
				filteredEdges, err := store.FindEdgesByType(ctx, RelatedTo, filters)
				So(err, ShouldBeNil)
				So(len(filteredEdges), ShouldEqual, 1)
				So(filteredEdges[0].ID, ShouldEqual, "edge1")
			})
		})
		
		Convey("When updating nodes and edges", func() {
			node := NewNode("update_node", EntityNode)
			node.SetProperty("version", 1)
			
			err := store.CreateNode(ctx, node)
			So(err, ShouldBeNil)
			
			// Update node
			node.SetProperty("version", 2)
			node.SetProperty("updated", true)
			err = store.UpdateNode(ctx, node)
			
			Convey("Then node update should succeed", func() {
				So(err, ShouldBeNil)
				
				retrieved, err := store.GetNode(ctx, "update_node")
				So(err, ShouldBeNil)
				version, _ := retrieved.GetProperty("version")
				So(version, ShouldEqual, 2)
				updated, _ := retrieved.GetProperty("updated")
				So(updated, ShouldEqual, true)
			})
		})
		
		Convey("When deleting nodes and edges", func() {
			// Create nodes and edge
			node1 := NewNode("delete1", EntityNode)
			node2 := NewNode("delete2", ClaimNode)
			
			err := store.CreateNode(ctx, node1)
			So(err, ShouldBeNil)
			err = store.CreateNode(ctx, node2)
			So(err, ShouldBeNil)
			
			edge := NewEdge("delete_edge", "delete1", "delete2", RelatedTo, 1.0)
			err = store.CreateEdge(ctx, edge)
			So(err, ShouldBeNil)
			
			// Delete node (should also delete connected edges)
			err = store.DeleteNode(ctx, "delete1")
			
			Convey("Then node deletion should succeed", func() {
				So(err, ShouldBeNil)
				
				_, err := store.GetNode(ctx, "delete1")
				So(err, ShouldNotBeNil)
				
				// Connected edge should also be deleted
				_, err = store.GetEdge(ctx, "delete_edge")
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("When getting counts", func() {
			// Create some nodes and edges
			for i := 0; i < 3; i++ {
				node := NewNode(fmt.Sprintf("count_node_%d", i), EntityNode)
				err := store.CreateNode(ctx, node)
				So(err, ShouldBeNil)
			}
			
			nodeCount, err := store.NodeCount(ctx)
			
			Convey("Then node count should be correct", func() {
				So(err, ShouldBeNil)
				So(nodeCount, ShouldEqual, 3)
			})
			
			edgeCount, err := store.EdgeCount(ctx)
			
			Convey("And edge count should be correct", func() {
				So(err, ShouldBeNil)
				So(edgeCount, ShouldEqual, 0) // No edges created in this test
			})
		})
		
		Convey("When checking health", func() {
			err := store.Health(ctx)
			
			Convey("Then it should be healthy", func() {
				So(err, ShouldBeNil)
			})
		})
		
		Convey("When closing the store", func() {
			err := store.Close()
			
			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
			
			Convey("And subsequent operations should fail", func() {
				node := NewNode("test", EntityNode)
				err := store.CreateNode(ctx, node)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "closed")
			})
		})
		
		Convey("When loading and saving", func() {
			// Create some data
			node := NewNode("persist_node", EntityNode)
			node.SetProperty("persistent", true)
			err := store.CreateNode(ctx, node)
			So(err, ShouldBeNil)
			
			// Save explicitly
			err = store.Save()
			So(err, ShouldBeNil)
			
			// Create new store instance and load
			newStore := NewFileGraphStore(filePath)
			err = newStore.Load()
			
			Convey("Then data should persist", func() {
				So(err, ShouldBeNil)
				
				retrieved, err := newStore.GetNode(ctx, "persist_node")
				So(err, ShouldBeNil)
				So(retrieved.ID, ShouldEqual, "persist_node")
				So(retrieved.Type, ShouldEqual, EntityNode)
				persistent, exists := retrieved.GetProperty("persistent")
				So(exists, ShouldBeTrue)
				So(persistent, ShouldEqual, true)
			})
		})
	})
}

func TestFileGraphStoreEdgeCases(t *testing.T) {
	Convey("Given a FileGraphStore with edge cases", t, func() {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "graph_edge.json")
		store := NewFileGraphStore(filePath)
		ctx := context.Background()
		
		Convey("When creating an edge without nodes", func() {
			edge := NewEdge("orphan_edge", "nonexistent1", "nonexistent2", RelatedTo, 1.0)
			err := store.CreateEdge(ctx, edge)
			
			Convey("Then it should fail", func() {
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("When getting a non-existent node", func() {
			_, err := store.GetNode(ctx, "nonexistent")
			
			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("When getting a non-existent edge", func() {
			_, err := store.GetEdge(ctx, "nonexistent")
			
			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
		
		Convey("When loading from non-existent file", func() {
			nonExistentStore := NewFileGraphStore(filepath.Join(tempDir, "nonexistent.json"))
			err := nonExistentStore.Load()
			
			Convey("Then it should succeed with empty graph", func() {
				So(err, ShouldBeNil)
				nodeCount, err := nonExistentStore.NodeCount(ctx)
				So(err, ShouldBeNil)
				So(nodeCount, ShouldEqual, 0)
			})
		})
		
		Convey("When creating invalid nodes", func() {
			invalidNode := &Node{ID: "", Type: EntityNode}
			err := store.CreateNode(ctx, invalidNode)
			
			Convey("Then it should fail validation", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "invalid node")
			})
		})
		
		Convey("When creating invalid edges", func() {
			invalidEdge := &Edge{ID: "test", From: "", To: "node2", Type: RelatedTo}
			err := store.CreateEdge(ctx, invalidEdge)
			
			Convey("Then it should fail validation", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "invalid edge")
			})
		})
	})
}

func BenchmarkFileGraphStore(b *testing.B) {
	tempDir := b.TempDir()
	filePath := filepath.Join(tempDir, "bench_graph.json")
	store := NewFileGraphStore(filePath)
	ctx := context.Background()
	
	// Prepare test data
	b.Run("CreateNode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			node := NewNode(fmt.Sprintf("bench_node_%d", i), EntityNode)
			store.CreateNode(ctx, node)
		}
	})
	
	// Create nodes for edge benchmark
	for i := 0; i < 1000; i++ {
		node := NewNode(fmt.Sprintf("edge_bench_node_%d", i), EntityNode)
		store.CreateNode(ctx, node)
	}
	
	b.Run("CreateEdge", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			from := fmt.Sprintf("edge_bench_node_%d", i%1000)
			to := fmt.Sprintf("edge_bench_node_%d", (i+1)%1000)
			edge := NewEdge(fmt.Sprintf("bench_edge_%d", i), from, to, RelatedTo, 1.0)
			store.CreateEdge(ctx, edge)
		}
	})
	
	b.Run("GetNode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			nodeID := fmt.Sprintf("edge_bench_node_%d", i%1000)
			store.GetNode(ctx, nodeID)
		}
	})
}