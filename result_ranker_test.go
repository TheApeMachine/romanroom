package main

import (
	"context"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestResultRanker(t *testing.T) {
	Convey("Given a ResultRanker", t, func() {
		ranker := NewResultRanker()

		Convey("When creating with default config", func() {
			So(ranker, ShouldNotBeNil)
			So(ranker.config.RelevanceWeight, ShouldEqual, 1.0)
			So(ranker.config.FreshnessWeight, ShouldEqual, 0.2)
			So(ranker.config.AuthorityWeight, ShouldEqual, 0.3)
			So(ranker.config.QualityWeight, ShouldEqual, 0.4)
			So(ranker.config.MaxResults, ShouldEqual, 100)
		})

		Convey("When creating with custom config", func() {
			config := &ResultRankerConfig{
				RelevanceWeight: 2.0,
				FreshnessWeight: 0.5,
				MaxResults:      50,
			}
			customRanker := NewResultRankerWithConfig(config)

			So(customRanker.config.RelevanceWeight, ShouldEqual, 2.0)
			So(customRanker.config.FreshnessWeight, ShouldEqual, 0.5)
			So(customRanker.config.MaxResults, ShouldEqual, 50)
		})
	})
}

func TestResultRanker_Rank(t *testing.T) {
	Convey("Given a ResultRanker and rankable results", t, func() {
		ranker := NewResultRanker()
		ctx := context.Background()

		Convey("When ranking empty results", func() {
			results := []RankableResult{}
			context := &RankingContext{Query: "test query"}

			response, err := ranker.Rank(ctx, results, context)

			So(err, ShouldBeNil)
			So(response, ShouldNotBeNil)
			So(response.TotalResults, ShouldEqual, 0)
			So(len(response.Results), ShouldEqual, 0)
		})

		Convey("When ranking single result", func() {
			results := []RankableResult{
				{
					ID:        "doc1",
					Content:   "This is a test document about machine learning",
					BaseScore: 0.8,
					Source:    "test_source",
					Timestamp: time.Now().Add(-24 * time.Hour),
					Metadata:  map[string]interface{}{"title": "ML Guide"},
				},
			}
			context := &RankingContext{
				Query:       "machine learning",
				TimeContext: time.Now(),
			}

			response, err := ranker.Rank(ctx, results, context)

			So(err, ShouldBeNil)
			So(response.TotalResults, ShouldEqual, 1)
			So(len(response.Results), ShouldEqual, 1)

			result := response.Results[0]
			So(result.ID, ShouldEqual, "doc1")
			So(result.Rank, ShouldEqual, 1)
			So(result.RelevanceScore, ShouldBeGreaterThan, 0)
			So(result.FreshnessScore, ShouldBeGreaterThan, 0)
			So(result.QualityScore, ShouldBeGreaterThan, 0)
			So(result.FinalScore, ShouldBeGreaterThan, 0)
		})

		Convey("When ranking multiple results", func() {
			now := time.Now()
			results := []RankableResult{
				{
					ID:        "doc1",
					Content:   "Old document about cats",
					BaseScore: 0.6,
					Source:    "blog",
					Timestamp: now.Add(-30 * 24 * time.Hour), // 30 days old
				},
				{
					ID:        "doc2",
					Content:   "Recent document about machine learning and AI",
					BaseScore: 0.8,
					Source:    "official_source",
					Timestamp: now.Add(-1 * time.Hour), // 1 hour old
					Metadata:  map[string]interface{}{"title": "machine learning guide"},
				},
				{
					ID:        "doc3",
					Content:   "Medium quality document",
					BaseScore: 0.7,
					Source:    "verified_source",
					Timestamp: now.Add(-7 * 24 * time.Hour), // 7 days old
				},
			}
			context := &RankingContext{
				Query:       "machine learning",
				TimeContext: now,
			}

			response, err := ranker.Rank(ctx, results, context)

			So(err, ShouldBeNil)
			So(response.TotalResults, ShouldEqual, 3)

			// doc2 should be ranked highest due to:
			// - High relevance (matches query + title)
			// - High freshness (recent)
			// - Authority boost (official source)
			So(response.Results[0].ID, ShouldEqual, "doc2")
			So(response.Results[0].Rank, ShouldEqual, 1)

			// Check that final scores are properly calculated
			for _, result := range response.Results {
				So(result.FinalScore, ShouldBeGreaterThan, 0)
				So(result.FinalScore, ShouldBeLessThanOrEqualTo, 1.0)
			}

			// Check ranking stats
			So(response.RankingStats.RankingTime, ShouldBeGreaterThan, 0)
			So(len(response.RankingStats.ScoreDistribution), ShouldBeGreaterThan, 0)
		})

		Convey("When ranking with personalization", func() {
			results := []RankableResult{
				{
					ID:        "doc1",
					Content:   "Document about Python programming",
					BaseScore: 0.7,
					Source:    "tutorial_site",
				},
				{
					ID:        "doc2",
					Content:   "Document about Java programming",
					BaseScore: 0.7,
					Source:    "blog",
				},
			}
			context := &RankingContext{
				Query:  "programming",
				UserID: "user123",
				UserPreferences: map[string]interface{}{
					"topics":  []string{"python", "machine learning"},
					"sources": []string{"tutorial_site"},
				},
			}

			response, err := ranker.Rank(ctx, results, context)

			So(err, ShouldBeNil)
			So(response.TotalResults, ShouldEqual, 2)

			// doc1 should be ranked higher due to personalization
			// (matches preferred topic "python" and preferred source)
			So(response.Results[0].ID, ShouldEqual, "doc1")
			So(response.Results[0].PersonalizationScore, ShouldBeGreaterThan, response.Results[1].PersonalizationScore)
		})
	})
}

func TestResultRanker_Score(t *testing.T) {
	Convey("Given a ResultRanker and a single result", t, func() {
		ranker := NewResultRanker()

		Convey("When scoring a high-quality result", func() {
			result := &RankableResult{
				ID:        "doc1",
				Content:   "This is a comprehensive guide about machine learning algorithms and their applications in real-world scenarios.",
				BaseScore: 0.9,
				Source:    "official_documentation",
				Timestamp: time.Now().Add(-1 * time.Hour),
				Metadata:  map[string]interface{}{"title": "machine learning guide", "authority_score": 0.9},
			}
			context := &RankingContext{
				Query:       "machine learning",
				TimeContext: time.Now(),
			}

			finalScore := ranker.Score(result, context)

			So(finalScore, ShouldBeGreaterThan, 0.8)
			So(result.RelevanceScore, ShouldBeGreaterThan, 0.8)
			So(result.FreshnessScore, ShouldBeGreaterThan, 0.9)
			So(result.AuthorityScore, ShouldBeGreaterThan, 0.8)
			So(result.QualityScore, ShouldBeGreaterThan, 0.5)
		})

		Convey("When scoring a low-quality result", func() {
			result := &RankableResult{
				ID:        "doc2",
				Content:   "Short text",
				BaseScore: 0.3,
				Source:    "unknown",
				Timestamp: time.Now().Add(-365 * 24 * time.Hour), // 1 year old
			}
			context := &RankingContext{
				Query:       "machine learning",
				TimeContext: time.Now(),
			}

			finalScore := ranker.Score(result, context)

			So(finalScore, ShouldBeLessThan, 0.5)
			So(result.FreshnessScore, ShouldBeLessThan, 0.1)
			So(result.QualityScore, ShouldBeLessThan, 0.5)
		})
	})
}

func TestResultRanker_SortResults(t *testing.T) {
	Convey("Given a ResultRanker and unsorted results", t, func() {
		ranker := NewResultRanker()

		Convey("When sorting results by final score", func() {
			results := []RankableResult{
				{ID: "doc1", FinalScore: 0.7, BaseScore: 0.8},
				{ID: "doc2", FinalScore: 0.9, BaseScore: 0.7},
				{ID: "doc3", FinalScore: 0.5, BaseScore: 0.6},
				{ID: "doc4", FinalScore: 0.9, BaseScore: 0.9}, // Same final score as doc2
			}

			sorted := ranker.SortResults(results)

			So(len(sorted), ShouldEqual, 4)
			So(sorted[0].FinalScore, ShouldBeGreaterThanOrEqualTo, sorted[1].FinalScore)
			So(sorted[1].FinalScore, ShouldBeGreaterThanOrEqualTo, sorted[2].FinalScore)
			So(sorted[2].FinalScore, ShouldBeGreaterThanOrEqualTo, sorted[3].FinalScore)

			// When final scores are equal, should use base score as tiebreaker
			if sorted[0].FinalScore == sorted[1].FinalScore {
				So(sorted[0].BaseScore, ShouldBeGreaterThanOrEqualTo, sorted[1].BaseScore)
			}
		})

		Convey("When sorting preserves original slice", func() {
			original := []RankableResult{
				{ID: "doc1", FinalScore: 0.5},
				{ID: "doc2", FinalScore: 0.9},
			}

			sorted := ranker.SortResults(original)

			// Original should be unchanged
			So(original[0].ID, ShouldEqual, "doc1")
			So(original[1].ID, ShouldEqual, "doc2")

			// Sorted should be in correct order
			So(sorted[0].ID, ShouldEqual, "doc2")
			So(sorted[1].ID, ShouldEqual, "doc1")
		})
	})
}

func TestResultRanker_RelevanceScore(t *testing.T) {
	Convey("Given a ResultRanker", t, func() {
		ranker := NewResultRanker()

		Convey("When calculating relevance score with exact match", func() {
			result := &RankableResult{
				ID:       "doc1",
				Content:  "This document is about machine learning algorithms",
				BaseScore: 0.8,
				Metadata: map[string]interface{}{"title": "Machine Learning Guide"},
			}
			context := &RankingContext{Query: "machine learning"}

			ranker.calculateRelevanceScore(result, context)

			// Should get boost for content match and title match
			So(result.RelevanceScore, ShouldBeGreaterThan, 0.8)
			So(result.RelevanceScore, ShouldBeLessThanOrEqualTo, 1.0)
		})

		Convey("When calculating relevance score with no match", func() {
			result := &RankableResult{
				ID:       "doc1",
				Content:  "This document is about cooking recipes",
				BaseScore: 0.8,
			}
			context := &RankingContext{Query: "machine learning"}

			ranker.calculateRelevanceScore(result, context)

			// Should use base score when no matches
			So(result.RelevanceScore, ShouldEqual, 0.8)
		})

		Convey("When calculating relevance score with empty query", func() {
			result := &RankableResult{
				ID:       "doc1",
				Content:  "Any content",
				BaseScore: 0.7,
			}
			context := &RankingContext{Query: ""}

			ranker.calculateRelevanceScore(result, context)

			So(result.RelevanceScore, ShouldEqual, 0.7)
		})
	})
}

func TestResultRanker_FreshnessScore(t *testing.T) {
	Convey("Given a ResultRanker", t, func() {
		ranker := NewResultRanker()
		now := time.Now()

		Convey("When calculating freshness score for recent content", func() {
			result := &RankableResult{
				ID:        "doc1",
				Timestamp: now.Add(-1 * time.Hour),
			}
			context := &RankingContext{TimeContext: now}

			ranker.calculateFreshnessScore(result, context)

			So(result.FreshnessScore, ShouldBeGreaterThan, 0.9)
		})

		Convey("When calculating freshness score for old content", func() {
			result := &RankableResult{
				ID:        "doc1",
				Timestamp: now.Add(-365 * 24 * time.Hour), // 1 year old
			}
			context := &RankingContext{TimeContext: now}

			ranker.calculateFreshnessScore(result, context)

			So(result.FreshnessScore, ShouldBeLessThan, 0.1)
		})

		Convey("When calculating freshness score with no timestamp", func() {
			result := &RankableResult{
				ID: "doc1",
				// No timestamp
			}
			context := &RankingContext{TimeContext: now}

			ranker.calculateFreshnessScore(result, context)

			So(result.FreshnessScore, ShouldEqual, 0.5) // Default neutral score
		})
	})
}

func TestResultRanker_AuthorityScore(t *testing.T) {
	Convey("Given a ResultRanker", t, func() {
		ranker := NewResultRanker()

		Convey("When calculating authority score with trusted source", func() {
			result := &RankableResult{
				ID:     "doc1",
				Source: "official_documentation",
			}
			context := &RankingContext{}

			ranker.calculateAuthorityScore(result, context)

			So(result.AuthorityScore, ShouldBeGreaterThan, 0.5)
		})

		Convey("When calculating authority score with metadata", func() {
			result := &RankableResult{
				ID:       "doc1",
				Source:   "blog",
				Metadata: map[string]interface{}{"authority_score": 0.9},
			}
			context := &RankingContext{}

			ranker.calculateAuthorityScore(result, context)

			So(result.AuthorityScore, ShouldEqual, 0.9)
		})

		Convey("When calculating authority score with unknown source", func() {
			result := &RankableResult{
				ID:     "doc1",
				Source: "random_blog",
			}
			context := &RankingContext{}

			ranker.calculateAuthorityScore(result, context)

			So(result.AuthorityScore, ShouldEqual, 0.5) // Default score
		})
	})
}

func TestResultRanker_QualityScore(t *testing.T) {
	Convey("Given a ResultRanker", t, func() {
		ranker := NewResultRanker()

		Convey("When calculating quality score for optimal length content", func() {
			result := &RankableResult{
				ID:      "doc1",
				Content: strings.Repeat("This is good quality content. ", 20), // ~600 chars
			}
			context := &RankingContext{}

			ranker.calculateQualityScore(result, context)

			So(result.QualityScore, ShouldBeGreaterThan, 0.5)
		})

		Convey("When calculating quality score for very short content", func() {
			result := &RankableResult{
				ID:      "doc1",
				Content: "Short",
			}
			context := &RankingContext{}

			ranker.calculateQualityScore(result, context)

			So(result.QualityScore, ShouldBeLessThan, 0.5)
		})

		Convey("When calculating quality score with metadata", func() {
			result := &RankableResult{
				ID:       "doc1",
				Content:  "Some content of reasonable length for testing quality scoring",
				Metadata: map[string]interface{}{"quality_score": 0.8},
			}
			context := &RankingContext{}

			ranker.calculateQualityScore(result, context)

			So(result.QualityScore, ShouldBeGreaterThan, 0.5)
		})
	})
}

func TestResultRanker_DiversityScores(t *testing.T) {
	Convey("Given a ResultRanker and multiple results", t, func() {
		ranker := NewResultRanker()

		Convey("When calculating diversity scores for similar content", func() {
			results := []RankableResult{
				{ID: "doc1", Content: "Machine learning algorithms and neural networks"},
				{ID: "doc2", Content: "Machine learning algorithms and deep learning"},
				{ID: "doc3", Content: "Cooking recipes and kitchen tips"},
			}
			context := &RankingContext{}

			ranker.calculateDiversityScores(results, context)

			// doc1 and doc2 should have lower diversity scores due to similarity
			// doc3 should have higher diversity score as it's different
			So(results[2].DiversityScore, ShouldBeGreaterThan, results[0].DiversityScore)
			So(results[2].DiversityScore, ShouldBeGreaterThan, results[1].DiversityScore)
		})

		Convey("When calculating diversity scores for single result", func() {
			results := []RankableResult{
				{ID: "doc1", Content: "Single document"},
			}
			context := &RankingContext{}

			ranker.calculateDiversityScores(results, context)

			So(results[0].DiversityScore, ShouldEqual, 1.0)
		})
	})
}

func TestResultRanker_BoostsAndPenalties(t *testing.T) {
	Convey("Given a ResultRanker", t, func() {
		ranker := NewResultRanker()

		Convey("When applying boosts to high-scoring result", func() {
			result := &RankableResult{
				ID:             "doc1",
				FinalScore:     0.85, // Above boost threshold
				AuthorityScore: 0.85, // Above authority boost threshold
				FreshnessScore: 0.95, // Above freshness boost threshold
			}
			context := &RankingContext{}

			originalScore := result.FinalScore
			ranker.applyBoostsAndPenalties(result, context)

			So(result.FinalScore, ShouldBeGreaterThan, originalScore)
			So(len(result.Boosts), ShouldBeGreaterThan, 0)
			So(result.Boosts, ShouldContain, "high_score_boost")
			So(result.Boosts, ShouldContain, "authority_boost")
			So(result.Boosts, ShouldContain, "freshness_boost")
		})

		Convey("When applying penalties to low-scoring result", func() {
			result := &RankableResult{
				ID:           "doc1",
				FinalScore:   0.25, // Below penalty threshold
				QualityScore: 0.25, // Below quality penalty threshold
			}
			context := &RankingContext{}

			originalScore := result.FinalScore
			ranker.applyBoostsAndPenalties(result, context)

			So(result.FinalScore, ShouldBeLessThan, originalScore)
			So(len(result.Penalties), ShouldBeGreaterThan, 0)
			So(result.Penalties, ShouldContain, "low_score_penalty")
			So(result.Penalties, ShouldContain, "quality_penalty")
		})

		Convey("When final score stays within bounds", func() {
			result := &RankableResult{
				ID:         "doc1",
				FinalScore: 0.95,
			}
			context := &RankingContext{}

			ranker.applyBoostsAndPenalties(result, context)

			So(result.FinalScore, ShouldBeLessThanOrEqualTo, 1.0)
			So(result.FinalScore, ShouldBeGreaterThanOrEqualTo, 0.0)
		})
	})
}

func TestResultRanker_Configuration(t *testing.T) {
	Convey("Given a ResultRanker", t, func() {
		ranker := NewResultRanker()

		Convey("When getting configuration", func() {
			config := ranker.GetConfig()
			So(config, ShouldNotBeNil)
			So(config.RelevanceWeight, ShouldEqual, 1.0)
		})

		Convey("When updating configuration", func() {
			newConfig := &ResultRankerConfig{
				RelevanceWeight: 2.0,
				FreshnessWeight: 0.5,
				MaxResults:      50,
			}

			ranker.UpdateConfig(newConfig)
			config := ranker.GetConfig()

			So(config.RelevanceWeight, ShouldEqual, 2.0)
			So(config.FreshnessWeight, ShouldEqual, 0.5)
			So(config.MaxResults, ShouldEqual, 50)
		})

		Convey("When updating with nil config", func() {
			originalConfig := ranker.GetConfig()
			originalWeight := originalConfig.RelevanceWeight

			ranker.UpdateConfig(nil)
			config := ranker.GetConfig()

			So(config.RelevanceWeight, ShouldEqual, originalWeight)
		})
	})
}