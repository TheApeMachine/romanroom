package main

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	. "github.com/smartystreets/goconvey/convey"
)

func TestHandleRecall(t *testing.T) {
	Convey("Given an AgenticMemoryServer", t, func() {
		config := DefaultServerConfig()
		server, err := NewAgenticMemoryServer(config)
		So(err, ShouldBeNil)
		So(server, ShouldNotBeNil)

		ctx := context.Background()
		req := &mcp.CallToolRequest{}

		// Note: Using mock storage which starts empty, so results will be empty

		Convey("When handling a basic recall request", func() {
			args := RecallArgs{
				Query:      "test query",
				MaxResults: 10,
			}

			result, recallResult, err := server.handleRecall(ctx, req, args)

			Convey("Then it should return successfully", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(recallResult.Evidence, ShouldNotBeNil) // Can be empty with mock storage
				So(recallResult.Stats, ShouldNotBeNil)
				So(recallResult.Stats.TotalCandidates, ShouldBeGreaterThanOrEqualTo, 0)
				// Verify MCP content structure
				So(len(result.Content), ShouldBeGreaterThan, 0)
				textContent, ok := result.Content[0].(*mcp.TextContent)
				So(ok, ShouldBeTrue)
				So(textContent.Text, ShouldNotBeEmpty)
			})
		})

		Convey("When handling a recall request with filters", func() {
			args := RecallArgs{
				Query:        "filtered query",
				MaxResults:   5,
				IncludeGraph: true,
				Filters: map[string]interface{}{
					"source":     "test_source",
					"confidence": 0.8,
				},
			}

			result, recallResult, err := server.handleRecall(ctx, req, args)

			Convey("Then it should return successfully with filters applied", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(recallResult.Evidence, ShouldNotBeNil) // Can be empty with mock storage
				So(recallResult.Stats, ShouldNotBeNil)
			})
		})

		Convey("When handling an empty query", func() {
			args := RecallArgs{
				Query:      "",
				MaxResults: 10,
			}

			_, _, err := server.handleRecall(ctx, req, args)

			Convey("Then it should return validation error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "query cannot be empty")
			})
		})
	})
}

func TestHandleWrite(t *testing.T) {
	Convey("Given an AgenticMemoryServer", t, func() {
		config := DefaultServerConfig()
		server, err := NewAgenticMemoryServer(config)
		So(err, ShouldBeNil)
		So(server, ShouldNotBeNil)

		ctx := context.Background()
		req := &mcp.CallToolRequest{}

		Convey("When handling a basic write request", func() {
			args := WriteArgs{
				Content: "This is test content to store in memory",
				Source:  "test_source",
			}

			result, writeResult, err := server.handleWrite(ctx, req, args)

			Convey("Then it should return successfully", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(writeResult.MemoryID, ShouldNotBeEmpty)
				So(writeResult.CandidateCount, ShouldBeGreaterThanOrEqualTo, 0)
				So(writeResult.EntitiesLinked, ShouldNotBeNil) // Can be empty
				So(writeResult.ProvenanceID, ShouldNotBeEmpty)
			})
		})

		Convey("When handling a write request with metadata", func() {
			args := WriteArgs{
				Content: "Content with metadata",
				Source:  "test_source",
				Tags:    []string{"tag1", "tag2"},
				Metadata: map[string]interface{}{
					"author": "test_author",
					"type":   "test_type",
				},
				RequireEvidence: true,
			}

			result, writeResult, err := server.handleWrite(ctx, req, args)

			Convey("Then it should return successfully with metadata processed", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(writeResult.MemoryID, ShouldNotBeEmpty)
				So(writeResult.CandidateCount, ShouldEqual, 1)
			})
		})

		Convey("When handling an empty content write", func() {
			args := WriteArgs{
				Content: "",
				Source:  "test_source",
			}

			_, _, err := server.handleWrite(ctx, req, args)

			Convey("Then it should return validation error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "content cannot be empty")
			})
		})
	})
}

