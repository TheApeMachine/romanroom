package main

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestQueryExpander(t *testing.T) {
	Convey("Given a QueryExpander", t, func() {
		qe := NewQueryExpander()

		Convey("When creating a new QueryExpander", func() {
			So(qe, ShouldNotBeNil)
			So(qe.config, ShouldNotBeNil)
			So(qe.config.MaxExpansions, ShouldEqual, 8)
			So(qe.config.EnableSynonyms, ShouldBeTrue)
			So(qe.config.EnableParaphrases, ShouldBeTrue)
			So(qe.config.EnableSpelling, ShouldBeTrue)
			So(qe.config.EnableAcronyms, ShouldBeTrue)
		})

		Convey("When creating with custom config", func() {
			config := &QueryExpanderConfig{
				MaxExpansions:       5,
				EnableSynonyms:      false,
				EnableParaphrases:   true,
				EnableSpelling:      false,
				EnableAcronyms:      true,
				SimilarityThreshold: 0.8,
			}
			qe := NewQueryExpanderWithConfig(config)

			So(qe.config.MaxExpansions, ShouldEqual, 5)
			So(qe.config.EnableSynonyms, ShouldBeFalse)
			So(qe.config.EnableParaphrases, ShouldBeTrue)
			So(qe.config.EnableSpelling, ShouldBeFalse)
			So(qe.config.SimilarityThreshold, ShouldEqual, 0.8)
		})

		Convey("When creating with nil config", func() {
			qe := NewQueryExpanderWithConfig(nil)

			So(qe, ShouldNotBeNil)
			So(qe.config, ShouldNotBeNil)
		})
	})
}

func TestQueryExpanderExpand(t *testing.T) {
	Convey("Given a QueryExpander", t, func() {
		qe := NewQueryExpander()

		Convey("When expanding a simple query", func() {
			ctx := context.Background()
			query := "machine learning"
			parsed := &ParsedQuery{
				Terms:     []string{"machine", "learning"},
				QueryType: QueryTypeKeyword,
			}

			expansions, err := qe.Expand(ctx, query, parsed)

			Convey("Then it should return expanded queries", func() {
				So(err, ShouldBeNil)
				So(expansions, ShouldNotBeNil)
				So(len(expansions), ShouldBeGreaterThan, 0)
			})
		})

		Convey("When expanding with empty query", func() {
			ctx := context.Background()
			query := ""
			parsed := &ParsedQuery{}

			expansions, err := qe.Expand(ctx, query, parsed)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(expansions, ShouldNotBeNil)
			})
		})

		Convey("When expanding with whitespace-only query", func() {
			ctx := context.Background()
			query := "   "
			parsed := &ParsedQuery{}

			expansions, err := qe.Expand(ctx, query, parsed)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(expansions, ShouldNotBeNil)
			})
		})

		Convey("When expanding with nil parsed query", func() {
			ctx := context.Background()
			query := "test query"

			expansions, err := qe.Expand(ctx, query, nil)

			Convey("Then it should still work with limited expansions", func() {
				So(err, ShouldBeNil)
				So(expansions, ShouldNotBeNil)
				So(len(expansions), ShouldBeGreaterThan, 0)
			})
		})
	})
}

