package main

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWriteResponseFormatter(t *testing.T) {
	Convey("WriteResponseFormatter", t, func() {
		formatter := NewWriteResponseFormatter()

		Convey("NewWriteResponseFormatter", func() {
			Convey("Should create formatter with default config", func() {
				So(formatter, ShouldNotBeNil)
				So(formatter.config, ShouldNotBeNil)
				So(formatter.config.IncludeMetadata, ShouldBeTrue)
				So(formatter.config.FormatTimestamps, ShouldBeTrue)
				So(formatter.config.TruncateIDs, ShouldBeTrue)
				So(formatter.config.MaxIDLength, ShouldEqual, 16)
				So(formatter.config.SortConflictsBySeverity, ShouldBeTrue)
			})

			Convey("Should create formatter with custom config", func() {
				config := &WriteFormatterConfig{
					IncludeMetadata:         false,
					FormatTimestamps:        false,
					TruncateIDs:             false,
					MaxIDLength:             32,
					SortConflictsBySeverity: false,
					IncludeProcessingStats:  false,
				}
				customFormatter := NewWriteResponseFormatterWithConfig(config)

				So(customFormatter.config.IncludeMetadata, ShouldBeFalse)
				So(customFormatter.config.FormatTimestamps, ShouldBeFalse)
				So(customFormatter.config.TruncateIDs, ShouldBeFalse)
				So(customFormatter.config.MaxIDLength, ShouldEqual, 32)
				So(customFormatter.config.SortConflictsBySeverity, ShouldBeFalse)
			})
		})

		Convey("Format", func() {
			Convey("Should format valid response", func() {
				response := &WriteResponse{
					MemoryID:       "test_memory_12345",
					CandidateCount: 3,
					ConflictsFound: []ConflictInfo{
						{
							ID:          "conflict_1",
							Type:        "duplicate_content",
							Description: "Similar content found",
							Severity:    "low",
						},
					},
					EntitiesLinked: []string{"entity_1", "entity_2"},
					ProvenanceID:   "prov_12345",
				}

				args := WriteArgs{
					Content: "Test content",
					Source:  "test_source",
				}

				result, err := formatter.Format(response, args)

				So(err, ShouldBeNil)
				So(result.MemoryID, ShouldEqual, "test_memory_1234...")
				So(result.CandidateCount, ShouldEqual, 3)
				So(len(result.ConflictsFound), ShouldEqual, 1)
				So(len(result.EntitiesLinked), ShouldEqual, 2)
				So(result.ProvenanceID, ShouldEqual, "prov_12345")
			})

			Convey("Should handle nil response", func() {
				args := WriteArgs{Content: "Test", Source: "test"}

				_, err := formatter.Format(nil, args)

				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "response cannot be nil")
			})

			Convey("Should truncate long IDs when configured", func() {
				response := &WriteResponse{
					MemoryID:     "very_long_memory_id_that_exceeds_limit",
					ProvenanceID: "very_long_provenance_id_that_exceeds_limit",
					EntitiesLinked: []string{
						"very_long_entity_id_that_exceeds_limit",
					},
				}

				args := WriteArgs{Content: "Test", Source: "test"}
				result, err := formatter.Format(response, args)

				So(err, ShouldBeNil)
				So(len(result.MemoryID), ShouldBeLessThanOrEqualTo, 19) // 16 + "..."
				So(result.MemoryID, ShouldEndWith, "...")
				So(len(result.ProvenanceID), ShouldBeLessThanOrEqualTo, 19)
				So(result.ProvenanceID, ShouldEndWith, "...")
			})

			Convey("Should not truncate short IDs", func() {
				response := &WriteResponse{
					MemoryID:       "short_id",
					ProvenanceID:   "short_prov",
					EntitiesLinked: []string{"short_entity"},
				}

				args := WriteArgs{Content: "Test", Source: "test"}
				result, err := formatter.Format(response, args)

				So(err, ShouldBeNil)
				So(result.MemoryID, ShouldEqual, "short_id")
				So(result.ProvenanceID, ShouldEqual, "short_prov")
				So(result.EntitiesLinked[0], ShouldEqual, "short_entity")
			})
		})

		Convey("BuildWriteResult", func() {
			Convey("Should build result from components", func() {
				memoryID := "test_memory_123"
				candidateCount := 2
				entitiesLinked := []string{"entity_1", "entity_2", "entity_1"} // Duplicate
				provenanceID := "prov_123"

				result := formatter.BuildWriteResult(memoryID, candidateCount, entitiesLinked, provenanceID)

				So(result.MemoryID, ShouldEqual, memoryID)
				So(result.CandidateCount, ShouldEqual, candidateCount)
				So(len(result.EntitiesLinked), ShouldEqual, 2) // Duplicates removed
				So(result.ProvenanceID, ShouldEqual, provenanceID)
				So(len(result.ConflictsFound), ShouldEqual, 0) // Empty by default
			})

			Convey("Should sort entities alphabetically", func() {
				entitiesLinked := []string{"zebra", "alpha", "beta"}

				result := formatter.BuildWriteResult("test", 1, entitiesLinked, "prov")

				So(result.EntitiesLinked[0], ShouldEqual, "alpha")
				So(result.EntitiesLinked[1], ShouldEqual, "beta")
				So(result.EntitiesLinked[2], ShouldEqual, "zebra")
			})
		})

		Convey("AddConflictInfo", func() {
			Convey("Should add conflicts to result", func() {
				result := &WriteResult{
					MemoryID:       "test",
					ConflictsFound: []ConflictInfo{},
				}

				conflicts := []ConflictInfo{
					{
						ID:          "conflict_1",
						Type:        "duplicate",
						Description: "Duplicate content",
						Severity:    "low",
					},
					{
						ID:          "conflict_2",
						Type:        "entity_conflict",
						Description: "Entity conflict",
						Severity:    "high",
					},
				}

				formatter.AddConflictInfo(result, conflicts)

				So(len(result.ConflictsFound), ShouldEqual, 2)
				// Should be sorted by severity (high first)
				So(result.ConflictsFound[0].Severity, ShouldEqual, "high")
				So(result.ConflictsFound[1].Severity, ShouldEqual, "low")
			})

			Convey("Should handle nil conflicts", func() {
				result := &WriteResult{
					MemoryID:       "test",
					ConflictsFound: []ConflictInfo{},
				}

				formatter.AddConflictInfo(result, nil)

				So(len(result.ConflictsFound), ShouldEqual, 0)
			})
		})

		Convey("Conflict Formatting", func() {
			Convey("Should sort conflicts by severity", func() {
				conflicts := []ConflictInfo{
					{ID: "1", Severity: "low"},
					{ID: "2", Severity: "critical"},
					{ID: "3", Severity: "medium"},
					{ID: "4", Severity: "high"},
				}

				context := &WriteFormattingContext{}
				formatted := formatter.formatConflicts(conflicts, context)

				So(formatted[0].Severity, ShouldEqual, "critical")
				So(formatted[1].Severity, ShouldEqual, "high")
				So(formatted[2].Severity, ShouldEqual, "medium")
				So(formatted[3].Severity, ShouldEqual, "low")
			})

			Convey("Should normalize severity levels", func() {
				conflicts := []ConflictInfo{
					{ID: "1", Severity: "ERROR"},
					{ID: "2", Severity: "warning"},
					{ID: "3", Severity: "info"},
					{ID: "4", Severity: "unknown"},
				}

				context := &WriteFormattingContext{}
				formatted := formatter.formatConflicts(conflicts, context)

				// Conflicts are sorted by severity (high, medium, low), so order changes
				So(formatted[0].Severity, ShouldEqual, "high")   // ERROR -> high (highest priority)
				So(formatted[1].Severity, ShouldEqual, "medium") // warning -> medium
				So(formatted[2].Severity, ShouldEqual, "medium") // unknown -> medium (default)
				So(formatted[3].Severity, ShouldEqual, "low")    // info -> low (lowest priority)
			})

			Convey("Should enhance conflict descriptions with context", func() {
				conflicts := []ConflictInfo{
					{
						ID:          "1",
						Description: "Duplicate content found",
						Severity:    "low",
					},
				}

				context := &WriteFormattingContext{
					OriginalContent: "This is the original content that was submitted",
				}
				formatted := formatter.formatConflicts(conflicts, context)

				So(formatted[0].Description, ShouldContainSubstring, "Duplicate content found")
				So(formatted[0].Description, ShouldContainSubstring, "content:")
				So(formatted[0].Description, ShouldContainSubstring, "This is the original")
			})
		})

		Convey("Entity Formatting", func() {
			Convey("Should remove duplicate entities", func() {
				entities := []string{"entity_1", "entity_2", "entity_1", "entity_3", "entity_2"}
				context := &WriteFormattingContext{}

				formatted := formatter.formatEntitiesLinked(entities, context)

				So(len(formatted), ShouldEqual, 3)
				// Should be sorted alphabetically
				So(formatted[0], ShouldEqual, "entity_1")
				So(formatted[1], ShouldEqual, "entity_2")
				So(formatted[2], ShouldEqual, "entity_3")
			})

			Convey("Should handle empty entities list", func() {
				entities := []string{}
				context := &WriteFormattingContext{}

				formatted := formatter.formatEntitiesLinked(entities, context)

				So(len(formatted), ShouldEqual, 0)
			})

			Convey("Should truncate long entity IDs", func() {
				entities := []string{"very_long_entity_id_that_exceeds_the_maximum_length_limit"}
				context := &WriteFormattingContext{}

				formatted := formatter.formatEntitiesLinked(entities, context)

				So(len(formatted), ShouldEqual, 1)
				So(len(formatted[0]), ShouldBeLessThanOrEqualTo, 19) // 16 + "..."
				So(formatted[0], ShouldEndWith, "...")
			})
		})

		Convey("Configuration", func() {
			Convey("GetConfig should return current config", func() {
				config := formatter.GetConfig()
				So(config, ShouldNotBeNil)
				So(config.MaxIDLength, ShouldEqual, 16)
			})

			Convey("UpdateConfig should update configuration", func() {
				newConfig := &WriteFormatterConfig{
					MaxIDLength: 32,
					TruncateIDs: false,
				}

				formatter.UpdateConfig(newConfig)
				config := formatter.GetConfig()

				So(config.MaxIDLength, ShouldEqual, 32)
				So(config.TruncateIDs, ShouldBeFalse)
			})
		})

		Convey("FormatForDisplay", func() {
			Convey("Should create human-readable display", func() {
				result := WriteResult{
					MemoryID:       "test_memory_123",
					CandidateCount: 2,
					EntitiesLinked: []string{"entity_1", "entity_2"},
					ProvenanceID:   "prov_123",
					ConflictsFound: []ConflictInfo{
						{
							ID:          "conflict_1",
							Type:        "duplicate",
							Description: "Duplicate content found",
							Severity:    "low",
						},
					},
				}

				display := formatter.FormatForDisplay(result)

				So(display, ShouldContainSubstring, "Memory Write Results")
				So(display, ShouldContainSubstring, "test_memory_123")
				So(display, ShouldContainSubstring, "Candidates Created: 2")
				So(display, ShouldContainSubstring, "Entities Linked: 2")
				So(display, ShouldContainSubstring, "entity_1")
				So(display, ShouldContainSubstring, "entity_2")
				So(display, ShouldContainSubstring, "Conflicts Detected")
				So(display, ShouldContainSubstring, "duplicate")
			})

			Convey("Should handle result with no conflicts", func() {
				result := WriteResult{
					MemoryID:       "test_memory_123",
					CandidateCount: 1,
					EntitiesLinked: []string{},
					ProvenanceID:   "prov_123",
					ConflictsFound: []ConflictInfo{},
				}

				display := formatter.FormatForDisplay(result)

				So(display, ShouldContainSubstring, "Memory Write Results")
				So(display, ShouldNotContainSubstring, "Conflicts Detected")
				So(display, ShouldNotContainSubstring, "Linked Entities")
			})
		})

		Convey("FormatSummary", func() {
			Convey("Should create brief summary", func() {
				result := WriteResult{
					MemoryID:       "test_123",
					CandidateCount: 3,
					EntitiesLinked: []string{"entity_1", "entity_2"},
					ConflictsFound: []ConflictInfo{
						{ID: "conflict_1", Severity: "low"},
					},
				}

				summary := formatter.FormatSummary(result)

				So(summary, ShouldContainSubstring, "Memory stored")
				So(summary, ShouldContainSubstring, "test_123")
				So(summary, ShouldContainSubstring, "3 candidates created")
				So(summary, ShouldContainSubstring, "2 entities linked")
				So(summary, ShouldContainSubstring, "1 conflicts detected")
			})

			Convey("Should handle minimal result", func() {
				result := WriteResult{
					MemoryID:       "test_123",
					CandidateCount: 1,
					EntitiesLinked: []string{},
					ConflictsFound: []ConflictInfo{},
				}

				summary := formatter.FormatSummary(result)

				So(summary, ShouldEqual, "Memory stored (ID: test_123)")
			})
		})

		Convey("ValidateResult", func() {
			Convey("Should validate correct result", func() {
				result := WriteResult{
					MemoryID:       "test_123",
					CandidateCount: 1,
					EntitiesLinked: []string{"entity_1"},
					ProvenanceID:   "prov_123",
					ConflictsFound: []ConflictInfo{
						{
							ID:          "conflict_1",
							Type:        "duplicate",
							Description: "Test conflict",
							Severity:    "low",
						},
					},
				}

				err := formatter.ValidateResult(result)
				So(err, ShouldBeNil)
			})

			Convey("Should reject result with empty memory ID", func() {
				result := WriteResult{
					MemoryID:       "",
					CandidateCount: 1,
					EntitiesLinked: []string{},
					ConflictsFound: []ConflictInfo{},
				}

				err := formatter.ValidateResult(result)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "memory ID cannot be empty")
			})

			Convey("Should reject result with negative candidate count", func() {
				result := WriteResult{
					MemoryID:       "test_123",
					CandidateCount: -1,
					EntitiesLinked: []string{},
					ConflictsFound: []ConflictInfo{},
				}

				err := formatter.ValidateResult(result)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "candidate count cannot be negative")
			})

			Convey("Should reject result with nil entities", func() {
				result := WriteResult{
					MemoryID:       "test_123",
					CandidateCount: 1,
					EntitiesLinked: nil,
					ConflictsFound: []ConflictInfo{},
				}

				err := formatter.ValidateResult(result)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "entities linked cannot be nil")
			})

			Convey("Should reject conflicts with empty fields", func() {
				result := WriteResult{
					MemoryID:       "test_123",
					CandidateCount: 1,
					EntitiesLinked: []string{},
					ConflictsFound: []ConflictInfo{
						{
							ID:          "", // Empty ID
							Type:        "duplicate",
							Description: "Test conflict",
							Severity:    "low",
						},
					},
				}

				err := formatter.ValidateResult(result)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "conflict at index 0 has empty ID")
			})
		})

		Convey("Utility Methods", func() {
			Convey("CreateConflictInfo should create valid conflict", func() {
				conflict := formatter.CreateConflictInfo(
					"test_id",
					"duplicate",
					"Test description",
					"HIGH",
					[]string{"id1", "id2"},
				)

				So(conflict.ID, ShouldEqual, "test_id")
				So(conflict.Type, ShouldEqual, "duplicate")
				So(conflict.Description, ShouldEqual, "Test description")
				So(conflict.Severity, ShouldEqual, "high") // Normalized
				So(len(conflict.ConflictingIDs), ShouldEqual, 2)
			})

			Convey("MergeConflicts should merge and deduplicate", func() {
				conflicts1 := []ConflictInfo{
					{ID: "1", Severity: "low"},
					{ID: "2", Severity: "high"},
				}
				conflicts2 := []ConflictInfo{
					{ID: "2", Severity: "high"}, // Duplicate
					{ID: "3", Severity: "medium"},
				}

				merged := formatter.MergeConflicts(conflicts1, conflicts2)

				So(len(merged), ShouldEqual, 3)
				// Should be sorted by severity
				So(merged[0].Severity, ShouldEqual, "high")
				So(merged[1].Severity, ShouldEqual, "medium")
				So(merged[2].Severity, ShouldEqual, "low")
			})
		})
	})
}

