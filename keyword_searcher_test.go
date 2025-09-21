package main

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestKeywordSearcher(t *testing.T) {
	Convey("Given a KeywordSearcher", t, func() {
		ks := NewKeywordSearcher()

		Convey("When creating a new KeywordSearcher", func() {
			So(ks, ShouldNotBeNil)
			So(ks.config, ShouldNotBeNil)
			So(ks.config.DefaultK, ShouldEqual, 10)
			So(ks.config.MinScore, ShouldEqual, 0.1)
			So(ks.config.BM25K1, ShouldEqual, 1.2)
			So(ks.config.BM25B, ShouldEqual, 0.75)
		})

		Convey("When creating with custom config", func() {
			config := &KeywordSearchConfig{
				DefaultK:      5,
				MinScore:      0.5,
				MaxResults:    50,
				BM25K1:        2.0,
				BM25B:         0.5,
				CaseSensitive: true,
				StemWords:     true,
			}
			ks := NewKeywordSearcherWithIndex(nil, config)

			So(ks.config.DefaultK, ShouldEqual, 5)
			So(ks.config.MinScore, ShouldEqual, 0.5)
			So(ks.config.BM25K1, ShouldEqual, 2.0)
			So(ks.config.CaseSensitive, ShouldBeTrue)
			So(ks.config.StemWords, ShouldBeTrue)
		})
	})
}

func TestKeywordSearcherSearch(t *testing.T) {
	Convey("Given a KeywordSearcher with mock index", t, func() {
		mockIndex := NewMockSearchIndex()
		ks := NewKeywordSearcherWithIndex(mockIndex, nil)

		// Add some test data
		ctx := context.Background()
		mockIndex.Index(ctx, IndexDocument{ID: "doc1", Content: "machine learning algorithms for data science", Metadata: map[string]interface{}{"title": "ML Guide"}})
		mockIndex.Index(ctx, IndexDocument{ID: "doc2", Content: "artificial intelligence and neural networks", Metadata: map[string]interface{}{"title": "AI Overview"}})
		mockIndex.Index(ctx, IndexDocument{ID: "doc3", Content: "deep learning with python programming", Metadata: map[string]interface{}{"title": "Deep Learning"}})

		Convey("When searching with valid query", func() {
			ctx := context.Background()
			query := "machine learning"
			k := 2

			result, err := ks.Search(ctx, query, k, nil)

			Convey("Then it should return search results", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(result.Metadata, ShouldNotBeNil)
				So(result.Metadata["query_terms"], ShouldNotBeNil)
				So(result.Metadata["processed_query"], ShouldNotBeNil)
			})
		})

		Convey("When searching with empty query", func() {
			ctx := context.Background()
			query := ""
			k := 2

			result, err := ks.Search(ctx, query, k, nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
			})
		})

		Convey("When searching with whitespace-only query", func() {
			ctx := context.Background()
			query := "   "
			k := 2

			result, err := ks.Search(ctx, query, k, nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
			})
		})

		Convey("When searching with zero k", func() {
			ctx := context.Background()
			query := "machine learning"
			k := 0

			result, err := ks.Search(ctx, query, k, nil)

			Convey("Then it should use default k", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
			})
		})

		Convey("When searching without search index", func() {
			ksNoIndex := NewKeywordSearcher()
			ctx := context.Background()
			query := "test query"

			result, err := ksNoIndex.Search(ctx, query, 5, nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
			})
		})
	})
}

func TestKeywordSearcherMatchKeywords(t *testing.T) {
	Convey("Given a KeywordSearcher", t, func() {
		ks := NewKeywordSearcher()

		Convey("When matching keywords in content", func() {
			content := "Machine learning algorithms are used in artificial intelligence applications"
			queryTerms := []string{"machine", "learning", "algorithms"}

			matchedTerms, highlights := ks.MatchKeywords(content, queryTerms)

			Convey("Then it should find matching terms", func() {
				So(len(matchedTerms), ShouldBeGreaterThan, 0)
				So(matchedTerms, ShouldContain, "machine")
				So(matchedTerms, ShouldContain, "learning")
				So(matchedTerms, ShouldContain, "algorithms")
				So(len(highlights), ShouldBeGreaterThan, 0)
			})
		})

		Convey("When matching keywords with no matches", func() {
			content := "This is completely different content"
			queryTerms := []string{"machine", "learning", "algorithms"}

			matchedTerms, highlights := ks.MatchKeywords(content, queryTerms)

			Convey("Then it should return empty results", func() {
				So(len(matchedTerms), ShouldEqual, 0)
				So(len(highlights), ShouldEqual, 0)
			})
		})

		Convey("When matching with empty content", func() {
			content := ""
			queryTerms := []string{"machine", "learning"}

			matchedTerms, highlights := ks.MatchKeywords(content, queryTerms)

			Convey("Then it should return empty results", func() {
				So(len(matchedTerms), ShouldEqual, 0)
				So(len(highlights), ShouldEqual, 0)
			})
		})

		Convey("When matching with empty query terms", func() {
			content := "Some content here"
			queryTerms := []string{}

			matchedTerms, highlights := ks.MatchKeywords(content, queryTerms)

			Convey("Then it should return empty results", func() {
				So(len(matchedTerms), ShouldEqual, 0)
				So(len(highlights), ShouldEqual, 0)
			})
		})
	})
}

