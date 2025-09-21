package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEvidenceAssembler(t *testing.T) {
	Convey("Given an EvidenceAssembler", t, func() {
		assembler := NewEvidenceAssembler()

		Convey("When creating with default config", func() {
			So(assembler, ShouldNotBeNil)
			So(assembler.config.MaxEvidenceItems, ShouldEqual, 50)
			So(assembler.config.MinConfidence, ShouldEqual, 0.1)
			So(assembler.config.RequireProvenance, ShouldBeTrue)
			So(assembler.config.IncludeRelationMaps, ShouldBeTrue)
			So(assembler.config.ValidateEvidence, ShouldBeTrue)
		})

		Convey("When creating with custom config", func() {
			config := &EvidenceAssemblerConfig{
				MaxEvidenceItems:  25,
				MinConfidence:     0.3,
				RequireProvenance: false,
				MaxContentLength:  1000,
			}
			customAssembler := NewEvidenceAssemblerWithConfig(config)

			So(customAssembler.config.MaxEvidenceItems, ShouldEqual, 25)
			So(customAssembler.config.MinConfidence, ShouldEqual, 0.3)
			So(customAssembler.config.RequireProvenance, ShouldBeFalse)
			So(customAssembler.config.MaxContentLength, ShouldEqual, 1000)
		})
	})
}

func TestEvidenceAssembler_Assemble(t *testing.T) {
	Convey("Given an EvidenceAssembler and assembly inputs", t, func() {
		assembler := NewEvidenceAssembler()
		ctx := context.Background()

		Convey("When assembling empty inputs", func() {
			inputs := []AssemblyInput{}
			context := &AssemblyContext{Query: "test query"}

			response, err := assembler.Assemble(ctx, inputs, context)

			So(err, ShouldBeNil)
			So(response, ShouldNotBeNil)
			So(response.TotalEvidence, ShouldEqual, 0)
			So(len(response.Evidence), ShouldEqual, 0)
		})

		Convey("When assembling valid inputs", func() {
			now := time.Now()
			inputs := []AssemblyInput{
				{
					ID:           "doc1",
					Content:      "This is a comprehensive document about machine learning algorithms",
					Score:        0.9,
					Source:       "academic_paper",
					Timestamp:    now.Add(-1 * time.Hour),
					MatchedTerms: []string{"machine", "learning"},
					Metadata:     map[string]interface{}{"author": "Dr. Smith", "category": "AI"},
				},
				{
					ID:        "doc2",
					Content:   "Short document",
					Score:     0.7,
					Source:    "blog",
					Timestamp: now.Add(-24 * time.Hour),
				},
			}
			context := &AssemblyContext{
				Query:           "machine learning",
				QueryTerms:      []string{"machine", "learning"},
				RequestTime:     now,
				RetrievalMethod: "vector",
			}

			response, err := assembler.Assemble(ctx, inputs, context)

			So(err, ShouldBeNil)
			So(response.TotalEvidence, ShouldEqual, 2)
			So(len(response.Evidence), ShouldEqual, 2)

			// Check that evidence is sorted by confidence (descending)
			So(response.Evidence[0].Confidence, ShouldBeGreaterThanOrEqualTo, response.Evidence[1].Confidence)

			// Check first evidence item
			evidence1 := response.Evidence[0]
			So(evidence1.Content, ShouldEqual, "This is a comprehensive document about machine learning algorithms")
			So(evidence1.Source, ShouldEqual, "academic_paper")
			So(evidence1.Confidence, ShouldEqual, 0.9)
			So(evidence1.WhySelected, ShouldContainSubstring, "high relevance score")
			So(evidence1.WhySelected, ShouldContainSubstring, "machine, learning")
			So(evidence1.Provenance.Source, ShouldEqual, "academic_paper")
			So(len(evidence1.RelationMap), ShouldBeGreaterThan, 0)

			// Check assembly stats
			So(response.AssemblyStats.InputCount, ShouldEqual, 2)
			So(response.AssemblyStats.ValidatedCount, ShouldEqual, 2)
			So(response.AssemblyStats.ProvenanceCount, ShouldEqual, 2)
			So(response.AssemblyStats.AssemblyTime, ShouldBeGreaterThan, 0)
		})

		Convey("When assembling with confidence filtering", func() {
			config := &EvidenceAssemblerConfig{
				MinConfidence: 0.8,
			}
			assembler := NewEvidenceAssemblerWithConfig(config)

			inputs := []AssemblyInput{
				{ID: "doc1", Content: "High confidence doc", Score: 0.9, Source: "source1"},
				{ID: "doc2", Content: "Low confidence doc", Score: 0.5, Source: "source2"},
			}
			context := &AssemblyContext{Query: "test"}

			response, err := assembler.Assemble(ctx, inputs, context)

			So(err, ShouldBeNil)
			So(response.TotalEvidence, ShouldEqual, 1)
			So(response.Evidence[0].Confidence, ShouldEqual, 0.9)
		})

		Convey("When assembling with deduplication", func() {
			inputs := []AssemblyInput{
				{ID: "doc1", Content: "This is a unique document about AI", Score: 0.9, Source: "source1"},
				{ID: "doc2", Content: "This is a unique document about AI", Score: 0.8, Source: "source2"}, // Duplicate content
				{ID: "doc3", Content: "This is a different document about ML", Score: 0.7, Source: "source3"},
			}
			context := &AssemblyContext{Query: "AI"}

			response, err := assembler.Assemble(ctx, inputs, context)

			So(err, ShouldBeNil)
			So(response.TotalEvidence, ShouldEqual, 2) // One duplicate removed
			So(response.AssemblyStats.DeduplicatedCount, ShouldEqual, 1)
		})

		Convey("When assembling with max items limit", func() {
			config := &EvidenceAssemblerConfig{
				MaxEvidenceItems: 2,
			}
			assembler := NewEvidenceAssemblerWithConfig(config)

			inputs := make([]AssemblyInput, 5)
			for i := 0; i < 5; i++ {
				inputs[i] = AssemblyInput{
					ID:      fmt.Sprintf("doc%d", i),
					Content: fmt.Sprintf("Document %d content", i),
					Score:   0.8 - float64(i)*0.1,
					Source:  "source",
				}
			}
			context := &AssemblyContext{Query: "test"}

			response, err := assembler.Assemble(ctx, inputs, context)

			So(err, ShouldBeNil)
			So(response.TotalEvidence, ShouldEqual, 2)
		})
	})
}

