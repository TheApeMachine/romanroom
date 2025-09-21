package main

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRecallResponseFormatter(t *testing.T) {
	Convey("Given a RecallResponseFormatter", t, func() {
		formatter := NewRecallResponseFormatter()

		Convey("When creating a new formatter", func() {
			So(formatter, ShouldNotBeNil)
			So(formatter.config, ShouldNotBeNil)
			So(formatter.config.MaxEvidenceLength, ShouldEqual, 500)
			So(formatter.config.TruncateContent, ShouldBeTrue)
			So(formatter.config.IncludeMetadata, ShouldBeTrue)
		})

		Convey("When creating with custom config", func() {
			config := &RecallFormatterConfig{
				MaxEvidenceLength:    200,
				TruncateContent:      false,
				IncludeMetadata:      false,
				SortByConfidence:     false,
				MinConfidenceDisplay: 0.5,
			}
			customFormatter := NewRecallResponseFormatterWithConfig(config)

			So(customFormatter.config.MaxEvidenceLength, ShouldEqual, 200)
			So(customFormatter.config.TruncateContent, ShouldBeFalse)
			So(customFormatter.config.IncludeMetadata, ShouldBeFalse)
			So(customFormatter.config.SortByConfidence, ShouldBeFalse)
			So(customFormatter.config.MinConfidenceDisplay, ShouldEqual, 0.5)
		})
	})
}

func TestRecallResponseFormatterFormat(t *testing.T) {
	Convey("Given a RecallResponseFormatter", t, func() {
		formatter := NewRecallResponseFormatter()

		Convey("When formatting a valid response", func() {
			response := &RecallResponse{
				Evidence: []Evidence{
					{
						Content:     "Test evidence content",
						Source:      "test_source",
						Confidence:  0.8,
						WhySelected: "High relevance score",
						RelationMap: map[string]string{"related_to": "entity1"},
						Provenance: ProvenanceInfo{
							Source:    "test_source",
							Timestamp: "2024-01-01T00:00:00Z",
							Version:   "1.0",
						},
					},
				},
				CommunityCards: []CommunityCard{
					{
						ID:          "community1",
						Title:       "Test Community",
						Summary:     "A test community",
						EntityCount: 5,
						Entities:    []string{"entity1", "entity2"},
					},
				},
				Conflicts: []ConflictInfo{
					{
						ID:          "conflict1",
						Type:        "confidence_mismatch",
						Description: "Different confidence levels",
						Severity:    "medium",
					},
				},
				RetrievalStats: RetrievalStats{
					QueryTime:       100,
					VectorResults:   1,
					GraphResults:    0,
					SearchResults:   0,
					FusionScore:     0.8,
					TotalCandidates: 1,
				},
				SelfCritique: "Results look good",
			}

			args := RecallArgs{
				Query: "test query",
			}

			result, err := formatter.Format(response, args)

			Convey("Then formatting should succeed", func() {
				So(err, ShouldBeNil)
				So(result.Evidence, ShouldHaveLength, 1)
				So(result.Evidence[0].Content, ShouldEqual, "Test evidence content")
				So(result.Evidence[0].Confidence, ShouldEqual, 0.8)
				So(result.CommunityCards, ShouldHaveLength, 1)
				So(result.Conflicts, ShouldHaveLength, 1)
				So(result.SelfCritique, ShouldContainSubstring, "test query")
			})
		})

		Convey("When formatting a nil response", func() {
			args := RecallArgs{Query: "test"}
			_, err := formatter.Format(nil, args)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "response cannot be nil")
			})
		})

		Convey("When formatting an empty response", func() {
			response := &RecallResponse{
				Evidence:       []Evidence{},
				CommunityCards: []CommunityCard{},
				Conflicts:      []ConflictInfo{},
				RetrievalStats: RetrievalStats{},
			}

			args := RecallArgs{Query: "test"}
			result, err := formatter.Format(response, args)

			Convey("Then it should handle empty response gracefully", func() {
				So(err, ShouldBeNil)
				So(result.Evidence, ShouldHaveLength, 0)
				So(result.CommunityCards, ShouldHaveLength, 0)
				So(result.Conflicts, ShouldHaveLength, 0)
			})
		})
	})
}