func TestQueryExpanderGenerateSynonyms(t *testing.T) {
	Convey("Given a QueryExpander", t, func() {
		qe := NewQueryExpander()

		Convey("When generating synonyms for terms with known synonyms", func() {
			query := "find good algorithms"
			parsed := &ParsedQuery{
				Terms: []string{"find", "good", "algorithms"},
			}

			synonyms := qe.GenerateSynonyms(query, parsed)

			Convey("Then it should generate synonym variations", func() {
				So(synonyms, ShouldNotBeNil)
				// Should contain variations with synonyms for "find" and "good"
				foundSynonym := false
				for _, synonym := range synonyms {
					if synonym != query {
						foundSynonym = true
						break
					}
				}
				So(foundSynonym, ShouldBeTrue)
			})
		})

		Convey("When generating synonyms for phrases", func() {
			query := `"machine learning" is good`
			parsed := &ParsedQuery{
				Terms:   []string{"machine", "learning", "good"},
				Phrases: []string{"machine learning"},
			}

			synonyms := qe.GenerateSynonyms(query, parsed)

			Convey("Then it should generate phrase synonym variations", func() {
				So(synonyms, ShouldNotBeNil)
			})
		})

		Convey("When generating synonyms with no known synonyms", func() {
			query := "specialized technical terminology"
			parsed := &ParsedQuery{
				Terms: []string{"specialized", "technical", "terminology"},
			}

			synonyms := qe.GenerateSynonyms(query, parsed)

			Convey("Then it should return empty or minimal results", func() {
				So(synonyms, ShouldNotBeNil)
				So(len(synonyms), ShouldBeGreaterThanOrEqualTo, 0)
			})
		})

		Convey("When generating synonyms with nil parsed query", func() {
			query := "test query"

			synonyms := qe.GenerateSynonyms(query, nil)

			Convey("Then it should return empty results", func() {
				So(len(synonyms), ShouldEqual, 0)
			})
		})
	})
}

func TestQueryExpanderAddContext(t *testing.T) {
	Convey("Given a QueryExpander", t, func() {
		qe := NewQueryExpander()

		Convey("When adding context to entity query", func() {
			query := "OpenAI"
			parsed := &ParsedQuery{
				Terms:     []string{"OpenAI"},
				QueryType: QueryTypeEntity,
			}

			contextQueries := qe.AddContext(query, parsed)

			Convey("Then it should add entity-specific context", func() {
				So(contextQueries, ShouldNotBeNil)
				So(len(contextQueries), ShouldBeGreaterThan, 0)
				
				// Should contain entity-specific variations
				foundEntityContext := false
				for _, cq := range contextQueries {
					if cq == "OpenAI information" || cq == "about OpenAI" {
						foundEntityContext = true
						break
					}
				}
				So(foundEntityContext, ShouldBeTrue)
			})
		})

		Convey("When adding context to semantic query", func() {
			query := "what is machine learning"
			parsed := &ParsedQuery{
				Terms:     []string{"what", "machine", "learning"},
				QueryType: QueryTypeSemantic,
			}

			contextQueries := qe.AddContext(query, parsed)

			Convey("Then it should add semantic-specific context", func() {
				So(contextQueries, ShouldNotBeNil)
				So(len(contextQueries), ShouldBeGreaterThan, 0)
			})
		})

		Convey("When adding context with time range", func() {
			query := "recent AI developments"
			timeRange := &TimeRange{} // Non-nil time range
			parsed := &ParsedQuery{
				Terms:     []string{"recent", "developments"},
				TimeRange: timeRange,
				QueryType: QueryTypeKeyword,
			}

			contextQueries := qe.AddContext(query, parsed)

			Convey("Then it should add temporal context", func() {
				So(contextQueries, ShouldNotBeNil)
				So(len(contextQueries), ShouldBeGreaterThan, 0)
				
				// Should contain temporal variations
				foundTemporalContext := false
				for _, cq := range contextQueries {
					if cq == "recent AI developments recent" || cq == "recent AI developments latest" {
						foundTemporalContext = true
						break
					}
				}
				So(foundTemporalContext, ShouldBeTrue)
			})
		})

		Convey("When adding context with nil parsed query", func() {
			query := "test query"

			contextQueries := qe.AddContext(query, nil)

			Convey("Then it should still add basic question variations", func() {
				So(contextQueries, ShouldNotBeNil)
				So(len(contextQueries), ShouldBeGreaterThan, 0)
				
				// Should contain question variations
				foundQuestionContext := false
				for _, cq := range contextQueries {
					if cq == "what is test query" {
						foundQuestionContext = true
						break
					}
				}
				So(foundQuestionContext, ShouldBeTrue)
			})
		})
	})
}