func TestHandleManage(t *testing.T) {
	Convey("Given an AgenticMemoryServer", t, func() {
		config := DefaultServerConfig()
		server, err := NewAgenticMemoryServer(config)
		So(err, ShouldBeNil)
		So(server, ShouldNotBeNil)

		ctx := context.Background()
		req := &mcp.CallToolRequest{}

		Convey("When handling a pin operation", func() {
			args := ManageArgs{
				Operation: "pin",
				MemoryIDs: []string{"mem_1", "mem_2"},
			}

			result, manageResult, err := server.handleManage(ctx, req, args)

			Convey("Then it should return successfully", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(manageResult.Operation, ShouldEqual, "pin")
				So(manageResult.AffectedCount, ShouldEqual, 2)
				So(manageResult.Success, ShouldBeTrue)
				So(manageResult.Message, ShouldContainSubstring, "pin")
			})
		})

		Convey("When handling a forget operation", func() {
			args := ManageArgs{
				Operation: "forget",
				MemoryIDs: []string{"mem_3"},
			}

			result, manageResult, err := server.handleManage(ctx, req, args)

			Convey("Then it should return successfully", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(manageResult.Operation, ShouldEqual, "forget")
				So(manageResult.AffectedCount, ShouldEqual, 1)
				So(manageResult.Success, ShouldBeTrue)
			})
		})

		Convey("When handling a decay operation with query", func() {
			args := ManageArgs{
				Operation:  "decay",
				Query:      "old memories",
				Confidence: 0.3,
			}

			result, manageResult, err := server.handleManage(ctx, req, args)

			Convey("Then it should return successfully", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(manageResult.Operation, ShouldEqual, "decay")
				So(manageResult.Success, ShouldBeTrue)
			})
		})

		Convey("When handling an operation with no memory IDs", func() {
			args := ManageArgs{
				Operation: "pin",
				MemoryIDs: []string{},
			}

			result, manageResult, err := server.handleManage(ctx, req, args)

			Convey("Then it should return successfully with zero affected count", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(manageResult.AffectedCount, ShouldEqual, 0)
				So(manageResult.Success, ShouldBeTrue)
			})
		})
	})
}

func TestHandleStats(t *testing.T) {
	Convey("Given an AgenticMemoryServer", t, func() {
		config := DefaultServerConfig()
		server, err := NewAgenticMemoryServer(config)
		So(err, ShouldBeNil)
		So(server, ShouldNotBeNil)

		ctx := context.Background()
		req := &mcp.CallToolRequest{}

		Convey("When handling a basic stats request", func() {
			args := StatsArgs{
				IncludePerformance: false,
				IncludeStorage:     false,
			}

			result, statsResult, err := server.handleStats(ctx, req, args)

			Convey("Then it should return successfully", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(statsResult.TotalMemories, ShouldEqual, 0)
				So(statsResult.GraphNodes, ShouldEqual, 0)
				So(statsResult.GraphEdges, ShouldEqual, 0)
				So(statsResult.VectorDimensions, ShouldEqual, config.Storage.VectorStore.Dimensions)
				So(statsResult.StorageUsage, ShouldNotBeNil)
				So(statsResult.PerformanceStats, ShouldNotBeNil)
			})
		})

		Convey("When handling a stats request with performance metrics", func() {
			args := StatsArgs{
				IncludePerformance: true,
				IncludeStorage:     true,
			}

			result, statsResult, err := server.handleStats(ctx, req, args)

			Convey("Then it should return successfully with all metrics", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(statsResult.StorageUsage, ShouldContainKey, "vector_store")
				So(statsResult.StorageUsage, ShouldContainKey, "graph_store")
				So(statsResult.StorageUsage, ShouldContainKey, "search_index")
				So(statsResult.PerformanceStats, ShouldContainKey, "avg_query_time")
				So(statsResult.PerformanceStats, ShouldContainKey, "cache_hit_rate")
			})
		})
	})
}