func TestRecallResponseFormatterBuildRecallResult(t *testing.T) {
	Convey("Given a RecallResponseFormatter", t, func() {
		formatter := NewRecallResponseFormatter()

		Convey("When building a recall result from components", func() {
			evidence := []Evidence{
				{
					Content:    "Test content",
					Confidence: 0.9,
				},
			}

			communityCards := []CommunityCard{
				{
					ID:    "community1",
					Title: "Test Community",
				},
			}

			conflicts := []ConflictInfo{
				{
					ID:   "conflict1",
					Type: "test_conflict",
				},
			}

			stats := RetrievalStats{
				QueryTime:     100,
				FusionScore:   0.8,
			}

			result := formatter.BuildRecallResult(evidence, communityCards, conflicts, stats)

			Convey("Then result should be properly built", func() {
				So(result.Evidence, ShouldHaveLength, 1)
				So(result.Evidence[0].Content, ShouldEqual, "Test content")
				So(result.CommunityCards, ShouldHaveLength, 1)
				So(result.Conflicts, ShouldHaveLength, 1)
				So(result.Stats.QueryTime, ShouldEqual, 100)
			})
		})
	})
}

func TestRecallResponseFormatterAddMetadata(t *testing.T) {
	Convey("Given a RecallResponseFormatter", t, func() {
		formatter := NewRecallResponseFormatter()

		Convey("When adding metadata to a result", func() {
			result := &RecallResult{
				Evidence: []Evidence{
					{
						Content:     "Test content",
						RelationMap: map[string]string{"existing": "value"},
					},
				},
				Stats: RetrievalStats{},
			}

			metadata := map[string]interface{}{
				"processing_time": 150 * time.Millisecond,
				"method":          "vector",
				"irrelevant_key":  "should_be_ignored",
			}

			formatter.AddMetadata(result, metadata)

			Convey("Then relevant metadata should be added", func() {
				So(result.Evidence[0].RelationMap, ShouldContainKey, "existing")
				So(result.Evidence[0].RelationMap, ShouldContainKey, "meta_processing_time")
				So(result.Evidence[0].RelationMap, ShouldContainKey, "meta_method")
				So(result.Stats.QueryTime, ShouldEqual, 150)
			})
		})

		Convey("When adding nil metadata", func() {
			result := &RecallResult{
				Evidence: []Evidence{
					{Content: "Test content"},
				},
			}

			originalEvidence := result.Evidence[0]
			formatter.AddMetadata(result, nil)

			Convey("Then result should remain unchanged", func() {
				So(result.Evidence[0], ShouldResemble, originalEvidence)
			})
		})
	})
}

func TestRecallResponseFormatterFormatEvidence(t *testing.T) {
	Convey("Given a RecallResponseFormatter", t, func() {
		formatter := NewRecallResponseFormatter()

		Convey("When formatting evidence with confidence sorting", func() {
			evidence := []Evidence{
				{Content: "Low confidence", Confidence: 0.3},
				{Content: "High confidence", Confidence: 0.9},
				{Content: "Medium confidence", Confidence: 0.6},
			}

			context := &FormattingContext{
				OriginalQuery: "test query",
			}

			formatted, err := formatter.formatEvidence(evidence, context)

			Convey("Then evidence should be sorted by confidence", func() {
				So(err, ShouldBeNil)
				So(formatted, ShouldHaveLength, 3)
				So(formatted[0].Content, ShouldEqual, "High confidence")
				So(formatted[1].Content, ShouldEqual, "Medium confidence")
				So(formatted[2].Content, ShouldEqual, "Low confidence")
			})
		})

		Convey("When formatting evidence with confidence filtering", func() {
			config := &RecallFormatterConfig{
				MinConfidenceDisplay: 0.5,
				SortByConfidence:     true,
			}
			formatter := NewRecallResponseFormatterWithConfig(config)

			evidence := []Evidence{
				{Content: "Low confidence", Confidence: 0.3},
				{Content: "High confidence", Confidence: 0.9},
				{Content: "Medium confidence", Confidence: 0.6},
			}

			context := &FormattingContext{}
			formatted, err := formatter.formatEvidence(evidence, context)

			Convey("Then low confidence evidence should be filtered out", func() {
				So(err, ShouldBeNil)
				So(formatted, ShouldHaveLength, 2)
				So(formatted[0].Content, ShouldEqual, "High confidence")
				So(formatted[1].Content, ShouldEqual, "Medium confidence")
			})
		})

		Convey("When formatting empty evidence", func() {
			evidence := []Evidence{}
			context := &FormattingContext{}

			formatted, err := formatter.formatEvidence(evidence, context)

			Convey("Then it should return empty slice", func() {
				So(err, ShouldBeNil)
				So(formatted, ShouldHaveLength, 0)
			})
		})
	})
}

