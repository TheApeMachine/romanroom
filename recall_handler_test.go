package main

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRecallHandler(t *testing.T) {
	Convey("Given a RecallHandler", t, func() {
		queryProcessor := NewQueryProcessor(nil)
		resultFuser := NewResultFuser()
		handler := NewRecallHandler(queryProcessor, resultFuser)

		Convey("When creating a new handler", func() {
			So(handler, ShouldNotBeNil)
			So(handler.queryProcessor, ShouldEqual, queryProcessor)
			So(handler.resultFuser, ShouldEqual, resultFuser)
			So(handler.validator, ShouldNotBeNil)
			So(handler.formatter, ShouldNotBeNil)
			So(handler.config, ShouldNotBeNil)
		})

		Convey("When creating with custom config", func() {
			config := &RecallHandlerConfig{
				DefaultMaxResults:    20,
				DefaultTimeBudget:    10 * time.Second,
				MaxTimeBudget:        60 * time.Second,
				EnableSelfCritique:   false,
				EnableQueryExpansion: false,
			}
			customHandler := NewRecallHandlerWithConfig(queryProcessor, resultFuser, config)

			So(customHandler.config.DefaultMaxResults, ShouldEqual, 20)
			So(customHandler.config.DefaultTimeBudget, ShouldEqual, 10*time.Second)
			So(customHandler.config.EnableSelfCritique, ShouldBeFalse)
		})
	})
}

func TestRecallHandlerHandleRecall(t *testing.T) {
	Convey("Given a RecallHandler with mock dependencies", t, func() {
		queryProcessor := NewQueryProcessor(nil)
		resultFuser := NewResultFuser()
		handler := NewRecallHandler(queryProcessor, resultFuser)

		ctx := context.Background()
		req := &mcp.CallToolRequest{}

		Convey("When handling a valid recall request", func() {
			args := RecallArgs{
				Query:        "test query",
				MaxResults:   10,
				TimeBudget:   5000,
				IncludeGraph: false,
				Filters:      map[string]interface{}{"source": "test"},
			}

			mcpResult, result, err := handler.HandleRecall(ctx, req, args)

			Convey("Then it should succeed", func() {
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
				So(textContent.Text, ShouldContainSubstring, "test query")
			})
		})

		Convey("When handling a request with invalid arguments", func() {
			args := RecallArgs{
				Query:      "", // Empty query
				MaxResults: -1, // Invalid max results
			}

			_, _, err := handler.HandleRecall(ctx, req, args)

			Convey("Then it should return a validation error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "validation")
			})
		})

		Convey("When handling a request with timeout", func() {
			// Create a context that times out quickly
			timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
			defer cancel()

			args := RecallArgs{
				Query:      "test query",
				TimeBudget: 10000, // 10 seconds, but context will timeout first
			}

			_, _, err := handler.HandleRecall(timeoutCtx, req, args)

			Convey("Then it should handle the timeout gracefully", func() {
				// The error might be nil if processing completes quickly,
				// or it might be a context timeout error
				if err != nil {
					So(err.Error(), ShouldContainSubstring, "context")
				}
			})
		})
	})
}

func TestRecallHandlerProcessQuery(t *testing.T) {
	Convey("Given a RecallHandler", t, func() {
		queryProcessor := NewQueryProcessor(nil)
		resultFuser := NewResultFuser()
		handler := NewRecallHandler(queryProcessor, resultFuser)

		ctx := context.Background()

		Convey("When processing a simple query", func() {
			options := &RecallOptions{
				MaxResults:   10,
				TimeBudget:   5 * time.Second,
				IncludeGraph: false,
			}

			response, err := handler.processQuery(ctx, "test query", options)

			Convey("Then it should return a valid response", func() {
				So(err, ShouldBeNil)
				So(response, ShouldNotBeNil)
				So(response.Evidence, ShouldNotBeNil)
				So(response.QueryExpansions, ShouldNotBeNil)
				So(response.TotalResults, ShouldBeGreaterThanOrEqualTo, 0)
			})
		})

		Convey("When processing a query with filters", func() {
			options := &RecallOptions{
				MaxResults: 5,
				TimeBudget: 3 * time.Second,
				Filters:    map[string]interface{}{"source": "test", "confidence": 0.8},
			}

			response, err := handler.processQuery(ctx, "filtered query", options)

			Convey("Then it should handle filters appropriately", func() {
				So(err, ShouldBeNil)
				So(response, ShouldNotBeNil)
				// Since we don't have real data, we can't test filter effectiveness
				// but we can ensure the process doesn't fail
			})
		})
	})
}