func TestEvidenceAssembler_AddProvenance(t *testing.T) {
	Convey("Given an EvidenceAssembler", t, func() {
		assembler := NewEvidenceAssembler()

		Convey("When adding provenance to evidence", func() {
			evidence := &Evidence{}
			input := AssemblyInput{
				Source:    "test_source",
				Timestamp: time.Now(),
				Metadata:  map[string]interface{}{"version": "2.0", "author": "John Doe"},
			}
			context := &AssemblyContext{UserID: "user123"}

			assembler.AddProvenance(evidence, input, context)

			So(evidence.Provenance.Source, ShouldEqual, "test_source")
			So(evidence.Provenance.Version, ShouldEqual, "2.0")
			So(evidence.Provenance.UserID, ShouldEqual, "user123")
			So(evidence.Provenance.Timestamp, ShouldNotBeEmpty)
		})

		Convey("When adding provenance with metadata author", func() {
			evidence := &Evidence{}
			input := AssemblyInput{
				Source:    "test_source",
				Timestamp: time.Now(),
				Metadata:  map[string]interface{}{"author": "Jane Smith"},
			}
			context := &AssemblyContext{} // No UserID

			assembler.AddProvenance(evidence, input, context)

			So(evidence.Provenance.UserID, ShouldEqual, "Jane Smith")
		})
	})
}