func TestRecallResponseFormatterFormatEvidenceItem(t *testing.T) {
	Convey("Given a RecallResponseFormatter", t, func() {
		formatter := NewRecallResponseFormatter()

		Convey("When formatting evidence item with long content", func() {
			longContent := make([]byte, 600)
			for i := range longContent {
				longContent[i] = 'a'
			}

			evidence := Evidence{
				Content:     string(longContent),
				WhySelected: "Test reason",
				Provenance: ProvenanceInfo{
					Timestamp: "2024-01-01T00:00:00Z",
				},
				RelationMap: map[string]string{
					"test_key": "test_value",
					"":         "empty_key", // should be filtered
				},
			}

			context := &FormattingContext{
				OriginalQuery: "test query",
			}

			formatted := formatter.formatEvidenceItem(evidence, context)

			Convey("Then content should be truncated", func() {
				So(len(formatted.Content), ShouldEqual, 503) // 500 + "..."
				So(formatted.Content, ShouldEndWith, "...")
			})

			Convey("And why_selected should be enhanced", func() {
				So(formatted.WhySelected, ShouldContainSubstring, "test query")
			})

			Convey("And timestamp should be formatted", func() {
				So(formatted.Provenance.Timestamp, ShouldEqual, "2024-01-01 00:00:00")
			})

			Convey("And relation map should be cleaned", func() {
				So(formatted.RelationMap, ShouldContainKey, "Test Key")
				So(formatted.RelationMap, ShouldNotContainKey, "")
			})
		})

		Convey("When formatting evidence item with truncation disabled", func() {
			config := &RecallFormatterConfig{
				TruncateContent: false,
			}
			formatter := NewRecallResponseFormatterWithConfig(config)

			longContent := make([]byte, 600)
			for i := range longContent {
				longContent[i] = 'a'
			}

			evidence := Evidence{
				Content: string(longContent),
			}

			context := &FormattingContext{}
			formatted := formatter.formatEvidenceItem(evidence, context)

			Convey("Then content should not be truncated", func() {
				So(len(formatted.Content), ShouldEqual, 600)
				So(formatted.Content, ShouldNotEndWith, "...")
			})
		})
	})
}

func TestRecallResponseFormatterFormatCommunityCards(t *testing.T) {
	Convey("Given a RecallResponseFormatter", t, func() {
		formatter := NewRecallResponseFormatter()

		Convey("When formatting community cards", func() {
			cards := []CommunityCard{
				{
					ID:          "community1",
					Title:       "",
					Summary:     "Short summary",
					EntityCount: 3,
					Entities:    []string{"e1", "e2", "e3"},
				},
				{
					ID:          "community2",
					Title:       "Large Community",
					Summary:     string(make([]byte, 250)), // Long summary
					EntityCount: 15,
					Entities:    make([]string, 20), // Many entities
				},
			}

			// Fill long summary and entities
			longSummary := make([]byte, 250)
			for i := range longSummary {
				longSummary[i] = 'x'
			}
			cards[1].Summary = string(longSummary)
			for i := range cards[1].Entities {
				cards[1].Entities[i] = "entity" + string(rune('0'+i))
			}

			context := &FormattingContext{}
			formatted := formatter.formatCommunityCards(cards, context)

			Convey("Then cards should be formatted and sorted", func() {
				So(formatted, ShouldHaveLength, 2)
				// Should be sorted by entity count (descending)
				So(formatted[0].EntityCount, ShouldEqual, 15)
				So(formatted[1].EntityCount, ShouldEqual, 3)
			})

			Convey("And empty titles should be filled", func() {
				So(formatted[1].Title, ShouldEqual, "Community community1")
			})

			Convey("And long summaries should be truncated", func() {
				So(len(formatted[0].Summary), ShouldEqual, 203) // 200 + "..."
				So(formatted[0].Summary, ShouldEndWith, "...")
			})

			Convey("And entity lists should be limited", func() {
				So(formatted[0].Entities, ShouldHaveLength, 10)
			})
		})

		Convey("When formatting empty community cards", func() {
			cards := []CommunityCard{}
			context := &FormattingContext{}

			formatted := formatter.formatCommunityCards(cards, context)

			Convey("Then it should return empty slice", func() {
				So(formatted, ShouldHaveLength, 0)
			})
		})
	})
}