func TestKeywordSearcherScoreResults(t *testing.T) {
	Convey("Given a KeywordSearcher", t, func() {
		ks := NewKeywordSearcher()

		Convey("When scoring and ranking results", func() {
			results := []KeywordSearchResult{
				{ID: "doc1", Score: 0.5, BM25Score: 0.5, MatchedTerms: []string{"machine"}},
				{ID: "doc2", Score: 0.9, BM25Score: 0.9, MatchedTerms: []string{"machine", "learning"}},
				{ID: "doc3", Score: 0.3, BM25Score: 0.3, MatchedTerms: []string{"learning"}},
			}
			queryTerms := []string{"machine", "learning"}

			scored := ks.ScoreResults(results, queryTerms)

			Convey("Then results should be ranked by enhanced score", func() {
				So(len(scored), ShouldEqual, 3)
				So(scored[0].Score, ShouldBeGreaterThanOrEqualTo, scored[1].Score)
				So(scored[1].Score, ShouldBeGreaterThanOrEqualTo, scored[2].Score)
			})
		})

		Convey("When scoring results with identical scores", func() {
			results := []KeywordSearchResult{
				{ID: "doc1", Score: 0.5, BM25Score: 0.5, MatchedTerms: []string{"machine"}},
				{ID: "doc2", Score: 0.5, BM25Score: 0.5, MatchedTerms: []string{"machine", "learning"}},
			}
			queryTerms := []string{"machine", "learning"}

			scored := ks.ScoreResults(results, queryTerms)

			Convey("Then results should use match count as tiebreaker", func() {
				So(len(scored), ShouldEqual, 2)
				So(len(scored[0].MatchedTerms), ShouldBeGreaterThanOrEqualTo, len(scored[1].MatchedTerms))
			})
		})

		Convey("When scoring empty results", func() {
			results := []KeywordSearchResult{}
			queryTerms := []string{"machine", "learning"}

			scored := ks.ScoreResults(results, queryTerms)

			Convey("Then it should return empty slice", func() {
				So(len(scored), ShouldEqual, 0)
			})
		})
	})
}

func TestKeywordSearcherTextProcessing(t *testing.T) {
	Convey("Given a KeywordSearcher", t, func() {
		ks := NewKeywordSearcher()

		Convey("When preprocessing text", func() {
			text := "  Machine Learning   Algorithms  "
			processed := ks.preprocessText(text)

			Convey("Then it should normalize whitespace and case", func() {
				So(processed, ShouldEqual, "machine learning algorithms")
			})
		})

		Convey("When tokenizing text", func() {
			text := "Machine learning, algorithms & neural networks!"
			tokens := ks.tokenize(text)

			Convey("Then it should split on punctuation and filter stop words", func() {
				So(tokens, ShouldContain, "machine")
				So(tokens, ShouldContain, "learning")
				So(tokens, ShouldContain, "algorithms")
				So(tokens, ShouldContain, "neural")
				So(tokens, ShouldContain, "networks")
				So(tokens, ShouldNotContain, "&")
				So(tokens, ShouldNotContain, "!")
			})
		})

		Convey("When filtering stop words", func() {
			tokens := []string{"the", "machine", "learning", "is", "good", "and", "useful"}
			filtered := ks.filterStopWords(tokens)

			Convey("Then it should remove common stop words", func() {
				So(filtered, ShouldNotContain, "the")
				So(filtered, ShouldNotContain, "is")
				So(filtered, ShouldNotContain, "and")
				So(filtered, ShouldContain, "machine")
				So(filtered, ShouldContain, "learning")
				So(filtered, ShouldContain, "good")
				So(filtered, ShouldContain, "useful")
			})
		})

		Convey("When generating highlights", func() {
			content := "Machine learning algorithms are powerful tools for data analysis and pattern recognition"
			term := "learning"

			highlight := ks.generateHighlight(content, term)

			Convey("Then it should create a snippet around the term", func() {
				So(highlight, ShouldNotBeEmpty)
				So(highlight, ShouldContainSubstring, "learning")
			})
		})

		Convey("When generating highlight for non-existent term", func() {
			content := "Machine learning algorithms"
			term := "nonexistent"

			highlight := ks.generateHighlight(content, term)

			Convey("Then it should return empty string", func() {
				So(highlight, ShouldBeEmpty)
			})
		})
	})
}