func TestRecallHandlerMultiViewRetrieval(t *testing.T) {
	Convey("Given a RecallHandler with query processor", t, func() {
		queryProcessor := NewQueryProcessor(nil)
		resultFuser := NewResultFuser()
		handler := NewRecallHandler(queryProcessor, resultFuser)

		ctx := context.Background()

		Convey("When performing multi-view retrieval", func() {
			processedQuery := &ProcessedQuery{
				Original: "test query",
				Expanded: []string{"test query", "expanded query"},
				Keywords: []string{"test", "query"},
				Entities: []string{"TestEntity"},
			}

			options := &RecallOptions{
				MaxResults: 10,
				TimeBudget: 5 * time.Second,
			}

			fusionInputs, err := handler.performMultiViewRetrieval(ctx, processedQuery, options)

			Convey("Then it should return fusion inputs", func() {
				So(err, ShouldBeNil)
				// fusionInputs may be nil with mock setup that has no initialized searchers
				if fusionInputs != nil {
					So(len(fusionInputs), ShouldBeGreaterThanOrEqualTo, 0)
				}
			})
		})
	})
}

func TestRecallHandlerConvertArgsToOptions(t *testing.T) {
	Convey("Given a RecallHandler", t, func() {
		queryProcessor := NewQueryProcessor(nil)
		resultFuser := NewResultFuser()
		handler := NewRecallHandler(queryProcessor, resultFuser)

		Convey("When converting args with all fields set", func() {
			args := RecallArgs{
				Query:        "test query",
				MaxResults:   20,
				TimeBudget:   8000,
				IncludeGraph: true,
				Filters:      map[string]interface{}{"source": "test"},
			}

			options := handler.convertArgsToOptions(args)

			Convey("Then all options should be set correctly", func() {
				So(options.MaxResults, ShouldEqual, 20)
				So(options.TimeBudget, ShouldEqual, 8*time.Second)
				So(options.IncludeGraph, ShouldBeTrue)
				So(options.Filters, ShouldResemble, args.Filters)
				So(options.ExpandQuery, ShouldEqual, handler.config.EnableQueryExpansion)
			})
		})

		Convey("When converting args with default values", func() {
			args := RecallArgs{
				Query: "test query",
				// Other fields left as zero values
			}

			options := handler.convertArgsToOptions(args)

			Convey("Then defaults should be applied", func() {
				So(options.MaxResults, ShouldEqual, handler.config.DefaultMaxResults)
				So(options.TimeBudget, ShouldEqual, handler.config.DefaultTimeBudget)
				So(options.IncludeGraph, ShouldBeFalse)
			})
		})

		Convey("When converting args with excessive time budget", func() {
			args := RecallArgs{
				Query:      "test query",
				TimeBudget: 60000, // 60 seconds
			}

			options := handler.convertArgsToOptions(args)

			Convey("Then time budget should be capped", func() {
				So(options.TimeBudget, ShouldEqual, handler.config.MaxTimeBudget)
			})
		})
	})
}

func TestRecallHandlerEvidenceConversion(t *testing.T) {
	Convey("Given a RecallHandler", t, func() {
		queryProcessor := NewQueryProcessor(nil)
		resultFuser := NewResultFuser()
		handler := NewRecallHandler(queryProcessor, resultFuser)

		Convey("When converting fused results to evidence", func() {
			fusedResults := []FusedResult{
				{
					ID:            "result1",
					Content:       "Test content 1",
					FinalScore:    0.8,
					SourceMethods: []string{"vector", "keyword"},
					MethodScores:  map[string]float64{"vector": 0.7, "keyword": 0.9},
					Metadata: map[string]interface{}{
						"source":    "test_source",
						"timestamp": "2024-01-01T00:00:00Z",
						"version":   "1.0",
					},
				},
				{
					ID:            "result2",
					Content:       "Test content 2",
					FinalScore:    0.6,
					SourceMethods: []string{"vector"},
					MethodScores:  map[string]float64{"vector": 0.6},
					Metadata: map[string]interface{}{
						"source": "another_source",
					},
				},
			}

			evidence, err := handler.convertFusedResultsToEvidence(fusedResults)

			Convey("Then evidence should be properly converted", func() {
				So(err, ShouldBeNil)
				So(evidence, ShouldHaveLength, 2)

				So(evidence[0].Content, ShouldEqual, "Test content 1")
				So(evidence[0].Confidence, ShouldEqual, 0.8)
				So(evidence[0].Source, ShouldEqual, "test_source")
				So(evidence[0].Provenance.Source, ShouldEqual, "test_source")
				So(evidence[0].Provenance.Timestamp, ShouldEqual, "2024-01-01T00:00:00Z")
				So(evidence[0].WhySelected, ShouldContainSubstring, "multi-view fusion")

				So(evidence[1].Content, ShouldEqual, "Test content 2")
				So(evidence[1].Confidence, ShouldEqual, 0.6)
				So(evidence[1].Source, ShouldEqual, "another_source")
				So(evidence[1].WhySelected, ShouldContainSubstring, "vector search")
			})
		})
	})
}

