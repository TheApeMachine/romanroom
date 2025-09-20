package main

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNode(t *testing.T) {
	Convey("Given a new node", t, func() {
		id := "test-node-1"
		nodeType := EntityNode
		
		Convey("When creating a new node", func() {
			node := NewNode(id, nodeType)
			
			Convey("Then it should have correct initial values", func() {
				So(node.ID, ShouldEqual, id)
				So(node.Type, ShouldEqual, nodeType)
				So(node.Properties, ShouldNotBeNil)
				So(len(node.Properties), ShouldEqual, 0)
				So(node.CreatedAt, ShouldHappenWithin, time.Second, time.Now())
				So(node.UpdatedAt, ShouldHappenWithin, time.Second, time.Now())
			})
		})
		
		Convey("When validating a valid node", func() {
			node := NewNode(id, nodeType)
			err := node.Validate()
			
			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})
		
		Convey("When validating an invalid node", func() {
			Convey("With empty ID", func() {
				node := NewNode("", nodeType)
				err := node.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "node ID cannot be empty")
			})
			
			Convey("With empty type", func() {
				node := NewNode(id, "")
				err := node.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "node type cannot be empty")
			})
			
			Convey("With invalid type", func() {
				node := NewNode(id, "INVALID_TYPE")
				err := node.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "invalid node type")
			})
		})
		
		Convey("When setting and getting properties", func() {
			node := NewNode(id, nodeType)
			key := "test-property"
			value := "test-value"
			initialUpdateTime := node.UpdatedAt
			
			time.Sleep(time.Millisecond) // Ensure time difference
			node.SetProperty(key, value)
			retrievedValue, exists := node.GetProperty(key)
			
			Convey("Then property should be stored and retrieved correctly", func() {
				So(exists, ShouldBeTrue)
				So(retrievedValue, ShouldEqual, value)
				So(node.UpdatedAt, ShouldHappenAfter, initialUpdateTime)
			})
			
			Convey("And getting non-existent property should return false", func() {
				_, exists := node.GetProperty("non-existent")
				So(exists, ShouldBeFalse)
			})
		})
		
		Convey("When setting embedding", func() {
			node := NewNode(id, nodeType)
			embedding := []float32{0.1, 0.2, 0.3, 0.4}
			initialUpdateTime := node.UpdatedAt
			
			time.Sleep(time.Millisecond) // Ensure time difference
			node.SetEmbedding(embedding)
			
			Convey("Then embedding should be set and UpdatedAt should be updated", func() {
				So(node.Embedding, ShouldResemble, embedding)
				So(node.UpdatedAt, ShouldHappenAfter, initialUpdateTime)
			})
		})
	})
}

func TestEdge(t *testing.T) {
	Convey("Given a new edge", t, func() {
		id := "test-edge-1"
		from := "node-1"
		to := "node-2"
		edgeType := RelatedTo
		weight := 0.8
		
		Convey("When creating a new edge", func() {
			edge := NewEdge(id, from, to, edgeType, weight)
			
			Convey("Then it should have correct initial values", func() {
				So(edge.ID, ShouldEqual, id)
				So(edge.From, ShouldEqual, from)
				So(edge.To, ShouldEqual, to)
				So(edge.Type, ShouldEqual, edgeType)
				So(edge.Weight, ShouldEqual, weight)
				So(edge.Properties, ShouldNotBeNil)
				So(len(edge.Properties), ShouldEqual, 0)
				So(edge.CreatedAt, ShouldHappenWithin, time.Second, time.Now())
			})
		})
		
		Convey("When validating a valid edge", func() {
			edge := NewEdge(id, from, to, edgeType, weight)
			err := edge.Validate()
			
			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})
		
		Convey("When validating an invalid edge", func() {
			Convey("With empty ID", func() {
				edge := NewEdge("", from, to, edgeType, weight)
				err := edge.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "edge ID cannot be empty")
			})
			
			Convey("With empty from node", func() {
				edge := NewEdge(id, "", to, edgeType, weight)
				err := edge.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "edge 'from' node cannot be empty")
			})
			
			Convey("With empty to node", func() {
				edge := NewEdge(id, from, "", edgeType, weight)
				err := edge.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "edge 'to' node cannot be empty")
			})
			
			Convey("With empty type", func() {
				edge := NewEdge(id, from, to, "", weight)
				err := edge.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "edge type cannot be empty")
			})
			
			Convey("With invalid type", func() {
				edge := NewEdge(id, from, to, "INVALID_TYPE", weight)
				err := edge.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "invalid edge type")
			})
			
			Convey("With negative weight", func() {
				edge := NewEdge(id, from, to, edgeType, -0.5)
				err := edge.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "edge weight cannot be negative")
			})
		})
		
		Convey("When setting and getting properties", func() {
			edge := NewEdge(id, from, to, edgeType, weight)
			key := "test-property"
			value := "test-value"
			
			edge.SetProperty(key, value)
			retrievedValue, exists := edge.GetProperty(key)
			
			Convey("Then property should be stored and retrieved correctly", func() {
				So(exists, ShouldBeTrue)
				So(retrievedValue, ShouldEqual, value)
			})
			
			Convey("And getting non-existent property should return false", func() {
				_, exists := edge.GetProperty("non-existent")
				So(exists, ShouldBeFalse)
			})
		})
	})
}

func TestNodeTypeValidation(t *testing.T) {
	Convey("Given node type validation", t, func() {
		validTypes := []NodeType{
			EntityNode, ClaimNode, EventNode, TaskNode, ConversationNode, SourceNode,
		}
		
		Convey("When checking valid node types", func() {
			for _, nodeType := range validTypes {
				Convey("Type "+string(nodeType)+" should be valid", func() {
					So(isValidNodeType(nodeType), ShouldBeTrue)
				})
			}
		})
		
		Convey("When checking invalid node type", func() {
			invalidType := NodeType("INVALID")
			So(isValidNodeType(invalidType), ShouldBeFalse)
		})
	})
}

func TestEdgeTypeValidation(t *testing.T) {
	Convey("Given edge type validation", t, func() {
		validTypes := []EdgeType{
			RelatedTo, PartOf, Supports, Refutes, TemporalNext, CausedBy,
		}
		
		Convey("When checking valid edge types", func() {
			for _, edgeType := range validTypes {
				Convey("Type "+string(edgeType)+" should be valid", func() {
					So(isValidEdgeType(edgeType), ShouldBeTrue)
				})
			}
		})
		
		Convey("When checking invalid edge type", func() {
			invalidType := EdgeType("INVALID")
			So(isValidEdgeType(invalidType), ShouldBeFalse)
		})
	})
}

func BenchmarkNodeCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewNode("test-id", EntityNode)
	}
}

func BenchmarkNodeValidation(b *testing.B) {
	node := NewNode("test-id", EntityNode)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		node.Validate()
	}
}

func BenchmarkEdgeCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewEdge("test-id", "from", "to", RelatedTo, 0.5)
	}
}

func BenchmarkEdgeValidation(b *testing.B) {
	edge := NewEdge("test-id", "from", "to", RelatedTo, 0.5)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		edge.Validate()
	}
}

func BenchmarkNodePropertyOperations(b *testing.B) {
	node := NewNode("test-id", EntityNode)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		node.SetProperty("key", "value")
		node.GetProperty("key")
	}
}

func BenchmarkEdgePropertyOperations(b *testing.B) {
	edge := NewEdge("test-id", "from", "to", RelatedTo, 0.5)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		edge.SetProperty("key", "value")
		edge.GetProperty("key")
	}
}