package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestClaimExtractor(t *testing.T) {
	Convey("Given a ClaimExtractor", t, func() {
		extractor := NewClaimExtractor()
		
		Convey("When creating a new ClaimExtractor", func() {
			So(extractor, ShouldNotBeNil)
			So(extractor.minConfidence, ShouldEqual, 0.6)
			So(extractor.maxClaimLength, ShouldEqual, 200)
			So(len(extractor.verbPatterns), ShouldBeGreaterThan, 0)
			So(len(extractor.factualIndicators), ShouldBeGreaterThan, 0)
		})
		
		Convey("When extracting claims", func() {
			Convey("From empty text", func() {
				claims, err := extractor.ExtractClaims("", "test_source")
				
				So(err, ShouldBeNil)
				So(claims, ShouldBeEmpty)
			})
			
			Convey("From text with simple factual claims", func() {
				text := "The sky is blue. Water boils at 100 degrees Celsius."
				claims, err := extractor.ExtractClaims(text, "test_source")
				
				So(err, ShouldBeNil)
				So(len(claims), ShouldBeGreaterThan, 0)
				
				// Check for basic subject-verb-object structure
				foundClaim := false
				for _, claim := range claims {
					if claim.Subject != "" && claim.Predicate != "" && claim.Object != "" {
						foundClaim = true
						So(claim.Confidence, ShouldBeGreaterThan, 0.0)
						So(claim.Source, ShouldEqual, "test_source")
					}
				}
				So(foundClaim, ShouldBeTrue)
			})
			
			Convey("From text with definition claims", func() {
				text := "Machine learning is defined as a subset of artificial intelligence. AI means computer systems that can perform tasks requiring human intelligence."
				claims, err := extractor.ExtractClaims(text, "test_source")
				
				So(err, ShouldBeNil)
				So(len(claims), ShouldBeGreaterThan, 0)
				
				// Check for definition-style claims
				foundDefinition := false
				for _, claim := range claims {
					if claim.Predicate == "is_defined_as" || claim.Predicate == "means" {
						foundDefinition = true
						So(claim.Confidence, ShouldBeGreaterThan, 0.6)
					}
				}
				So(foundDefinition, ShouldBeTrue)
			})
			
			Convey("From text with causal claims", func() {
				text := "Smoking causes cancer. Exercise leads to better health. Poor diet results in obesity."
				claims, err := extractor.ExtractClaims(text, "test_source")
				
				So(err, ShouldBeNil)
				So(len(claims), ShouldBeGreaterThan, 0)
				
				// Check for causal claims
				foundCausal := false
				for _, claim := range claims {
					if claim.Predicate == "causes" {
						foundCausal = true
						So(claim.Subject, ShouldNotBeEmpty)
						So(claim.Object, ShouldNotBeEmpty)
					}
				}
				So(foundCausal, ShouldBeTrue)
			})
			
			Convey("From text with temporal claims", func() {
				text := "In 2024, the company launched a new product. The meeting occurred during the morning session."
				claims, err := extractor.ExtractClaims(text, "test_source")
				
				So(err, ShouldBeNil)
				So(len(claims), ShouldBeGreaterThan, 0)
				
				// Check for temporal claims
				foundTemporal := false
				for _, claim := range claims {
					if claim.Predicate == "occurs_during" {
						foundTemporal = true
					}
				}
				So(foundTemporal, ShouldBeTrue)
			})
			
			Convey("From text with factual indicators", func() {
				text := "According to research, exercise improves mental health. Studies indicate that reading enhances cognitive function."
				claims, err := extractor.ExtractClaims(text, "test_source")
				
				So(err, ShouldBeNil)
				So(len(claims), ShouldBeGreaterThan, 0)
				
				// Claims with factual indicators should have higher confidence
				foundHighConfidence := false
				for _, claim := range claims {
					if claim.Confidence > 0.7 {
						foundHighConfidence = true
					}
				}
				So(foundHighConfidence, ShouldBeTrue)
			})
		})
		
		Convey("When validating claims", func() {
			Convey("With valid claim", func() {
				claim := NewClaim("test_id", "The sky", "is", "blue", "test_source")
				claim.Confidence = 0.8
				
				isValid := extractor.ValidateClaim(claim)
				So(isValid, ShouldBeTrue)
			})
			
			Convey("With nil claim", func() {
				isValid := extractor.ValidateClaim(nil)
				So(isValid, ShouldBeFalse)
			})
			
			Convey("With low confidence claim", func() {
				claim := NewClaim("test_id", "The sky", "is", "blue", "test_source")
				claim.Confidence = 0.3
				
				isValid := extractor.ValidateClaim(claim)
				So(isValid, ShouldBeFalse)
			})
			
			Convey("With empty subject", func() {
				claim := NewClaim("test_id", "", "is", "blue", "test_source")
				claim.Confidence = 0.8
				
				isValid := extractor.ValidateClaim(claim)
				So(isValid, ShouldBeFalse)
			})
			
			Convey("With very long claim", func() {
				longObject := ""
				for i := 0; i < 300; i++ {
					longObject += "a"
				}
				claim := NewClaim("test_id", "subject", "is", longObject, "test_source")
				claim.Confidence = 0.8
				
				isValid := extractor.ValidateClaim(claim)
				So(isValid, ShouldBeFalse)
			})
			
			Convey("With trivial claim", func() {
				claim := NewClaim("test_id", "this", "is", "it", "test_source")
				claim.Confidence = 0.8
				
				isValid := extractor.ValidateClaim(claim)
				So(isValid, ShouldBeFalse)
			})
		})
		
		Convey("When scoring claims", func() {
			Convey("With factual indicators in context", func() {
				claim := NewClaim("test_id", "Exercise", "improves", "health", "test_source")
				context := "According to research, exercise improves health significantly."
				
				score := extractor.ScoreClaim(claim, context)
				So(score, ShouldBeGreaterThan, 0.6)
			})
			
			Convey("With opinion indicators in context", func() {
				claim := NewClaim("test_id", "This movie", "is", "good", "test_source")
				context := "I think this movie is good and entertaining."
				
				score := extractor.ScoreClaim(claim, context)
				So(score, ShouldBeLessThan, 0.6)
			})
			
			Convey("With strong verbs", func() {
				claim := NewClaim("test_id", "Water", "is", "wet", "test_source")
				context := "Water is wet by definition."
				
				score := extractor.ScoreClaim(claim, context)
				So(score, ShouldBeGreaterThan, 0.5)
			})
			
			Convey("With proper nouns", func() {
				claim := NewClaim("test_id", "Einstein", "developed", "relativity theory", "test_source")
				context := "Einstein developed the theory of relativity."
				
				score := extractor.ScoreClaim(claim, context)
				So(score, ShouldBeGreaterThan, 0.5)
			})
		})
		
		Convey("When setting configuration", func() {
			Convey("Setting minimum confidence", func() {
				extractor.SetMinConfidence(0.8)
				So(extractor.minConfidence, ShouldEqual, 0.8)
			})
			
			Convey("Setting maximum claim length", func() {
				extractor.SetMaxClaimLength(150)
				So(extractor.maxClaimLength, ShouldEqual, 150)
			})
		})
		
		Convey("When extracting subject-verb-object claims", func() {
			Convey("With clear SVO structure", func() {
				sentence := "The cat sits on the mat."
				claims := extractor.extractSubjectVerbObjectClaims(sentence, "test_source")
				
				So(len(claims), ShouldBeGreaterThan, 0)
				
				foundSVO := false
				for _, claim := range claims {
					if claim.Subject != "" && claim.Predicate != "" && claim.Object != "" {
						foundSVO = true
					}
				}
				So(foundSVO, ShouldBeTrue)
			})
		})
		
		Convey("When extracting definition claims", func() {
			Convey("With 'is defined as' pattern", func() {
				sentence := "Artificial intelligence is defined as machine intelligence."
				claims := extractor.extractDefinitionClaims(sentence, "test_source")
				
				So(len(claims), ShouldBeGreaterThan, 0)
				So(claims[0].Predicate, ShouldEqual, "is_defined_as")
			})
			
			Convey("With 'means' pattern", func() {
				sentence := "AI means artificial intelligence."
				claims := extractor.extractDefinitionClaims(sentence, "test_source")
				
				So(len(claims), ShouldBeGreaterThan, 0)
				So(claims[0].Predicate, ShouldEqual, "is_defined_as")
			})
		})
		
		Convey("When extracting causal claims", func() {
			Convey("With 'causes' pattern", func() {
				sentence := "Smoking causes lung cancer."
				claims := extractor.extractCausalClaims(sentence, "test_source")
				
				So(len(claims), ShouldBeGreaterThan, 0)
				So(claims[0].Predicate, ShouldEqual, "causes")
				So(claims[0].Subject, ShouldContainSubstring, "Smoking")
				So(claims[0].Object, ShouldContainSubstring, "lung cancer")
			})
		})
		
		Convey("When extracting temporal claims", func() {
			Convey("With year pattern", func() {
				sentence := "In 2024, the company went public."
				claims := extractor.extractTemporalClaims(sentence, "test_source")
				
				So(len(claims), ShouldBeGreaterThan, 0)
				So(claims[0].Predicate, ShouldEqual, "occurs_during")
			})
		})
		
		Convey("When filtering valid claims", func() {
			Convey("With mixed valid and invalid claims", func() {
				claims := []*Claim{
					NewClaim("1", "The sky", "is", "blue", "test_source"),
					NewClaim("2", "", "is", "empty", "test_source"), // Invalid: empty subject
					NewClaim("3", "Water", "boils", "at 100C", "test_source"),
					NewClaim("4", "this", "is", "it", "test_source"), // Invalid: trivial
				}
				
				// Set valid confidences
				for _, claim := range claims {
					claim.Confidence = 0.8
				}
				
				filtered := extractor.filterValidClaims(claims)
				
				So(len(filtered), ShouldEqual, 2) // Only valid claims
			})
		})
	})
}

