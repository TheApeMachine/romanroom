package main

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestQueryProcessor(t *testing.T) {
	Convey("Given a QueryProcessor", t, func() {
		config := &QueryProcessorConfig{
			MaxResults:      20,
			DefaultTimeout:  5 * time.Second,
			EnableExpansion: true,
			MinQueryLength:  2,
		}
		qp := NewQueryProcessor(config)

		Convey("When creating a new QueryProcessor", func() {
			So(qp, ShouldNotBeNil)
			So(qp.config, ShouldNotBeNil)
			So(qp.vectorSearcher, ShouldNotBeNil)
			So(qp.keywordSearcher, ShouldNotBeNil)
			So(qp.queryExpander, ShouldNotBeNil)
		})

		Convey("When processing a valid query", func() {
			ctx := context.Background()
			query := "machine learning algorithms"
			options := &RecallOptions{MaxResults: 10}

			result, err := qp.Process(ctx, query, options)

			Convey("Then it should return a processed query", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(result.Original, ShouldEqual, query)
				So(result.Parsed, ShouldNotBeNil)
				So(len(result.Expanded), ShouldBeGreaterThan, 0)
				So(result.Metadata, ShouldNotBeNil)
			})
		})

		Convey("When processing an empty query", func() {
			ctx := context.Background()
			query := ""
			options := &RecallOptions{MaxResults: 10}

			result, err := qp.Process(ctx, query, options)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
			})
		})

		Convey("When processing a query that's too short", func() {
			ctx := context.Background()
			query := "a"
			options := &RecallOptions{MaxResults: 10}

			result, err := qp.Process(ctx, query, options)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
			})
		})
	})
}

func TestQueryProcessorParse(t *testing.T) {
	Convey("Given a QueryProcessor", t, func() {
		qp := NewQueryProcessor(nil)

		Convey("When parsing a simple keyword query", func() {
			query := "machine learning"
			result, err := qp.Parse(query)

			Convey("Then it should extract terms correctly", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(result.Terms, ShouldContain, "machine")
				So(result.Terms, ShouldContain, "learning")
				So(result.QueryType, ShouldEqual, QueryTypeKeyword)
			})
		})

		Convey("When parsing a query with phrases", func() {
			query := `"artificial intelligence" and machine learning`
			result, err := qp.Parse(query)

			Convey("Then it should extract phrases correctly", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(result.Phrases, ShouldContain, "artificial intelligence")
				So(result.Terms, ShouldContain, "machine")
				So(result.Terms, ShouldContain, "learning")
				So(result.QueryType, ShouldEqual, QueryTypeSemantic)
			})
		})

		Convey("When parsing a query with filters", func() {
			query := "machine learning type:algorithm date:recent"
			result, err := qp.Parse(query)

			Convey("Then it should extract filters correctly", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(result.Filters["type"], ShouldEqual, "algorithm")
				So(result.Filters["date"], ShouldEqual, "recent")
				So(result.QueryType, ShouldEqual, QueryTypeHybrid)
			})
		})

		Convey("When parsing a query with time keywords", func() {
			query := "recent developments in AI"
			result, err := qp.Parse(query)

			Convey("Then it should extract time range", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(result.TimeRange, ShouldNotBeNil)
				So(result.TimeRange.Start, ShouldNotBeNil)
				So(result.TimeRange.End, ShouldNotBeNil)
			})
		})

		Convey("When parsing an entity-like query", func() {
			query := "OpenAI GPT"
			result, err := qp.Parse(query)

			Convey("Then it should identify as entity query", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(result.QueryType, ShouldEqual, QueryTypeEntity)
			})
		})

		Convey("When parsing an empty query", func() {
			query := ""
			result, err := qp.Parse(query)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
			})
		})
	})
}

func TestQueryProcessorExpand(t *testing.T) {
	Convey("Given a QueryProcessor with expansion enabled", t, func() {
		config := &QueryProcessorConfig{
			EnableExpansion: true,
		}
		qp := NewQueryProcessor(config)

		Convey("When expanding a simple query", func() {
			ctx := context.Background()
			query := "machine learning"
			parsed, _ := qp.Parse(query)

			result, err := qp.Expand(ctx, query, parsed)

			Convey("Then it should return expanded queries", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(len(result), ShouldBeGreaterThan, 1)
				So(result[0], ShouldEqual, query) // Original should be first
			})
		})

		Convey("When expanding with nil parsed query", func() {
			ctx := context.Background()
			query := "test query"

			result, err := qp.Expand(ctx, query, nil)

			Convey("Then it should still return the original query", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(len(result), ShouldBeGreaterThanOrEqualTo, 1)
				So(result[0], ShouldEqual, query)
			})
		})
	})

	Convey("Given a QueryProcessor with expansion disabled", t, func() {
		config := &QueryProcessorConfig{
			EnableExpansion: false,
		}
		qp := NewQueryProcessor(config)

		Convey("When trying to expand a query", func() {
			ctx := context.Background()
			query := "machine learning"
			parsed, _ := qp.Parse(query)

			result, err := qp.Expand(ctx, query, parsed)

			Convey("Then it should return only the original query", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(len(result), ShouldEqual, 1)
				So(result[0], ShouldEqual, query)
			})
		})
	})
}