func TestQueryExpanderGenerateParaphrases(t *testing.T) {
	Convey("Given a QueryExpander", t, func() {
		qe := NewQueryExpander()

		Convey("When generating paraphrases for queries with known patterns", func() {
			query := "how to learn machine learning"
			parsed := &ParsedQuery{
				Terms: []string{"how", "learn", "machine", "learning"},
			}

			paraphrases := qe.generateParaphrases(query, parsed)

			Convey("Then it should generate paraphrased variations", func() {
				So(paraphrases, ShouldNotBeNil)
				// Should contain paraphrases for "how to"
				foundParaphrase := false
				for _, paraphrase := range paraphrases {
					if paraphrase != query {
						foundParaphrase = true
						break
					}
				}
				So(foundParaphrase, ShouldBeTrue)
			})
		})

		Convey("When generating paraphrases with term reordering", func() {
			query := "machine learning algorithms"
			parsed := &ParsedQuery{
				Terms: []string{"machine", "learning", "algorithms"},
			}

			paraphrases := qe.generateParaphrases(query, parsed)

			Convey("Then it should include reordered variations", func() {
				So(paraphrases, ShouldNotBeNil)
				// Should contain reordered terms
				foundReordered := false
				for _, paraphrase := range paraphrases {
					if paraphrase == "algorithms learning machine" {
						foundReordered = true
						break
					}
				}
				So(foundReordered, ShouldBeTrue)
			})
		})

		Convey("When generating paraphrases with no known patterns", func() {
			query := "specialized technical terminology"
			parsed := &ParsedQuery{
				Terms: []string{"specialized", "technical", "terminology"},
			}

			paraphrases := qe.generateParaphrases(query, parsed)

			Convey("Then it should still attempt structural paraphrases", func() {
				So(paraphrases, ShouldNotBeNil)
			})
		})
	})
}

func TestQueryExpanderGenerateSpellingVariations(t *testing.T) {
	Convey("Given a QueryExpander", t, func() {
		qe := NewQueryExpander()

		Convey("When generating spelling variations for known misspellings", func() {
			query := "recieve information about performace"
			parsed := &ParsedQuery{
				Terms: []string{"recieve", "information", "performace"},
			}

			spellings := qe.generateSpellingVariations(query, parsed)

			Convey("Then it should generate corrected spellings", func() {
				So(spellings, ShouldNotBeNil)
				// Should contain corrections for "recieve" -> "receive" and "performace" -> "performance"
				foundCorrection := false
				for _, spelling := range spellings {
					if spelling != query {
						foundCorrection = true
						break
					}
				}
				So(foundCorrection, ShouldBeTrue)
			})
		})

		Convey("When generating typo variations for longer terms", func() {
			query := "machine learning"
			parsed := &ParsedQuery{
				Terms: []string{"machine", "learning"},
			}

			spellings := qe.generateSpellingVariations(query, parsed)

			Convey("Then it should generate typo variations", func() {
				So(spellings, ShouldNotBeNil)
			})
		})

		Convey("When generating spelling variations with short terms", func() {
			query := "ai ml"
			parsed := &ParsedQuery{
				Terms: []string{"ai", "ml"},
			}

			spellings := qe.generateSpellingVariations(query, parsed)

			Convey("Then it should not generate variations for short terms", func() {
				So(spellings, ShouldNotBeNil)
				// Short terms should not have typo variations
			})
		})
	})
}

func TestQueryExpanderGenerateAcronymExpansions(t *testing.T) {
	Convey("Given a QueryExpander", t, func() {
		qe := NewQueryExpander()

		Convey("When generating acronym expansions for known acronyms", func() {
			query := "AI and ML algorithms"
			parsed := &ParsedQuery{
				Terms: []string{"ai", "ml", "algorithms"},
			}

			acronyms := qe.generateAcronymExpansions(query, parsed)

			Convey("Then it should expand known acronyms", func() {
				So(acronyms, ShouldNotBeNil)
				// Should contain expansions for AI and ML
				foundExpansion := false
				for _, acronym := range acronyms {
					if acronym != query {
						foundExpansion = true
						break
					}
				}
				So(foundExpansion, ShouldBeTrue)
			})
		})

		Convey("When generating acronyms from phrases", func() {
			query := `"natural language processing" systems`
			parsed := &ParsedQuery{
				Terms:   []string{"natural", "language", "processing", "systems"},
				Phrases: []string{"natural language processing"},
			}

			acronyms := qe.generateAcronymExpansions(query, parsed)

			Convey("Then it should create acronyms from multi-word phrases", func() {
				So(acronyms, ShouldNotBeNil)
				// Should contain "nlp systems"
				foundAcronym := false
				for _, acronym := range acronyms {
					if acronym == "nlp systems" {
						foundAcronym = true
						break
					}
				}
				So(foundAcronym, ShouldBeTrue)
			})
		})

		Convey("When generating acronym expansions with unknown acronyms", func() {
			query := "xyz abc systems"
			parsed := &ParsedQuery{
				Terms: []string{"xyz", "abc", "systems"},
			}

			acronyms := qe.generateAcronymExpansions(query, parsed)

			Convey("Then it should return minimal results", func() {
				So(acronyms, ShouldNotBeNil)
			})
		})
	})
}