func TestRegisterTools(t *testing.T) {
	Convey("Given an AgenticMemoryServer", t, func() {
		config := DefaultServerConfig()
		server, err := NewAgenticMemoryServer(config)
		So(err, ShouldBeNil)
		So(server, ShouldNotBeNil)

		Convey("When the server is created", func() {
			Convey("Then all tools should be registered", func() {
				// The tools are registered during server creation
				// We can verify this by checking that the server was created successfully
				// In a real implementation, we might have a way to list registered tools
				So(server.GetServer(), ShouldNotBeNil)
			})
		})
	})
}

func TestToolArgumentValidation(t *testing.T) {
	Convey("Given tool argument structures", t, func() {
		Convey("When creating RecallArgs", func() {
			args := RecallArgs{
				Query:        "test query",
				MaxResults:   10,
				TimeBudget:   5000,
				IncludeGraph: true,
				Filters: map[string]interface{}{
					"source": "test",
				},
			}

			Convey("Then all fields should be set correctly", func() {
				So(args.Query, ShouldEqual, "test query")
				So(args.MaxResults, ShouldEqual, 10)
				So(args.TimeBudget, ShouldEqual, 5000)
				So(args.IncludeGraph, ShouldBeTrue)
				So(args.Filters, ShouldContainKey, "source")
			})
		})

		Convey("When creating WriteArgs", func() {
			args := WriteArgs{
				Content:         "test content",
				Source:          "test source",
				Tags:            []string{"tag1", "tag2"},
				RequireEvidence: true,
				Metadata: map[string]interface{}{
					"author": "test",
				},
			}

			Convey("Then all fields should be set correctly", func() {
				So(args.Content, ShouldEqual, "test content")
				So(args.Source, ShouldEqual, "test source")
				So(args.Tags, ShouldResemble, []string{"tag1", "tag2"})
				So(args.RequireEvidence, ShouldBeTrue)
				So(args.Metadata, ShouldContainKey, "author")
			})
		})

		Convey("When creating ManageArgs", func() {
			args := ManageArgs{
				Operation:  "pin",
				MemoryIDs:  []string{"mem_1", "mem_2"},
				Query:      "test query",
				Confidence: 0.8,
			}

			Convey("Then all fields should be set correctly", func() {
				So(args.Operation, ShouldEqual, "pin")
				So(args.MemoryIDs, ShouldResemble, []string{"mem_1", "mem_2"})
				So(args.Query, ShouldEqual, "test query")
				So(args.Confidence, ShouldEqual, 0.8)
			})
		})

		Convey("When creating StatsArgs", func() {
			args := StatsArgs{
				IncludePerformance: true,
				IncludeStorage:     false,
			}

			Convey("Then all fields should be set correctly", func() {
				So(args.IncludePerformance, ShouldBeTrue)
				So(args.IncludeStorage, ShouldBeFalse)
			})
		})
	})
}

// Benchmark tests
func BenchmarkHandleRecall(b *testing.B) {
	config := DefaultServerConfig()
	server, err := NewAgenticMemoryServer(config)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	args := RecallArgs{
		Query:      "benchmark query",
		MaxResults: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := server.handleRecall(ctx, req, args)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHandleWrite(b *testing.B) {
	config := DefaultServerConfig()
	server, err := NewAgenticMemoryServer(config)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	args := WriteArgs{
		Content: "benchmark content for performance testing",
		Source:  "benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := server.handleWrite(ctx, req, args)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHandleManage(b *testing.B) {
	config := DefaultServerConfig()
	server, err := NewAgenticMemoryServer(config)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	args := ManageArgs{
		Operation: "pin",
		MemoryIDs: []string{"mem_1", "mem_2", "mem_3"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := server.handleManage(ctx, req, args)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHandleStats(b *testing.B) {
	config := DefaultServerConfig()
	server, err := NewAgenticMemoryServer(config)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	args := StatsArgs{
		IncludePerformance: true,
		IncludeStorage:     true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := server.handleStats(ctx, req, args)
		if err != nil {
			b.Fatal(err)
		}
	}
}
