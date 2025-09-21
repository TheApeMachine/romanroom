package main

import (
	"context"
	"strconv"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestResultFuser(t *testing.T) {
	Convey("Given a ResultFuser", t, func() {
		fuser := NewResultFuser()

		Convey("When creating with default config", func() {
			So(fuser, ShouldNotBeNil)
			So(fuser.config.RRFConstant, ShouldEqual, 60.0)
			So(fuser.config.VectorWeight, ShouldEqual, 1.0)
			So(fuser.config.KeywordWeight, ShouldEqual, 1.0)
			So(fuser.config.GraphWeight, ShouldEqual, 1.0)
		})

		Convey("When creating with custom config", func() {
			config := &ResultFuserConfig{
				RRFConstant:   30.0,
				VectorWeight:  2.0,
				KeywordWeight: 1.5,
				MaxResults:    50,
			}
			customFuser := NewResultFuserWithConfig(config)

			So(customFuser.config.RRFConstant, ShouldEqual, 30.0)
			So(customFuser.config.VectorWeight, ShouldEqual, 2.0)
			So(customFuser.config.KeywordWeight, ShouldEqual, 1.5)
			So(customFuser.config.MaxResults, ShouldEqual, 50)
		})
	})
}

func TestResultFuser_Fuse(t *testing.T) {
	Convey("Given a ResultFuser and fusion inputs", t, func() {
		fuser := NewResultFuser()
		ctx := context.Background()

		Convey("When fusing empty inputs", func() {
			inputs := []FusionInput{}
			response, err := fuser.Fuse(ctx, inputs)

			So(err, ShouldBeNil)
			So(response, ShouldNotBeNil)
			So(response.TotalResults, ShouldEqual, 0)
			So(len(response.Results), ShouldEqual, 0)
		})

		Convey("When fusing single method results", func() {
			inputs := []FusionInput{
				{
					Method: "vector",
					Weight: 1.0,
					Results: []FusionItem{
						{ID: "doc1", Content: "First document", Score: 0.9},
						{ID: "doc2", Content: "Second document", Score: 0.8},
						{ID: "doc3", Content: "Third document", Score: 0.7},
					},
				},
			}

			response, err := fuser.Fuse(ctx, inputs)

			So(err, ShouldBeNil)
			So(response.TotalResults, ShouldEqual, 3)
			So(len(response.Results), ShouldEqual, 3)

			// Check that results are sorted by final score
			So(response.Results[0].ID, ShouldEqual, "doc1")
			So(response.Results[1].ID, ShouldEqual, "doc2")
			So(response.Results[2].ID, ShouldEqual, "doc3")

			// Check RRF scores
			So(response.Results[0].RRFScore, ShouldBeGreaterThan, response.Results[1].RRFScore)
			So(response.Results[1].RRFScore, ShouldBeGreaterThan, response.Results[2].RRFScore)
		})

		Convey("When fusing multiple method results with overlap", func() {
			inputs := []FusionInput{
				{
					Method: "vector",
					Weight: 1.0,
					Results: []FusionItem{
						{ID: "doc1", Content: "First document", Score: 0.9},
						{ID: "doc2", Content: "Second document", Score: 0.8},
					},
				},
				{
					Method: "keyword",
					Weight: 1.0,
					Results: []FusionItem{
						{ID: "doc2", Content: "Second document", Score: 0.85},
						{ID: "doc3", Content: "Third document", Score: 0.75},
					},
				},
			}

			response, err := fuser.Fuse(ctx, inputs)

			So(err, ShouldBeNil)
			So(response.TotalResults, ShouldEqual, 3)

			// doc2 should be ranked highest due to appearing in both methods
			So(response.Results[0].ID, ShouldEqual, "doc2")
			So(len(response.Results[0].SourceMethods), ShouldEqual, 2)
			So(response.Results[0].SourceMethods, ShouldContain, "vector")
			So(response.Results[0].SourceMethods, ShouldContain, "keyword")

			// Check fusion stats
			So(response.FusionStats.InputMethods, ShouldContain, "vector")
			So(response.FusionStats.InputMethods, ShouldContain, "keyword")
			So(response.FusionStats.MethodCounts["vector"], ShouldEqual, 2)
			So(response.FusionStats.MethodCounts["keyword"], ShouldEqual, 2)
		})

		Convey("When fusing with minimum score threshold", func() {
			config := &ResultFuserConfig{
				RRFConstant: 60.0,
				MinScore:    0.01, // Very low threshold since RRF scores are typically small
				MaxResults:  100,
			}
			fuser := NewResultFuserWithConfig(config)

			inputs := []FusionInput{
				{
					Method: "vector",
					Results: []FusionItem{
						{ID: "doc1", Content: "High score doc", Score: 0.9},
						{ID: "doc2", Content: "Low score doc", Score: 0.3},
					},
				},
			}

			response, err := fuser.Fuse(ctx, inputs)

			So(err, ShouldBeNil)
			So(response.TotalResults, ShouldEqual, 2) // Both should pass with low threshold
			// Verify that doc1 is ranked higher due to better original score
			So(response.Results[0].ID, ShouldEqual, "doc1")
		})
	})
}

func TestResultFuser_RRF(t *testing.T) {
	Convey("Given a ResultFuser and RRF inputs", t, func() {
		fuser := NewResultFuser()

		Convey("When calculating RRF scores", func() {
			inputs := []FusionInput{
				{
					Method: "vector",
					Results: []FusionItem{
						{ID: "doc1", Content: "First", Score: 0.9},
						{ID: "doc2", Content: "Second", Score: 0.8},
					},
				},
				{
					Method: "keyword",
					Results: []FusionItem{
						{ID: "doc2", Content: "Second", Score: 0.85},
						{ID: "doc1", Content: "First", Score: 0.75},
					},
				},
			}

			rrfResults := fuser.RRF(inputs)

			So(len(rrfResults), ShouldEqual, 2)

			// doc1: RRF = 1/(60+1) + 1/(60+2) = 1/61 + 1/62
			// doc2: RRF = 1/(60+2) + 1/(60+1) = 1/62 + 1/61
			// Both should have the same RRF score since they appear in both lists
			doc1RRF := rrfResults["doc1"].RRFScore
			doc2RRF := rrfResults["doc2"].RRFScore

			So(doc1RRF, ShouldAlmostEqual, doc2RRF, 0.001)
			So(doc1RRF, ShouldBeGreaterThan, 0)
			So(doc2RRF, ShouldBeGreaterThan, 0)
		})

		Convey("When calculating RRF with different constants", func() {
			config := &ResultFuserConfig{RRFConstant: 10.0}
			fuser := NewResultFuserWithConfig(config)

			inputs := []FusionInput{
				{
					Method: "vector",
					Results: []FusionItem{
						{ID: "doc1", Content: "First", Score: 0.9},
					},
				},
			}

			rrfResults := fuser.RRF(inputs)

			// RRF = 1/(10+1) = 1/11
			expectedRRF := 1.0 / 11.0
			So(rrfResults["doc1"].RRFScore, ShouldAlmostEqual, expectedRRF, 0.001)
		})
	})
}

func TestResultFuser_CombineScores(t *testing.T) {
	Convey("Given a ResultFuser and RRF results", t, func() {
		fuser := NewResultFuser()

		Convey("When combining scores with equal weights", func() {
			rrfResults := map[string]*FusedResult{
				"doc1": {
					ID:           "doc1",
					RRFScore:     0.5,
					MethodScores: map[string]float64{"vector": 0.9, "keyword": 0.8},
				},
			}

			inputs := []FusionInput{
				{Method: "vector", Weight: 1.0},
				{Method: "keyword", Weight: 1.0},
			}

			combinedResults := fuser.CombineScores(rrfResults, inputs)

			So(len(combinedResults), ShouldEqual, 1)
			result := combinedResults[0]

			// Combined score should be RRF * (1 + weighted_average)
			// weighted_average = (0.9*1.0 + 0.8*1.0) / (1.0 + 1.0) = 1.7/2.0 = 0.85
			// combined_score = 0.5 * (1 + 0.85) = 0.5 * 1.85 = 0.925
			expectedCombined := 0.5 * (1.0 + 0.85)
			So(result.CombinedScore, ShouldAlmostEqual, expectedCombined, 0.001)
			So(result.FinalScore, ShouldEqual, result.CombinedScore)
		})

		Convey("When combining scores with different weights", func() {
			config := &ResultFuserConfig{
				VectorWeight:  2.0,
				KeywordWeight: 1.0,
			}
			fuser := NewResultFuserWithConfig(config)

			rrfResults := map[string]*FusedResult{
				"doc1": {
					ID:           "doc1",
					RRFScore:     0.5,
					MethodScores: map[string]float64{"vector": 0.9, "keyword": 0.6},
				},
			}

			inputs := []FusionInput{
				{Method: "vector", Weight: 2.0},
				{Method: "keyword", Weight: 1.0},
			}

			combinedResults := fuser.CombineScores(rrfResults, inputs)

			result := combinedResults[0]

			// weighted_average = (0.9*2.0 + 0.6*1.0) / (2.0 + 1.0) = 2.4/3.0 = 0.8
			// combined_score = 0.5 * (1 + 0.8) = 0.5 * 1.8 = 0.9
			expectedCombined := 0.5 * (1.0 + 0.8)
			So(result.CombinedScore, ShouldAlmostEqual, expectedCombined, 0.001)
		})
	})
}

func TestResultFuser_FuseVectorAndKeyword(t *testing.T) {
	Convey("Given a ResultFuser and vector/keyword results", t, func() {
		fuser := NewResultFuser()
		ctx := context.Background()

		Convey("When fusing vector and keyword results", func() {
			vectorResults := []VectorSearchResult{
				{ID: "doc1", Content: "Vector result 1", Score: 0.9, Metadata: map[string]interface{}{"type": "vector"}},
				{ID: "doc2", Content: "Vector result 2", Score: 0.8, Metadata: map[string]interface{}{"type": "vector"}},
			}

			keywordResults := []KeywordSearchResult{
				{ID: "doc2", Content: "Keyword result 2", Score: 0.85, Metadata: map[string]interface{}{"type": "keyword"}},
				{ID: "doc3", Content: "Keyword result 3", Score: 0.75, Metadata: map[string]interface{}{"type": "keyword"}},
			}

			response, err := fuser.FuseVectorAndKeyword(ctx, vectorResults, keywordResults)

			So(err, ShouldBeNil)
			So(response.TotalResults, ShouldEqual, 3)

			// doc2 should be ranked highest due to appearing in both methods
			So(response.Results[0].ID, ShouldEqual, "doc2")
			So(len(response.Results[0].SourceMethods), ShouldEqual, 2)
		})

		Convey("When fusing with empty vector results", func() {
			vectorResults := []VectorSearchResult{}
			keywordResults := []KeywordSearchResult{
				{ID: "doc1", Content: "Keyword only", Score: 0.8},
			}

			response, err := fuser.FuseVectorAndKeyword(ctx, vectorResults, keywordResults)

			So(err, ShouldBeNil)
			So(response.TotalResults, ShouldEqual, 1)
			So(response.Results[0].ID, ShouldEqual, "doc1")
			So(len(response.Results[0].SourceMethods), ShouldEqual, 1)
			So(response.Results[0].SourceMethods[0], ShouldEqual, "keyword")
		})

		Convey("When fusing with empty keyword results", func() {
			vectorResults := []VectorSearchResult{
				{ID: "doc1", Content: "Vector only", Score: 0.8},
			}
			keywordResults := []KeywordSearchResult{}

			response, err := fuser.FuseVectorAndKeyword(ctx, vectorResults, keywordResults)

			So(err, ShouldBeNil)
			So(response.TotalResults, ShouldEqual, 1)
			So(response.Results[0].ID, ShouldEqual, "doc1")
			So(len(response.Results[0].SourceMethods), ShouldEqual, 1)
			So(response.Results[0].SourceMethods[0], ShouldEqual, "vector")
		})
	})
}

func TestResultFuser_Configuration(t *testing.T) {
	Convey("Given a ResultFuser", t, func() {
		fuser := NewResultFuser()

		Convey("When getting configuration", func() {
			config := fuser.GetConfig()
			So(config, ShouldNotBeNil)
			So(config.RRFConstant, ShouldEqual, 60.0)
		})

		Convey("When updating configuration", func() {
			newConfig := &ResultFuserConfig{
				RRFConstant:   30.0,
				VectorWeight:  2.0,
				MaxResults:    50,
			}

			fuser.UpdateConfig(newConfig)
			config := fuser.GetConfig()

			So(config.RRFConstant, ShouldEqual, 30.0)
			So(config.VectorWeight, ShouldEqual, 2.0)
			So(config.MaxResults, ShouldEqual, 50)
		})

		Convey("When updating with nil config", func() {
			originalConfig := fuser.GetConfig()
			originalRRF := originalConfig.RRFConstant

			fuser.UpdateConfig(nil)
			config := fuser.GetConfig()

			So(config.RRFConstant, ShouldEqual, originalRRF)
		})
	})
}

func TestResultFuser_EdgeCases(t *testing.T) {
	Convey("Given a ResultFuser and edge case inputs", t, func() {
		fuser := NewResultFuser()
		ctx := context.Background()

		Convey("When fusing with invalid inputs", func() {
			inputs := []FusionInput{
				{
					Method:  "", // Empty method
					Results: []FusionItem{{ID: "doc1", Score: 0.8}},
				},
			}

			response, err := fuser.Fuse(ctx, inputs)

			So(err, ShouldNotBeNil)
			So(response, ShouldBeNil)
		})

		Convey("When fusing with empty results in input", func() {
			inputs := []FusionInput{
				{
					Method:  "vector",
					Results: []FusionItem{}, // Empty results
				},
				{
					Method: "keyword",
					Results: []FusionItem{
						{ID: "doc1", Content: "Valid result", Score: 0.8},
					},
				},
			}

			response, err := fuser.Fuse(ctx, inputs)

			So(err, ShouldBeNil)
			So(response.TotalResults, ShouldEqual, 1)
			So(response.Results[0].ID, ShouldEqual, "doc1")
		})

		Convey("When fusing with zero weights", func() {
			inputs := []FusionInput{
				{
					Method: "vector",
					Weight: 0.0, // Zero weight should use default
					Results: []FusionItem{
						{ID: "doc1", Content: "Test", Score: 0.8},
					},
				},
			}

			response, err := fuser.Fuse(ctx, inputs)

			So(err, ShouldBeNil)
			So(response.TotalResults, ShouldEqual, 1)
		})

		Convey("When fusing with very large number of results", func() {
			config := &ResultFuserConfig{MaxResults: 5}
			fuser := NewResultFuserWithConfig(config)

			// Create 10 results but limit to 5
			results := make([]FusionItem, 10)
			for i := 0; i < 10; i++ {
				results[i] = FusionItem{
					ID:      "doc" + strconv.Itoa(i),
					Content: "Document " + strconv.Itoa(i),
					Score:   0.9 - float64(i)*0.05,
				}
			}

			inputs := []FusionInput{
				{Method: "vector", Results: results},
			}

			response, err := fuser.Fuse(ctx, inputs)

			So(err, ShouldBeNil)
			So(response.TotalResults, ShouldEqual, 5)
			So(len(response.Results), ShouldEqual, 5)
		})
	})
}

func TestResultFuser_Normalization(t *testing.T) {
	Convey("Given a ResultFuser with score normalization enabled", t, func() {
		config := &ResultFuserConfig{
			NormalizeScores: true,
			RRFConstant:     60.0,
		}
		fuser := NewResultFuserWithConfig(config)
		ctx := context.Background()

		Convey("When fusing results with different score ranges", func() {
			inputs := []FusionInput{
				{
					Method: "vector",
					Results: []FusionItem{
						{ID: "doc1", Score: 0.9},
						{ID: "doc2", Score: 0.1},
					},
				},
				{
					Method: "keyword",
					Results: []FusionItem{
						{ID: "doc1", Score: 100.0}, // Different scale
						{ID: "doc2", Score: 50.0},
					},
				},
			}

			response, err := fuser.Fuse(ctx, inputs)

			So(err, ShouldBeNil)
			So(response.TotalResults, ShouldEqual, 2)

			// After normalization, both methods should contribute equally
			// despite different original score ranges
			So(response.Metadata["normalized_scores"], ShouldEqual, true)
		})
	})
}