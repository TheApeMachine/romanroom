package main

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRecallResponse(t *testing.T) {
	Convey("Given RecallResponse", t, func() {
		Convey("When creating new RecallResponse", func() {
			response := NewRecallResponse()

			Convey("Then it should have correct default values", func() {
				So(response.Evidence, ShouldNotBeNil)
				So(response.CommunityCards, ShouldNotBeNil)
				So(response.Conflicts, ShouldNotBeNil)
				So(response.QueryExpansions, ShouldNotBeNil)
				So(response.TotalResults, ShouldEqual, 0)
				So(response.HasMore, ShouldBeFalse)
				So(response.NextOffset, ShouldEqual, 0)
			})
		})

		Convey("When validating valid response", func() {
			response := NewRecallResponse()
			err := response.Validate()

			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating invalid response", func() {
			Convey("With nil evidence", func() {
				response := NewRecallResponse()
				response.Evidence = nil
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "evidence cannot be nil")
				})
			})

			Convey("With negative total results", func() {
				response := NewRecallResponse()
				response.TotalResults = -1
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "total results cannot be negative")
				})
			})

			Convey("With negative next offset", func() {
				response := NewRecallResponse()
				response.NextOffset = -1
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "next offset cannot be negative")
				})
			})
		})

		Convey("When adding evidence", func() {
			response := NewRecallResponse()
			evidence := Evidence{
				Content:    "test evidence",
				Source:     "test",
				Confidence: 0.8,
			}
			response.AddEvidence(evidence)

			Convey("Then evidence should be added and total results updated", func() {
				So(len(response.Evidence), ShouldEqual, 1)
				So(response.TotalResults, ShouldEqual, 1)
				So(response.Evidence[0].Content, ShouldEqual, "test evidence")
			})
		})

		Convey("When adding conflict", func() {
			response := NewRecallResponse()
			conflict := ConflictInfo{
				ID:          "conflict1",
				Type:        "contradiction",
				Description: "test conflict",
			}
			response.AddConflict(conflict)

			Convey("Then conflict should be added", func() {
				So(len(response.Conflicts), ShouldEqual, 1)
				So(response.Conflicts[0].ID, ShouldEqual, "conflict1")
			})
		})
	})
}

func TestWriteResponse(t *testing.T) {
	Convey("Given WriteResponse", t, func() {
		Convey("When creating new WriteResponse", func() {
			response := NewWriteResponse("mem_123")

			Convey("Then it should have correct default values", func() {
				So(response.MemoryID, ShouldEqual, "mem_123")
				So(response.CandidateCount, ShouldEqual, 0)
				So(response.ConflictsFound, ShouldNotBeNil)
				So(response.EntitiesLinked, ShouldNotBeNil)
				So(response.ProvenanceID, ShouldEqual, "")
				So(response.ChunksCreated, ShouldEqual, 0)
				So(response.ProcessingTime, ShouldEqual, 0)
				So(response.Warnings, ShouldNotBeNil)
			})
		})

		Convey("When validating valid response", func() {
			response := NewWriteResponse("mem_123")
			err := response.Validate()

			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating invalid response", func() {
			Convey("With empty memory ID", func() {
				response := NewWriteResponse("")
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "memory ID cannot be empty")
				})
			})

			Convey("With negative candidate count", func() {
				response := NewWriteResponse("mem_123")
				response.CandidateCount = -1
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "candidate count cannot be negative")
				})
			})

			Convey("With negative chunks created", func() {
				response := NewWriteResponse("mem_123")
				response.ChunksCreated = -1
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "chunks created cannot be negative")
				})
			})

			Convey("With negative processing time", func() {
				response := NewWriteResponse("mem_123")
				response.ProcessingTime = -1 * time.Second
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "processing time cannot be negative")
				})
			})
		})

		Convey("When adding warning", func() {
			response := NewWriteResponse("mem_123")
			response.AddWarning("test warning")

			Convey("Then warning should be added", func() {
				So(len(response.Warnings), ShouldEqual, 1)
				So(response.Warnings[0], ShouldEqual, "test warning")
			})
		})

		Convey("When adding entity link", func() {
			response := NewWriteResponse("mem_123")
			response.AddEntityLink("entity_1")
			response.AddEntityLink("entity_2")
			response.AddEntityLink("entity_1") // Duplicate

			Convey("Then entities should be added without duplicates", func() {
				So(len(response.EntitiesLinked), ShouldEqual, 2)
				So(response.EntitiesLinked, ShouldContain, "entity_1")
				So(response.EntitiesLinked, ShouldContain, "entity_2")
			})
		})
	})
}