func TestRecallResponseFormatterFormatConflicts(t *testing.T) {
	Convey("Given a RecallResponseFormatter", t, func() {
		formatter := NewRecallResponseFormatter()

		Convey("When formatting conflicts", func() {
			conflicts := []ConflictInfo{
				{
					ID:          "conflict1",
					Type:        "type1",
					Description: "Low severity conflict",
					Severity:    "low",
				},
				{
					ID:          "conflict2",
					Type:        "type2",
					Description: "Critical conflict",
					Severity:    "critical",
				},
				{
					ID:          "conflict3",
					Type:        "type3",
					Description: "Medium conflict",
					Severity:    "medium",
				},
			}

			context := &FormattingContext{
				OriginalQuery: "test query",
			}

			formatted := formatter.formatConflicts(conflicts, context)

			Convey("Then conflicts should be sorted by severity", func() {
				So(formatted, ShouldHaveLength, 3)
				So(formatted[0].Severity, ShouldEqual, "critical")
				So(formatted[1].Severity, ShouldEqual, "medium")
				So(formatted[2].Severity, ShouldEqual, "low")
			})

			Convey("And descriptions should include query context", func() {
				for _, conflict := range formatted {
					So(conflict.Description, ShouldContainSubstring, "test query")
				}
			})
		})

		Convey("When formatting empty conflicts", func() {
			conflicts := []ConflictInfo{}
			context := &FormattingContext{}

			formatted := formatter.formatConflicts(conflicts, context)

			Convey("Then it should return empty slice", func() {
				So(formatted, ShouldHaveLength, 0)
			})
		})
	})
}

func TestRecallResponseFormatterFormatRetrievalStats(t *testing.T) {
	Convey("Given a RecallResponseFormatter", t, func() {
		formatter := NewRecallResponseFormatter()

		Convey("When formatting stats with negative values", func() {
			stats := RetrievalStats{
				QueryTime:       100,
				VectorResults:   -1, // Invalid
				GraphResults:    5,
				SearchResults:   -2, // Invalid
				FusionScore:     1.5, // Invalid (> 1)
				TotalCandidates: -3,  // Invalid
			}

			context := &FormattingContext{}
			formatted := formatter.formatRetrievalStats(stats, context)

			Convey("Then negative values should be corrected", func() {
				So(formatted.QueryTime, ShouldEqual, 100)
				So(formatted.VectorResults, ShouldEqual, 0)
				So(formatted.GraphResults, ShouldEqual, 5)
				So(formatted.SearchResults, ShouldEqual, 0)
				So(formatted.FusionScore, ShouldEqual, 1.0)
				So(formatted.TotalCandidates, ShouldEqual, 0)
			})
		})

		Convey("When formatting stats with valid values", func() {
			stats := RetrievalStats{
				QueryTime:       150,
				VectorResults:   10,
				GraphResults:    5,
				SearchResults:   8,
				FusionScore:     0.75,
				TotalCandidates: 23,
			}

			context := &FormattingContext{}
			formatted := formatter.formatRetrievalStats(stats, context)

			Convey("Then values should remain unchanged", func() {
				So(formatted.QueryTime, ShouldEqual, 150)
				So(formatted.VectorResults, ShouldEqual, 10)
				So(formatted.GraphResults, ShouldEqual, 5)
				So(formatted.SearchResults, ShouldEqual, 8)
				So(formatted.FusionScore, ShouldEqual, 0.75)
				So(formatted.TotalCandidates, ShouldEqual, 23)
			})
		})
	})
}