func TestWriteFormatterEdgeCases(t *testing.T) {
	Convey("WriteFormatter Edge Cases", t, func() {
		formatter := NewWriteResponseFormatter()

		Convey("Should handle empty strings gracefully", func() {
			response := &WriteResponse{
				MemoryID:       "",
				ProvenanceID:   "",
				EntitiesLinked: []string{"", "valid_entity", ""},
			}

			args := WriteArgs{Content: "Test", Source: "test"}
			result, err := formatter.Format(response, args)

			So(err, ShouldBeNil)
			So(result.MemoryID, ShouldEqual, "")
			So(result.ProvenanceID, ShouldEqual, "")
			// Empty entities should be handled gracefully
			So(len(result.EntitiesLinked), ShouldEqual, 1)
			So(result.EntitiesLinked[0], ShouldEqual, "valid_entity")
		})

		Convey("Should handle very long content in context", func() {
			longContent := strings.Repeat("Very long content ", 100)

			conflicts := []ConflictInfo{
				{
					ID:          "1",
					Description: "Test conflict",
					Severity:    "low",
				},
			}

			context := &WriteFormattingContext{
				OriginalContent: longContent,
			}
			formatted := formatter.formatConflicts(conflicts, context)

			So(len(formatted), ShouldEqual, 1)
			// Description should be enhanced but content should be truncated
			So(formatted[0].Description, ShouldContainSubstring, "content:")
			So(len(formatted[0].Description), ShouldBeLessThan, len(longContent)+100)
		})

		Convey("Should handle conflicts with empty conflicting IDs", func() {
			conflicts := []ConflictInfo{
				{
					ID:             "1",
					Type:           "test",
					Description:    "Test conflict",
					Severity:       "low",
					ConflictingIDs: []string{},
				},
			}

			context := &WriteFormattingContext{}
			formatted := formatter.formatConflicts(conflicts, context)

			So(len(formatted), ShouldEqual, 1)
			So(len(formatted[0].ConflictingIDs), ShouldEqual, 0)
		})

		Convey("Should handle unknown severity levels", func() {
			severityTests := []struct {
				input    string
				expected string
			}{
				{"CRITICAL", "critical"},
				{"High", "high"},
				{"MEDIUM", "medium"},
				{"low", "low"},
				{"error", "high"},
				{"warning", "medium"},
				{"info", "low"},
				{"unknown", "medium"},
				{"", "medium"},
			}

			for _, test := range severityTests {
				normalized := formatter.normalizeSeverity(test.input)
				So(normalized, ShouldEqual, test.expected)
			}
		})

		Convey("Should handle disabled truncation", func() {
			config := &WriteFormatterConfig{
				TruncateIDs: false,
				MaxIDLength: 16,
			}
			customFormatter := NewWriteResponseFormatterWithConfig(config)

			longID := "very_long_id_that_would_normally_be_truncated_but_should_not_be"
			formatted := customFormatter.formatMemoryID(longID)

			So(formatted, ShouldEqual, longID)
		})
	})
}