func TestManageResponse(t *testing.T) {
	Convey("Given ManageResponse", t, func() {
		Convey("When creating new ManageResponse", func() {
			response := NewManageResponse("pin")

			Convey("Then it should have correct default values", func() {
				So(response.Operation, ShouldEqual, "pin")
				So(response.AffectedCount, ShouldEqual, 0)
				So(response.Success, ShouldBeFalse)
				So(response.Message, ShouldEqual, "")
				So(response.ProcessingTime, ShouldEqual, 0)
				So(response.Errors, ShouldNotBeNil)
				So(response.Preview, ShouldNotBeNil)
			})
		})

		Convey("When validating valid response", func() {
			response := NewManageResponse("pin")
			err := response.Validate()

			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating invalid response", func() {
			Convey("With empty operation", func() {
				response := NewManageResponse("")
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "operation cannot be empty")
				})
			})

			Convey("With negative affected count", func() {
				response := NewManageResponse("pin")
				response.AffectedCount = -1
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "affected count cannot be negative")
				})
			})

			Convey("With negative processing time", func() {
				response := NewManageResponse("pin")
				response.ProcessingTime = -1 * time.Second
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "processing time cannot be negative")
				})
			})
		})

		Convey("When adding error", func() {
			response := NewManageResponse("pin")
			response.AddError("test error")

			Convey("Then error should be added and success set to false", func() {
				So(len(response.Errors), ShouldEqual, 1)
				So(response.Errors[0], ShouldEqual, "test error")
				So(response.Success, ShouldBeFalse)
			})
		})

		Convey("When adding preview item", func() {
			response := NewManageResponse("pin")
			response.AddPreviewItem("preview item")

			Convey("Then preview item should be added", func() {
				So(len(response.Preview), ShouldEqual, 1)
				So(response.Preview[0], ShouldEqual, "preview item")
			})
		})
	})
}

func TestStatsResponse(t *testing.T) {
	Convey("Given StatsResponse", t, func() {
		Convey("When creating new StatsResponse", func() {
			response := NewStatsResponse()

			Convey("Then it should have correct default values", func() {
				So(response.TotalMemories, ShouldEqual, 0)
				So(response.GraphNodes, ShouldEqual, 0)
				So(response.GraphEdges, ShouldEqual, 0)
				So(response.VectorDimensions, ShouldEqual, 0)
				So(response.Timestamp, ShouldNotBeZeroValue)
			})
		})

		Convey("When validating valid response", func() {
			response := NewStatsResponse()
			err := response.Validate()

			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating invalid response", func() {
			Convey("With negative total memories", func() {
				response := NewStatsResponse()
				response.TotalMemories = -1
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "total memories cannot be negative")
				})
			})

			Convey("With negative graph nodes", func() {
				response := NewStatsResponse()
				response.GraphNodes = -1
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "graph nodes cannot be negative")
				})
			})

			Convey("With negative graph edges", func() {
				response := NewStatsResponse()
				response.GraphEdges = -1
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "graph edges cannot be negative")
				})
			})

			Convey("With negative vector dimensions", func() {
				response := NewStatsResponse()
				response.VectorDimensions = -1
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "vector dimensions cannot be negative")
				})
			})

			Convey("With zero timestamp", func() {
				response := NewStatsResponse()
				response.Timestamp = time.Time{}
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "timestamp cannot be zero")
				})
			})
		})
	})
}