func TestQueryExpanderHelperMethods(t *testing.T) {
	Convey("Given a QueryExpander", t, func() {
		qe := NewQueryExpander()

		Convey("When building synonym map", func() {
			synonymMap := qe.buildSynonymMap()

			Convey("Then it should contain expected synonyms", func() {
				So(synonymMap, ShouldNotBeNil)
				So(synonymMap["big"], ShouldContain, "large")
				So(synonymMap["fast"], ShouldContain, "quick")
				So(synonymMap["good"], ShouldContain, "excellent")
			})
		})

		Convey("When generating typo variations", func() {
			term := "machine"
			variations := qe.generateTypoVariations(term)

			Convey("Then it should generate character swaps and deletions", func() {
				So(variations, ShouldNotBeNil)
				So(len(variations), ShouldBeGreaterThan, 0)
				So(len(variations), ShouldBeLessThanOrEqualTo, 3) // Limited to 3
			})
		})

		Convey("When generating typo variations for short terms", func() {
			term := "ai"
			variations := qe.generateTypoVariations(term)

			Convey("Then it should return empty results", func() {
				So(len(variations), ShouldEqual, 0)
			})
		})

		Convey("When creating acronym from phrase", func() {
			phrase := "natural language processing"
			acronym := qe.createAcronym(phrase)

			Convey("Then it should create correct acronym", func() {
				So(acronym, ShouldEqual, "nlp")
			})
		})

		Convey("When creating acronym from single word", func() {
			phrase := "machine"
			acronym := qe.createAcronym(phrase)

			Convey("Then it should return empty string", func() {
				So(acronym, ShouldBeEmpty)
			})
		})

		Convey("When deduplicating strings", func() {
			strings := []string{"apple", "banana", "apple", "cherry", "", "  ", "banana"}
			deduplicated := qe.deduplicateStrings(strings)

			Convey("Then it should remove duplicates and empty strings", func() {
				So(len(deduplicated), ShouldEqual, 3)
				So(deduplicated, ShouldContain, "apple")
				So(deduplicated, ShouldContain, "banana")
				So(deduplicated, ShouldContain, "cherry")
			})
		})
	})
}

// Benchmark tests
func BenchmarkQueryExpanderExpand(b *testing.B) {
	qe := NewQueryExpander()
	ctx := context.Background()
	query := "machine learning algorithms for artificial intelligence"
	parsed := &ParsedQuery{
		Terms:     []string{"machine", "learning", "algorithms", "artificial", "intelligence"},
		QueryType: QueryTypeKeyword,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = qe.Expand(ctx, query, parsed)
	}
}

func BenchmarkQueryExpanderGenerateSynonyms(b *testing.B) {
	qe := NewQueryExpander()
	query := "find good fast algorithms"
	parsed := &ParsedQuery{
		Terms: []string{"find", "good", "fast", "algorithms"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qe.GenerateSynonyms(query, parsed)
	}
}

func BenchmarkQueryExpanderAddContext(b *testing.B) {
	qe := NewQueryExpander()
	query := "machine learning algorithms"
	parsed := &ParsedQuery{
		Terms:     []string{"machine", "learning", "algorithms"},
		QueryType: QueryTypeKeyword,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qe.AddContext(query, parsed)
	}
}