func TestEvidenceAssembler_ValidateEvidence(t *testing.T) {
	Convey("Given an EvidenceAssembler", t, func() {
		assembler := NewEvidenceAssembler()

		Convey("When validating valid evidence", func() {
			evidence := &Evidence{
				Content:    "Valid content for testing",
				Source:     "valid_source",
				Confidence: 0.8,
				Provenance: ProvenanceInfo{
					Source:    "valid_source",
					Timestamp: time.Now().Format(time.RFC3339),
				},
			}

			err := assembler.ValidateEvidence(evidence)
			So(err, ShouldBeNil)
		})

		Convey("When validating nil evidence", func() {
			err := assembler.ValidateEvidence(nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "evidence cannot be nil")
		})

		Convey("When validating evidence with empty content", func() {
			evidence := &Evidence{
				Content: "",
				Source:  "source",
			}

			err := assembler.ValidateEvidence(evidence)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "content cannot be empty")
		})

		Convey("When validating evidence with empty source", func() {
			evidence := &Evidence{
				Content: "Valid content",
				Source:  "",
			}

			err := assembler.ValidateEvidence(evidence)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "source cannot be empty")
		})

		Convey("When validating evidence with invalid confidence", func() {
			evidence := &Evidence{
				Content:    "Valid content",
				Source:     "valid_source",
				Confidence: 1.5, // Invalid confidence > 1
			}

			err := assembler.ValidateEvidence(evidence)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "confidence must be between 0 and 1")
		})

		Convey("When validating evidence with missing provenance", func() {
			evidence := &Evidence{
				Content:    "Valid content",
				Source:     "valid_source",
				Confidence: 0.8,
				Provenance: ProvenanceInfo{}, // Empty provenance
			}

			err := assembler.ValidateEvidence(evidence)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "provenance source is required")
		})

		Convey("When validating evidence with content too long", func() {
			config := &EvidenceAssemblerConfig{
				MaxContentLength: 50,
			}
			assembler := NewEvidenceAssemblerWithConfig(config)

			evidence := &Evidence{
				Content:    strings.Repeat("This is a very long content string. ", 10), // > 50 chars
				Source:     "valid_source",
				Confidence: 0.8,
			}

			err := assembler.ValidateEvidence(evidence)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "exceeds maximum length")
		})
	})
}

func TestEvidenceAssembler_WhySelected(t *testing.T) {
	Convey("Given an EvidenceAssembler", t, func() {
		assembler := NewEvidenceAssembler()

		Convey("When generating why_selected for high score", func() {
			input := AssemblyInput{
				Score:        0.9,
				MatchedTerms: []string{"machine", "learning"},
				Source:       "official_documentation",
			}
			context := &AssemblyContext{Query: "machine learning"}

			whySelected := assembler.generateWhySelected(input, context)

			So(whySelected, ShouldContainSubstring, "high relevance score")
			So(whySelected, ShouldContainSubstring, "machine, learning")
			So(whySelected, ShouldContainSubstring, "official")
		})

		Convey("When generating why_selected for recent content", func() {
			input := AssemblyInput{
				Score:     0.7,
				Timestamp: time.Now().Add(-30 * time.Minute),
			}
			context := &AssemblyContext{}

			whySelected := assembler.generateWhySelected(input, context)

			So(whySelected, ShouldContainSubstring, "very recent content")
		})

		Convey("When generating why_selected with graph path", func() {
			input := AssemblyInput{
				Score:     0.7,
				GraphPath: []string{"entity1", "relation", "entity2"},
			}
			context := &AssemblyContext{}

			whySelected := assembler.generateWhySelected(input, context)

			So(whySelected, ShouldContainSubstring, "2-hop graph path")
		})

		Convey("When generating why_selected with different detail levels", func() {
			config := &EvidenceAssemblerConfig{
				WhySelectedDetail: "verbose",
			}
			assembler := NewEvidenceAssemblerWithConfig(config)

			input := AssemblyInput{
				Score:        0.7,
				MatchedTerms: []string{"term1", "term2", "term3", "term4"},
			}
			context := &AssemblyContext{}

			whySelected := assembler.generateWhySelected(input, context)

			So(whySelected, ShouldContainSubstring, "matches all terms")
			So(whySelected, ShouldContainSubstring, "term1, term2, term3, term4")
		})
	})
}