func TestSearchResponse(t *testing.T) {
	Convey("Given SearchResponse", t, func() {
		Convey("When creating new SearchResponse", func() {
			response := NewSearchResponse()

			Convey("Then it should have correct default values", func() {
				So(response.Results, ShouldNotBeNil)
				So(response.TotalResults, ShouldEqual, 0)
				So(response.HasMore, ShouldBeFalse)
				So(response.NextOffset, ShouldEqual, 0)
				So(response.Suggestions, ShouldNotBeNil)
				So(response.Facets, ShouldNotBeNil)
			})
		})

		Convey("When validating valid response", func() {
			response := NewSearchResponse()
			err := response.Validate()

			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating invalid response", func() {
			Convey("With nil results", func() {
				response := NewSearchResponse()
				response.Results = nil
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "results cannot be nil")
				})
			})

			Convey("With negative total results", func() {
				response := NewSearchResponse()
				response.TotalResults = -1
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "total results cannot be negative")
				})
			})

			Convey("With negative next offset", func() {
				response := NewSearchResponse()
				response.NextOffset = -1
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "next offset cannot be negative")
				})
			})
		})

		Convey("When adding result", func() {
			response := NewSearchResponse()
			result := SearchResult{
				ID:      "result_1",
				Score:   0.8,
				Content: "test content",
			}
			response.AddResult(result)

			Convey("Then result should be added and total results updated", func() {
				So(len(response.Results), ShouldEqual, 1)
				So(response.TotalResults, ShouldEqual, 1)
				So(response.Results[0].ID, ShouldEqual, "result_1")
			})
		})

		Convey("When adding suggestion", func() {
			response := NewSearchResponse()
			response.AddSuggestion("test suggestion")

			Convey("Then suggestion should be added", func() {
				So(len(response.Suggestions), ShouldEqual, 1)
				So(response.Suggestions[0], ShouldEqual, "test suggestion")
			})
		})
	})
}

func TestGraphResponse(t *testing.T) {
	Convey("Given GraphResponse", t, func() {
		Convey("When creating new GraphResponse", func() {
			response := NewGraphResponse()

			Convey("Then it should have correct default values", func() {
				So(response.Nodes, ShouldNotBeNil)
				So(response.Edges, ShouldNotBeNil)
				So(response.Paths, ShouldNotBeNil)
				So(response.Communities, ShouldNotBeNil)
				So(response.ProcessingTime, ShouldEqual, 0)
			})
		})

		Convey("When validating valid response", func() {
			response := NewGraphResponse()
			err := response.Validate()

			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating invalid response", func() {
			Convey("With nil nodes", func() {
				response := NewGraphResponse()
				response.Nodes = nil
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "nodes cannot be nil")
				})
			})

			Convey("With nil edges", func() {
				response := NewGraphResponse()
				response.Edges = nil
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "edges cannot be nil")
				})
			})

			Convey("With negative processing time", func() {
				response := NewGraphResponse()
				response.ProcessingTime = -1 * time.Second
				err := response.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "processing time cannot be negative")
				})
			})
		})

		Convey("When adding node", func() {
			response := NewGraphResponse()
			node := *NewNode("node_1", EntityNode)
			response.AddNode(node)

			Convey("Then node should be added and metrics updated", func() {
				So(len(response.Nodes), ShouldEqual, 1)
				So(response.Metrics.NodeCount, ShouldEqual, 1)
				So(response.Nodes[0].ID, ShouldEqual, "node_1")
			})
		})

		Convey("When adding edge", func() {
			response := NewGraphResponse()
			edge := *NewEdge("edge_1", "node_1", "node_2", RelatedTo, 1.0)
			response.AddEdge(edge)

			Convey("Then edge should be added and metrics updated", func() {
				So(len(response.Edges), ShouldEqual, 1)
				So(response.Metrics.EdgeCount, ShouldEqual, 1)
				So(response.Edges[0].ID, ShouldEqual, "edge_1")
			})
		})

		Convey("When adding path", func() {
			response := NewGraphResponse()
			path := Path{
				Nodes: []string{"node_1", "node_2"},
				Edges: []string{"edge_1"},
				Cost:  1.0,
			}
			response.AddPath(path)

			Convey("Then path should be added", func() {
				So(len(response.Paths), ShouldEqual, 1)
				So(response.Paths[0].Cost, ShouldEqual, 1.0)
			})
		})

		Convey("When adding community", func() {
			response := NewGraphResponse()
			community := Community{
				ID:    "comm_1",
				Nodes: []string{"node_1", "node_2"},
				Score: 0.8,
			}
			response.AddCommunity(community)

			Convey("Then community should be added", func() {
				So(len(response.Communities), ShouldEqual, 1)
				So(response.Communities[0].ID, ShouldEqual, "comm_1")
			})
		})
	})
}

// Benchmark tests
func BenchmarkNewRecallResponse(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response := NewRecallResponse()
		_ = response
	}
}

func BenchmarkRecallResponseValidation(b *testing.B) {
	response := NewRecallResponse()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := response.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteResponseValidation(b *testing.B) {
	response := NewWriteResponse("mem_123")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := response.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStatsResponseValidation(b *testing.B) {
	response := NewStatsResponse()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := response.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}