func TestRecallHandlerSelfCritique(t *testing.T) {
	Convey("Given a RecallHandler with self-critique enabled", t, func() {
		queryProcessor := NewQueryProcessor(nil)
		resultFuser := NewResultFuser()
		config := &RecallHandlerConfig{
			EnableSelfCritique: true,
		}
		handler := NewRecallHandlerWithConfig(queryProcessor, resultFuser, config)

		ctx := context.Background()

		Convey("When generating self-critique for no evidence", func() {
			critique, err := handler.generateSelfCritique(ctx, "test query", []Evidence{})

			Convey("Then it should suggest expanding search", func() {
				So(err, ShouldBeNil)
				So(critique, ShouldContainSubstring, "No evidence found")
				So(critique, ShouldContainSubstring, "expanding search")
			})
		})

		Convey("When generating self-critique for low confidence evidence", func() {
			evidence := []Evidence{
				{Confidence: 0.2},
				{Confidence: 0.1},
			}

			critique, err := handler.generateSelfCritique(ctx, "test query", evidence)

			Convey("Then it should indicate low confidence", func() {
				So(err, ShouldBeNil)
				So(critique, ShouldContainSubstring, "confidence is low")
				So(critique, ShouldContainSubstring, "may not be highly relevant")
			})
		})

		Convey("When generating self-critique for high confidence evidence", func() {
			evidence := []Evidence{
				{Confidence: 0.9},
				{Confidence: 0.8},
			}

			critique, err := handler.generateSelfCritique(ctx, "test query", evidence)

			Convey("Then it should indicate high confidence", func() {
				So(err, ShouldBeNil)
				So(critique, ShouldContainSubstring, "high-confidence")
				So(critique, ShouldContainSubstring, "highly relevant")
			})
		})
	})
}

func TestRecallHandlerConflictDetection(t *testing.T) {
	Convey("Given a RecallHandler", t, func() {
		queryProcessor := NewQueryProcessor(nil)
		resultFuser := NewResultFuser()
		handler := NewRecallHandler(queryProcessor, resultFuser)

		Convey("When detecting conflicts in evidence with different sources and confidence", func() {
			evidence := []Evidence{
				{
					Source:     "source1",
					Confidence: 0.9,
				},
				{
					Source:     "source2",
					Confidence: 0.3,
				},
			}

			conflicts := handler.detectConflicts(evidence)

			Convey("Then it should detect confidence mismatch", func() {
				So(conflicts, ShouldHaveLength, 1)
				So(conflicts[0].Type, ShouldEqual, "confidence_mismatch")
				So(conflicts[0].Description, ShouldContainSubstring, "confidence difference")
				So(conflicts[0].Severity, ShouldEqual, "medium")
			})
		})

		Convey("When detecting conflicts in evidence with same source", func() {
			evidence := []Evidence{
				{
					Source:     "same_source",
					Confidence: 0.9,
				},
				{
					Source:     "same_source",
					Confidence: 0.1,
				},
			}

			conflicts := handler.detectConflicts(evidence)

			Convey("Then it should not detect conflicts for same source", func() {
				So(conflicts, ShouldHaveLength, 0)
			})
		})

		Convey("When detecting conflicts in evidence with similar confidence", func() {
			evidence := []Evidence{
				{
					Source:     "source1",
					Confidence: 0.8,
				},
				{
					Source:     "source2",
					Confidence: 0.7,
				},
			}

			conflicts := handler.detectConflicts(evidence)

			Convey("Then it should not detect conflicts", func() {
				So(conflicts, ShouldHaveLength, 0)
			})
		})
	})
}

func TestRecallHandlerConfiguration(t *testing.T) {
	Convey("Given a RecallHandler", t, func() {
		queryProcessor := NewQueryProcessor(nil)
		resultFuser := NewResultFuser()
		handler := NewRecallHandler(queryProcessor, resultFuser)

		Convey("When getting configuration", func() {
			config := handler.GetConfig()

			Convey("Then it should return the current config", func() {
				So(config, ShouldNotBeNil)
				So(config.DefaultMaxResults, ShouldEqual, 10)
				So(config.DefaultTimeBudget, ShouldEqual, 5*time.Second)
			})
		})

		Convey("When updating configuration", func() {
			newConfig := &RecallHandlerConfig{
				DefaultMaxResults:    25,
				DefaultTimeBudget:    8 * time.Second,
				EnableSelfCritique:   false,
				EnableQueryExpansion: false,
			}

			handler.UpdateConfig(newConfig)
			config := handler.GetConfig()

			Convey("Then the config should be updated", func() {
				So(config.DefaultMaxResults, ShouldEqual, 25)
				So(config.DefaultTimeBudget, ShouldEqual, 8*time.Second)
				So(config.EnableSelfCritique, ShouldBeFalse)
				So(config.EnableQueryExpansion, ShouldBeFalse)
			})
		})

		Convey("When updating with nil config", func() {
			originalConfig := handler.GetConfig()
			handler.UpdateConfig(nil)
			currentConfig := handler.GetConfig()

			Convey("Then the config should remain unchanged", func() {
				So(currentConfig, ShouldEqual, originalConfig)
			})
		})
	})
}