func TestQueryProcessorHelperMethods(t *testing.T) {
	Convey("Given a QueryProcessor", t, func() {
		qp := NewQueryProcessor(nil)

		Convey("When extracting phrases from quoted text", func() {
			query := `This is "a quoted phrase" and "another phrase" here`
			phrases := qp.extractPhrases(query)

			Convey("Then it should extract all quoted phrases", func() {
				So(len(phrases), ShouldEqual, 2)
				So(phrases, ShouldContain, "a quoted phrase")
				So(phrases, ShouldContain, "another phrase")
			})
		})

		Convey("When removing phrases from query", func() {
			query := `This is "a quoted phrase" and some text`
			result := qp.removePhrases(query)

			Convey("Then it should remove quoted content", func() {
				So(result, ShouldNotContainSubstring, "a quoted phrase")
				So(result, ShouldContainSubstring, "This is")
				So(result, ShouldContainSubstring, "and some text")
			})
		})

		Convey("When extracting filters", func() {
			query := "search query type:document author:john date:2023"
			filters := qp.extractFilters(query)

			Convey("Then it should extract all key:value pairs", func() {
				So(len(filters), ShouldEqual, 3)
				So(filters["type"], ShouldEqual, "document")
				So(filters["author"], ShouldEqual, "john")
				So(filters["date"], ShouldEqual, "2023")
			})
		})

		Convey("When filtering terms", func() {
			terms := []string{"the", "machine", "learning", "a", "algorithm", "is"}
			filtered := qp.filterTerms(terms)

			Convey("Then it should remove stop words and short terms", func() {
				So(filtered, ShouldNotContain, "the")
				So(filtered, ShouldNotContain, "a")
				So(filtered, ShouldNotContain, "is")
				So(filtered, ShouldContain, "machine")
				So(filtered, ShouldContain, "learning")
				So(filtered, ShouldContain, "algorithm")
			})
		})

		Convey("When extracting entities from parsed query", func() {
			parsed := &ParsedQuery{
				Terms:   []string{"OpenAI", "machine", "learning"},
				Phrases: []string{"Artificial Intelligence"},
			}
			entities := qp.extractEntities(parsed)

			Convey("Then it should identify capitalized terms and phrases", func() {
				So(entities, ShouldContain, "OpenAI")
				So(entities, ShouldContain, "Artificial Intelligence")
				So(entities, ShouldNotContain, "machine")
				So(entities, ShouldNotContain, "learning")
			})
		})

		Convey("When deduplicating queries", func() {
			queries := []string{"query1", "query2", "query1", "query3", "query2"}
			deduplicated := qp.deduplicateQueries(queries)

			Convey("Then it should remove duplicates while preserving order", func() {
				So(len(deduplicated), ShouldEqual, 3)
				So(deduplicated[0], ShouldEqual, "query1")
				So(deduplicated[1], ShouldEqual, "query2")
				So(deduplicated[2], ShouldEqual, "query3")
			})
		})
	})
}

// Benchmark tests
func BenchmarkQueryProcessorProcess(b *testing.B) {
	qp := NewQueryProcessor(nil)
	ctx := context.Background()
	query := "machine learning algorithms for natural language processing"
	options := &RecallOptions{MaxResults: 10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = qp.Process(ctx, query, options)
	}
}

func BenchmarkQueryProcessorParse(b *testing.B) {
	qp := NewQueryProcessor(nil)
	query := "machine learning algorithms for natural language processing with filters type:research date:recent"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = qp.Parse(query)
	}
}

func BenchmarkQueryProcessorExpand(b *testing.B) {
	qp := NewQueryProcessor(nil)
	ctx := context.Background()
	query := "machine learning algorithms"
	parsed, _ := qp.Parse(query)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = qp.Expand(ctx, query, parsed)
	}
}