func TestEvidenceAssembler_RelationMap(t *testing.T) {
	Convey("Given an EvidenceAssembler", t, func() {
		assembler := NewEvidenceAssembler()

		Convey("When adding relation map to evidence", func() {
			evidence := &Evidence{RelationMap: make(map[string]string)}
			input := AssemblyInput{
				Source:          "test_source",
				RelatedEntities: []string{"entity1", "entity2"},
				Metadata:        map[string]interface{}{"category": "AI", "topic": "ML"},
			}
			context := &AssemblyContext{
				GraphContext: &GraphContext{
					QueryEntities:   []string{"query_entity"},
					RelatedEntities: []string{"related_entity"},
				},
			}

			assembler.addRelationMap(evidence, input, context)

			So(evidence.RelationMap["entity1"], ShouldEqual, "related_entity")
			So(evidence.RelationMap["entity2"], ShouldEqual, "related_entity")
			So(evidence.RelationMap["query_entity"], ShouldEqual, "query_entity")
			So(evidence.RelationMap["related_entity"], ShouldEqual, "contextual_entity")
			So(evidence.RelationMap["source"], ShouldEqual, "test_source")
			So(evidence.RelationMap["category"], ShouldEqual, "AI")
			So(evidence.RelationMap["topic"], ShouldEqual, "ML")
		})
	})
}

func TestEvidenceAssembler_Deduplication(t *testing.T) {
	Convey("Given an EvidenceAssembler", t, func() {
		assembler := NewEvidenceAssembler()

		Convey("When deduplicating similar content", func() {
			inputs := []AssemblyInput{
				{ID: "doc1", Content: "Machine learning is a subset of artificial intelligence"},
				{ID: "doc2", Content: "Machine learning is a subset of artificial intelligence"}, // Exact duplicate
				{ID: "doc3", Content: "Machine learning algorithms are part of AI systems"},      // Similar
				{ID: "doc4", Content: "Cooking recipes for Italian pasta dishes"},                // Different
			}

			deduplicated := assembler.deduplicateInputs(inputs)

			So(len(deduplicated), ShouldBeLessThan, len(inputs))
			So(len(deduplicated), ShouldBeGreaterThanOrEqualTo, 2) // Should keep at least different content
		})

		Convey("When deduplicating with custom similarity threshold", func() {
			config := &EvidenceAssemblerConfig{
				SimilarityThreshold: 0.9, // Very high threshold
			}
			assembler := NewEvidenceAssemblerWithConfig(config)

			inputs := []AssemblyInput{
				{ID: "doc1", Content: "Machine learning algorithms"},
				{ID: "doc2", Content: "Machine learning systems"}, // Similar but below threshold
			}

			deduplicated := assembler.deduplicateInputs(inputs)

			So(len(deduplicated), ShouldEqual, 2) // Both should be kept
		})
	})
}

func TestEvidenceAssembler_ContentSimilarity(t *testing.T) {
	Convey("Given an EvidenceAssembler", t, func() {
		assembler := NewEvidenceAssembler()

		Convey("When calculating similarity for identical content", func() {
			content1 := "Machine learning algorithms are powerful tools"
			content2 := "Machine learning algorithms are powerful tools"

			similarity := assembler.calculateContentSimilarity(content1, content2)
			So(similarity, ShouldEqual, 1.0)
		})

		Convey("When calculating similarity for completely different content", func() {
			content1 := "Machine learning algorithms"
			content2 := "Cooking pasta recipes"

			similarity := assembler.calculateContentSimilarity(content1, content2)
			So(similarity, ShouldEqual, 0.0)
		})

		Convey("When calculating similarity for partially similar content", func() {
			content1 := "Machine learning algorithms are powerful"
			content2 := "Machine learning systems are useful"

			similarity := assembler.calculateContentSimilarity(content1, content2)
			So(similarity, ShouldBeGreaterThan, 0.0)
			So(similarity, ShouldBeLessThan, 1.0)
		})

		Convey("When calculating similarity for empty content", func() {
			similarity := assembler.calculateContentSimilarity("", "some content")
			So(similarity, ShouldEqual, 0.0)

			similarity = assembler.calculateContentSimilarity("some content", "")
			So(similarity, ShouldEqual, 0.0)
		})
	})
}