func BenchmarkWriteFormatter(b *testing.B) {
	formatter := NewWriteResponseFormatter()
	response := &WriteResponse{
		MemoryID:       "test_memory_12345",
		CandidateCount: 3,
		ConflictsFound: []ConflictInfo{
			{ID: "1", Type: "duplicate", Description: "Test conflict", Severity: "low"},
			{ID: "2", Type: "entity", Description: "Entity conflict", Severity: "high"},
		},
		EntitiesLinked: []string{"entity_1", "entity_2", "entity_3"},
		ProvenanceID:   "prov_12345",
	}
	args := WriteArgs{Content: "Test content", Source: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := formatter.Format(response, args)
		if err != nil {
			b.Fatalf("Format failed: %v", err)
		}
	}
}

func BenchmarkWriteFormatterConflictSorting(b *testing.B) {
	formatter := NewWriteResponseFormatter()

	// Create many conflicts with random severities
	conflicts := make([]ConflictInfo, 100)
	severities := []string{"low", "medium", "high", "critical"}
	for i := range conflicts {
		conflicts[i] = ConflictInfo{
			ID:       string(rune('a' + i%26)),
			Severity: severities[i%4],
		}
	}

	context := &WriteFormattingContext{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.formatConflicts(conflicts, context)
	}
}