func BenchmarkClaimExtractor(b *testing.B) {
	extractor := NewClaimExtractor()
	
	// Create text with various claim types
	text := `
		The sky is blue and water is wet. According to research, exercise improves health.
		Machine learning is defined as a subset of artificial intelligence.
		Smoking causes cancer and leads to various health problems.
		In 2024, the company launched a revolutionary product.
		Studies indicate that reading enhances cognitive abilities significantly.
	`
	
	b.Run("ExtractClaims", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			extractor.ExtractClaims(text, "benchmark_source")
		}
	})
	
	b.Run("ExtractSubjectVerbObject", func(b *testing.B) {
		sentence := "The researchers discovered a new method for data processing."
		for i := 0; i < b.N; i++ {
			extractor.extractSubjectVerbObjectClaims(sentence, "benchmark_source")
		}
	})
	
	b.Run("ScoreClaim", func(b *testing.B) {
		claim := NewClaim("test", "Exercise", "improves", "health", "test_source")
		context := "According to multiple studies, exercise improves overall health."
		
		for i := 0; i < b.N; i++ {
			extractor.ScoreClaim(claim, context)
		}
	})
	
	// Benchmark with large text
	largeText := ""
	for i := 0; i < 100; i++ {
		largeText += text
	}
	
	b.Run("ExtractClaimsLargeText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			extractor.ExtractClaims(largeText, "benchmark_source")
		}
	})
}