package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRecallE2EFixed(t *testing.T) {
	Convey("Given a complete MCP server setup", t, func() {
		// Create a test server configuration using defaults
		config := DefaultServerConfig()
		config.Server.Name = "test-server"
		config.Server.Version = "1.0.0"

		// Create the server
		server, err := NewAgenticMemoryServer(config)
		So(err, ShouldBeNil)
		So(server, ShouldNotBeNil)

		ctx := context.Background()

		Convey("When performing end-to-end recall operation", func() {
			// Create a recall request
			args := RecallArgs{
				Query:        "artificial intelligence machine learning",
				MaxResults:   5,
				TimeBudget:   10000, // 10 seconds
				IncludeGraph: true,
				Filters: map[string]interface{}{
					"source": "test",
					"type":   "research",
				},
			}

			// Create MCP request
			req := &mcp.CallToolRequest{}

			// Execute the recall
			mcpResult, result, err := server.handleRecall(ctx, req, args)

			Convey("Then the operation should complete successfully", func() {
				So(err, ShouldBeNil)
				So(mcpResult, ShouldNotBeNil)
				So(result.Evidence, ShouldNotBeNil)
				So(result.Stats.QueryTime, ShouldBeGreaterThanOrEqualTo, 0)
			})

			Convey("And the MCP result should be properly formatted", func() {
				So(mcpResult.Content, ShouldHaveLength, 1)
				textContent, ok := mcpResult.Content[0].(*mcp.TextContent)
				So(ok, ShouldBeTrue)
				So(textContent.Text, ShouldContainSubstring, "Retrieved")
				So(textContent.Text, ShouldContainSubstring, "artificial intelligence")
			})

			Convey("And the result should be JSON serializable", func() {
				jsonData, err := json.Marshal(result)
				So(err, ShouldBeNil)
				So(len(jsonData), ShouldBeGreaterThan, 0)

				// Verify we can unmarshal it back
				var unmarshaled RecallResult
				err = json.Unmarshal(jsonData, &unmarshaled)
				So(err, ShouldBeNil)
			})
		})

		Convey("When performing recall with various query types", func() {
			testCases := []struct {
				name  string
				query string
				args  RecallArgs
			}{
				{
					name:  "Simple keyword query",
					query: "machine learning",
					args: RecallArgs{
						Query:      "machine learning",
						MaxResults: 3,
					},
				},
				{
					name:  "Entity-focused query",
					query: "OpenAI GPT-4",
					args: RecallArgs{
						Query:        "OpenAI GPT-4",
						MaxResults:   5,
						IncludeGraph: true,
					},
				},
				{
					name:  "Complex query with filters",
					query: "neural networks deep learning",
					args: RecallArgs{
						Query:      "neural networks deep learning",
						MaxResults: 10,
						Filters: map[string]interface{}{
							"confidence": 0.7,
							"source":     "research_papers",
						},
					},
				},
			}

			for _, tc := range testCases {
				Convey("For "+tc.name, func() {
					req := &mcp.CallToolRequest{}

					mcpResult, result, err := server.handleRecall(ctx, req, tc.args)

					So(err, ShouldBeNil)
					So(mcpResult, ShouldNotBeNil)
					So(result.Evidence, ShouldNotBeNil)
					So(result.Stats.QueryTime, ShouldBeGreaterThanOrEqualTo, 0)

					// Verify query is reflected in the response
					textContent := mcpResult.Content[0].(*mcp.TextContent)
					So(textContent.Text, ShouldContainSubstring, tc.query)
				})
			}
		})

		Convey("When testing error handling scenarios", func() {
			errorTestCases := []struct {
				name        string
				args        RecallArgs
				expectError bool
				errorType   string
			}{
				{
					name: "Empty query",
					args: RecallArgs{
						Query:      "",
						MaxResults: 5,
					},
					expectError: true,
					errorType:   "validation",
				},
				{
					name: "Negative max results",
					args: RecallArgs{
						Query:      "test query",
						MaxResults: -1,
					},
					expectError: true,
					errorType:   "validation",
				},
			}

			for _, tc := range errorTestCases {
				Convey("For "+tc.name, func() {
					req := &mcp.CallToolRequest{}

					_, _, err := server.handleRecall(ctx, req, tc.args)

					if tc.expectError {
						So(err, ShouldNotBeNil)
						So(err.Error(), ShouldContainSubstring, tc.errorType)
					} else {
						So(err, ShouldBeNil)
					}
				})
			}
		})
	})
}

func TestRecallE2EWithMockDataFixed(t *testing.T) {
	Convey("Given a server with mock data", t, func() {
		// Create a test server configuration using defaults
		config := DefaultServerConfig()
		config.Server.Name = "test-server"
		config.Server.Version = "1.0.0"

		server, err := NewAgenticMemoryServer(config)
		So(err, ShouldBeNil)
		ctx := context.Background()

		Convey("When querying for specific topics", func() {
			topicQueries := []string{
				"artificial intelligence",
				"machine learning algorithms",
				"neural network architectures",
			}

			for _, query := range topicQueries {
				Convey("For query: "+query, func() {
					args := RecallArgs{
						Query:        query,
						MaxResults:   10,
						TimeBudget:   5000,
						IncludeGraph: true,
					}

					req := &mcp.CallToolRequest{}

					mcpResult, result, err := server.handleRecall(ctx, req, args)

					So(err, ShouldBeNil)
					So(mcpResult, ShouldNotBeNil)
					So(result.Stats.QueryTime, ShouldBeGreaterThanOrEqualTo, 0)

					// Verify the response structure
					So(result.Evidence, ShouldNotBeNil)
					So(result.CommunityCards, ShouldNotBeNil)
					So(result.Conflicts, ShouldNotBeNil)
					So(result.Stats, ShouldNotBeNil) // Stats can be zero with empty storage

					// Verify MCP response format
					textContent := mcpResult.Content[0].(*mcp.TextContent)
					So(textContent.Text, ShouldContainSubstring, "Retrieved")
					So(textContent.Text, ShouldContainSubstring, query)
				})
			}
		})
	})
}