func TestRecallResponseFormatterFormatForDisplay(t *testing.T) {
	Convey("Given a RecallResponseFormatter", t, func() {
		formatter := NewRecallResponseFormatter()

		Convey("When formatting result for display", func() {
			result := RecallResult{
				Evidence: []Evidence{
					{
						Content:     "Test evidence",
						Source:      "test_source",
						Confidence:  0.8,
						WhySelected: "High relevance",
						RelationMap: map[string]string{
							"related_to": "entity1",
						},
					},
				},
				Conflicts: []ConflictInfo{
					{
						Type:        "confidence_mismatch",
						Severity:    "medium",
						Description: "Different confidence levels",
					},
				},
				Stats: RetrievalStats{
					QueryTime:     100,
					VectorResults: 1,
					FusionScore:   0.8,
				},
				SelfCritique: "Results look good",
			}

			display := formatter.FormatForDisplay(result)

			Convey("Then display should be human-readable", func() {
				So(display, ShouldContainSubstring, "=== Memory Recall Results ===")
				So(display, ShouldContainSubstring, "Found 1 evidence items")
				So(display, ShouldContainSubstring, "Evidence 1 (Confidence: 0.80)")
				So(display, ShouldContainSubstring, "Test evidence")
				So(display, ShouldContainSubstring, "=== Conflicts Detected ===")
				So(display, ShouldContainSubstring, "confidence_mismatch")
				So(display, ShouldContainSubstring, "=== Retrieval Statistics ===")
				So(display, ShouldContainSubstring, "Query Time: 100ms")
				So(display, ShouldContainSubstring, "=== Self-Critique ===")
				So(display, ShouldContainSubstring, "Results look good")
			})
		})

		Convey("When formatting result with no conflicts or critique", func() {
			result := RecallResult{
				Evidence: []Evidence{
					{Content: "Test evidence", Confidence: 0.8},
				},
				Stats: RetrievalStats{QueryTime: 100},
			}

			display := formatter.FormatForDisplay(result)

			Convey("Then optional sections should be omitted", func() {
				So(display, ShouldContainSubstring, "=== Memory Recall Results ===")
				So(display, ShouldContainSubstring, "=== Retrieval Statistics ===")
				So(display, ShouldNotContainSubstring, "=== Conflicts Detected ===")
				So(display, ShouldNotContainSubstring, "=== Self-Critique ===")
			})
		})
	})
}

func TestRecallResponseFormatterConfiguration(t *testing.T) {
	Convey("Given a RecallResponseFormatter", t, func() {
		formatter := NewRecallResponseFormatter()

		Convey("When getting configuration", func() {
			config := formatter.GetConfig()

			Convey("Then it should return the current config", func() {
				So(config, ShouldNotBeNil)
				So(config.MaxEvidenceLength, ShouldEqual, 500)
				So(config.TruncateContent, ShouldBeTrue)
				So(config.IncludeMetadata, ShouldBeTrue)
			})
		})

		Convey("When updating configuration", func() {
			newConfig := &RecallFormatterConfig{
				MaxEvidenceLength:    200,
				TruncateContent:      false,
				IncludeMetadata:      false,
				SortByConfidence:     false,
			}

			formatter.UpdateConfig(newConfig)
			config := formatter.GetConfig()

			Convey("Then the config should be updated", func() {
				So(config.MaxEvidenceLength, ShouldEqual, 200)
				So(config.TruncateContent, ShouldBeFalse)
				So(config.IncludeMetadata, ShouldBeFalse)
				So(config.SortByConfidence, ShouldBeFalse)
			})
		})

		Convey("When updating with nil config", func() {
			originalConfig := formatter.GetConfig()
			formatter.UpdateConfig(nil)
			currentConfig := formatter.GetConfig()

			Convey("Then the config should remain unchanged", func() {
				So(currentConfig, ShouldEqual, originalConfig)
			})
		})
	})
}