func TestEvidenceAssembler_AssembleFromResults(t *testing.T) {
	Convey("Given an EvidenceAssembler", t, func() {
		assembler := NewEvidenceAssembler()
		ctx := context.Background()

		Convey("When assembling from fused results", func() {
			fusedResults := []FusedResult{
				{
					ID:           "doc1",
					Content:      "Fused result content",
					FinalScore:   0.9,
					MethodScores: map[string]float64{"vector": 0.8, "keyword": 0.9},
					Metadata:     map[string]interface{}{"type": "fused"},
				},
			}
			context := &AssemblyContext{Query: "test"}

			response, err := assembler.AssembleFromFusedResults(ctx, fusedResults, context)

			So(err, ShouldBeNil)
			So(response.TotalEvidence, ShouldEqual, 1)
			So(response.Evidence[0].Content, ShouldEqual, "Fused result content")
			So(response.Evidence[0].Confidence, ShouldEqual, 0.9)
		})

		Convey("When assembling from vector results", func() {
			vectorResults := []VectorSearchResult{
				{
					ID:       "doc1",
					Content:  "Vector search result",
					Score:    0.85,
					Metadata: map[string]interface{}{"type": "vector"},
				},
			}
			context := &AssemblyContext{Query: "test"}

			response, err := assembler.AssembleFromVectorResults(ctx, vectorResults, context)

			So(err, ShouldBeNil)
			So(response.TotalEvidence, ShouldEqual, 1)
			So(response.Evidence[0].Content, ShouldEqual, "Vector search result")
			So(response.Metadata["retrieval_method"], ShouldEqual, "vector")
		})

		Convey("When assembling from keyword results", func() {
			keywordResults := []KeywordSearchResult{
				{
					ID:           "doc1",
					Content:      "Keyword search result",
					Score:        0.8,
					MatchedTerms: []string{"keyword", "search"},
					Metadata:     map[string]interface{}{"type": "keyword"},
				},
			}
			context := &AssemblyContext{Query: "keyword search"}

			response, err := assembler.AssembleFromKeywordResults(ctx, keywordResults, context)

			So(err, ShouldBeNil)
			So(response.TotalEvidence, ShouldEqual, 1)
			So(response.Evidence[0].Content, ShouldEqual, "Keyword search result")
			So(response.Metadata["retrieval_method"], ShouldEqual, "keyword")
		})
	})
}

func TestEvidenceAssembler_Configuration(t *testing.T) {
	Convey("Given an EvidenceAssembler", t, func() {
		assembler := NewEvidenceAssembler()

		Convey("When getting configuration", func() {
			config := assembler.GetConfig()
			So(config, ShouldNotBeNil)
			So(config.MaxEvidenceItems, ShouldEqual, 50)
		})

		Convey("When updating configuration", func() {
			newConfig := &EvidenceAssemblerConfig{
				MaxEvidenceItems: 25,
				MinConfidence:    0.5,
				ValidateEvidence: false,
			}

			assembler.UpdateConfig(newConfig)
			config := assembler.GetConfig()

			So(config.MaxEvidenceItems, ShouldEqual, 25)
			So(config.MinConfidence, ShouldEqual, 0.5)
			So(config.ValidateEvidence, ShouldBeFalse)
		})

		Convey("When updating with nil config", func() {
			originalConfig := assembler.GetConfig()
			originalMax := originalConfig.MaxEvidenceItems

			assembler.UpdateConfig(nil)
			config := assembler.GetConfig()

			So(config.MaxEvidenceItems, ShouldEqual, originalMax)
		})
	})
}