func TestKeywordSearcherBM25Scoring(t *testing.T) {
	Convey("Given a KeywordSearcher", t, func() {
		ks := NewKeywordSearcher()

		Convey("When calculating BM25 score", func() {
			content := "machine learning algorithms for data science and machine learning applications"
			queryTerms := []string{"machine", "learning"}
			metadata := map[string]interface{}{"avg_doc_length": 50.0}

			score := ks.calculateBM25Score(content, queryTerms, metadata)

			Convey("Then it should return a positive score", func() {
				So(score, ShouldBeGreaterThan, 0)
			})
		})

		Convey("When calculating BM25 score with empty content", func() {
			content := ""
			queryTerms := []string{"machine", "learning"}
			metadata := map[string]interface{}{}

			score := ks.calculateBM25Score(content, queryTerms, metadata)

			Convey("Then it should return zero score", func() {
				So(score, ShouldEqual, 0.0)
			})
		})

		Convey("When calculating BM25 score with no matching terms", func() {
			content := "artificial intelligence and neural networks"
			queryTerms := []string{"database", "storage"}
			metadata := map[string]interface{}{}

			score := ks.calculateBM25Score(content, queryTerms, metadata)

			Convey("Then it should return zero score", func() {
				So(score, ShouldEqual, 0.0)
			})
		})

		Convey("When calculating enhanced score", func() {
			result := KeywordSearchResult{
				BM25Score:    1.5,
				Score:        1.0,
				MatchedTerms: []string{"machine", "learning"},
				Content:      "Short content about machine learning",
			}
			queryTerms := []string{"machine", "learning", "algorithms"}

			enhancedScore := ks.calculateEnhancedScore(result, queryTerms)

			Convey("Then it should return enhanced score with boosts", func() {
				So(enhancedScore, ShouldBeGreaterThan, result.BM25Score)
			})
		})
	})
}

func TestKeywordSearcherSearchMultiple(t *testing.T) {
	Convey("Given a KeywordSearcher with mock index", t, func() {
		mockIndex := NewMockSearchIndex()
		ks := NewKeywordSearcherWithIndex(mockIndex, nil)

		// Add test data
		ctx := context.Background()
		mockIndex.Index(ctx, IndexDocument{ID: "doc1", Content: "machine learning algorithms", Metadata: map[string]interface{}{}})
		mockIndex.Index(ctx, IndexDocument{ID: "doc2", Content: "artificial intelligence systems", Metadata: map[string]interface{}{}})

		Convey("When searching with multiple queries", func() {
			ctx := context.Background()
			queries := []string{"machine learning", "artificial intelligence"}
			k := 5

			result, err := ks.SearchMultiple(ctx, queries, k, nil)

			Convey("Then it should return merged results", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(result.Metadata["query_count"], ShouldEqual, 2)
				So(result.Metadata["deduplication"], ShouldBeTrue)
				So(result.Metadata["combined_terms"], ShouldNotBeNil)
			})
		})

		Convey("When searching with empty queries", func() {
			ctx := context.Background()
			queries := []string{}
			k := 5

			result, err := ks.SearchMultiple(ctx, queries, k, nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
			})
		})
	})
}

func TestKeywordSearcherHelperMethods(t *testing.T) {
	Convey("Given a KeywordSearcher", t, func() {
		ks := NewKeywordSearcher()

		Convey("When deduplicating strings", func() {
			strings := []string{"apple", "banana", "apple", "cherry", "banana"}
			deduplicated := ks.deduplicateStrings(strings)

			Convey("Then it should remove duplicates while preserving order", func() {
				So(len(deduplicated), ShouldEqual, 3)
				So(deduplicated[0], ShouldEqual, "apple")
				So(deduplicated[1], ShouldEqual, "banana")
				So(deduplicated[2], ShouldEqual, "cherry")
			})
		})

		Convey("When deduplicating with empty strings", func() {
			strings := []string{"apple", "", "banana", "  ", "cherry"}
			deduplicated := ks.deduplicateStrings(strings)

			Convey("Then it should filter out empty strings", func() {
				So(len(deduplicated), ShouldEqual, 3)
				So(deduplicated, ShouldContain, "apple")
				So(deduplicated, ShouldContain, "banana")
				So(deduplicated, ShouldContain, "cherry")
			})
		})
	})
}

// Benchmark tests
func BenchmarkKeywordSearcherTokenize(b *testing.B) {
	ks := NewKeywordSearcher()
	text := "Machine learning algorithms are powerful tools for data analysis, pattern recognition, and artificial intelligence applications in various domains"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ks.tokenize(text)
	}
}

func BenchmarkKeywordSearcherMatchKeywords(b *testing.B) {
	ks := NewKeywordSearcher()
	content := "Machine learning algorithms are used in artificial intelligence applications for data analysis and pattern recognition tasks"
	queryTerms := []string{"machine", "learning", "algorithms", "artificial", "intelligence"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ks.MatchKeywords(content, queryTerms)
	}
}

func BenchmarkKeywordSearcherBM25Score(b *testing.B) {
	ks := NewKeywordSearcher()
	content := "Machine learning algorithms are powerful tools for data analysis, pattern recognition, and artificial intelligence applications in various domains including computer vision, natural language processing, and robotics"
	queryTerms := []string{"machine", "learning", "algorithms", "data", "analysis"}
	metadata := map[string]interface{}{"avg_doc_length": 100.0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ks.calculateBM25Score(content, queryTerms, metadata)
	}
}