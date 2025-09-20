package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEntityExtractor(t *testing.T) {
	Convey("Given an EntityExtractor", t, func() {
		extractor := NewEntityExtractor()
		
		Convey("When creating a new EntityExtractor", func() {
			So(extractor, ShouldNotBeNil)
			So(extractor.minConfidence, ShouldEqual, 0.5)
			So(len(extractor.patterns), ShouldBeGreaterThan, 0)
			So(len(extractor.keywords), ShouldBeGreaterThan, 0)
		})
		
		Convey("When extracting entities", func() {
			Convey("From empty text", func() {
				entities, err := extractor.Extract("", "test_source")
				
				So(err, ShouldBeNil)
				So(entities, ShouldBeEmpty)
			})
			
			Convey("From text with email addresses", func() {
				text := "Contact John Doe at john.doe@example.com for more information."
				entities, err := extractor.Extract(text, "test_source")
				
				So(err, ShouldBeNil)
				So(len(entities), ShouldBeGreaterThan, 0)
				
				// Check for email entity
				foundEmail := false
				for _, entity := range entities {
					if entity.Type == string(EmailEntity) && entity.Name == "john.doe@example.com" {
						foundEmail = true
						So(entity.Confidence, ShouldBeGreaterThan, 0.9)
					}
				}
				So(foundEmail, ShouldBeTrue)
			})
			
			Convey("From text with URLs", func() {
				text := "Visit our website at https://www.example.com for more details."
				entities, err := extractor.Extract(text, "test_source")
				
				So(err, ShouldBeNil)
				So(len(entities), ShouldBeGreaterThan, 0)
				
				// Check for URL entity
				foundURL := false
				for _, entity := range entities {
					if entity.Type == string(URLEntity) {
						foundURL = true
						So(entity.Confidence, ShouldBeGreaterThan, 0.9)
					}
				}
				So(foundURL, ShouldBeTrue)
			})
			
			Convey("From text with phone numbers", func() {
				text := "Call us at (555) 123-4567 or +1-555-987-6543."
				entities, err := extractor.Extract(text, "test_source")
				
				So(err, ShouldBeNil)
				So(len(entities), ShouldBeGreaterThan, 0)
				
				// Check for phone entity
				foundPhone := false
				for _, entity := range entities {
					if entity.Type == string(PhoneEntity) {
						foundPhone = true
						So(entity.Confidence, ShouldBeGreaterThan, 0.8)
					}
				}
				So(foundPhone, ShouldBeTrue)
			})
			
			Convey("From text with dates", func() {
				text := "The meeting is scheduled for January 15, 2024 and the deadline is 2024-01-30."
				entities, err := extractor.Extract(text, "test_source")
				
				So(err, ShouldBeNil)
				So(len(entities), ShouldBeGreaterThan, 0)
				
				// Check for date entities
				foundDate := false
				for _, entity := range entities {
					if entity.Type == string(DateEntity) {
						foundDate = true
						So(entity.Confidence, ShouldBeGreaterThan, 0.8)
					}
				}
				So(foundDate, ShouldBeTrue)
			})
			
			Convey("From text with numbers", func() {
				text := "The price is $1,234.56 and the discount is 15%."
				entities, err := extractor.Extract(text, "test_source")
				
				So(err, ShouldBeNil)
				So(len(entities), ShouldBeGreaterThan, 0)
				
				// Check for number entities
				foundNumber := false
				for _, entity := range entities {
					if entity.Type == string(NumberEntity) {
						foundNumber = true
						So(entity.Confidence, ShouldBeGreaterThan, 0.8)
					}
				}
				So(foundNumber, ShouldBeTrue)
			})
			
			Convey("From text with person indicators", func() {
				text := "Dr. Smith and Mr. Johnson are working on the project."
				entities, err := extractor.Extract(text, "test_source")
				
				So(err, ShouldBeNil)
				So(len(entities), ShouldBeGreaterThan, 0)
				
				// Check for person entities
				foundPerson := false
				for _, entity := range entities {
					if entity.Type == string(PersonEntity) {
						foundPerson = true
						So(entity.Confidence, ShouldBeGreaterThan, 0.5)
					}
				}
				So(foundPerson, ShouldBeTrue)
			})
		})
		
		Convey("When validating entities", func() {
			Convey("With valid entity", func() {
				entity := NewEntity("test_id", "John Doe", string(PersonEntity), "test_source")
				entity.Confidence = 0.8
				
				isValid := extractor.ValidateEntity(entity)
				So(isValid, ShouldBeTrue)
			})
			
			Convey("With nil entity", func() {
				isValid := extractor.ValidateEntity(nil)
				So(isValid, ShouldBeFalse)
			})
			
			Convey("With low confidence entity", func() {
				entity := NewEntity("test_id", "John Doe", string(PersonEntity), "test_source")
				entity.Confidence = 0.3
				
				isValid := extractor.ValidateEntity(entity)
				So(isValid, ShouldBeFalse)
			})
			
			Convey("With empty name", func() {
				entity := NewEntity("test_id", "", string(PersonEntity), "test_source")
				entity.Confidence = 0.8
				
				isValid := extractor.ValidateEntity(entity)
				So(isValid, ShouldBeFalse)
			})
			
			Convey("With very short name", func() {
				entity := NewEntity("test_id", "A", string(PersonEntity), "test_source")
				entity.Confidence = 0.8
				
				isValid := extractor.ValidateEntity(entity)
				So(isValid, ShouldBeFalse)
			})
			
			Convey("With very long name", func() {
				longName := ""
				for i := 0; i < 150; i++ {
					longName += "a"
				}
				entity := NewEntity("test_id", longName, string(PersonEntity), "test_source")
				entity.Confidence = 0.8
				
				isValid := extractor.ValidateEntity(entity)
				So(isValid, ShouldBeFalse)
			})
			
			Convey("With false positive name", func() {
				entity := NewEntity("test_id", "the", string(PersonEntity), "test_source")
				entity.Confidence = 0.8
				
				isValid := extractor.ValidateEntity(entity)
				So(isValid, ShouldBeFalse)
			})
		})
		
		Convey("When filtering entities", func() {
			Convey("With mixed valid and invalid entities", func() {
				entities := []*Entity{
					NewEntity("1", "John Doe", string(PersonEntity), "test_source"),
					NewEntity("2", "", string(PersonEntity), "test_source"), // Invalid: empty name
					NewEntity("3", "jane@example.com", string(EmailEntity), "test_source"),
					NewEntity("4", "the", string(PersonEntity), "test_source"), // Invalid: false positive
				}
				
				// Set valid confidences
				entities[0].Confidence = 0.8
				entities[1].Confidence = 0.8
				entities[2].Confidence = 0.9
				entities[3].Confidence = 0.8
				
				filtered := extractor.FilterEntities(entities)
				
				So(len(filtered), ShouldEqual, 2) // Only valid entities
				So(filtered[0].Name, ShouldEqual, "John Doe")
				So(filtered[1].Name, ShouldEqual, "jane@example.com")
			})
			
			Convey("With duplicate entities", func() {
				entities := []*Entity{
					NewEntity("1", "John Doe", string(PersonEntity), "test_source"),
					NewEntity("2", "john doe", string(PersonEntity), "test_source"), // Duplicate (case insensitive)
					NewEntity("3", "Jane Smith", string(PersonEntity), "test_source"),
				}
				
				// Set valid confidences
				for _, entity := range entities {
					entity.Confidence = 0.8
				}
				
				filtered := extractor.FilterEntities(entities)
				
				So(len(filtered), ShouldEqual, 2) // Duplicates removed
			})
		})
		
		Convey("When setting configuration", func() {
			Convey("Setting minimum confidence", func() {
				extractor.SetMinConfidence(0.7)
				So(extractor.minConfidence, ShouldEqual, 0.7)
			})
			
			Convey("Adding custom pattern", func() {
				err := extractor.AddPattern("CUSTOM", `\b[A-Z]{3}\d{3}\b`)
				So(err, ShouldBeNil)
				So(extractor.patterns["CUSTOM"], ShouldNotBeNil)
			})
			
			Convey("Adding invalid pattern", func() {
				err := extractor.AddPattern("INVALID", `[unclosed`)
				So(err, ShouldNotBeNil)
			})
			
			Convey("Adding custom keywords", func() {
				extractor.AddKeywords("CUSTOM_TYPE", []string{"keyword1", "keyword2"})
				So(len(extractor.keywords["CUSTOM_TYPE"]), ShouldEqual, 2)
			})
		})
		
		Convey("When extracting by patterns", func() {
			Convey("With matching patterns", func() {
				text := "Email: test@example.com and phone: 555-1234"
				entities := extractor.extractByPatterns(text, "test_source")
				
				So(len(entities), ShouldBeGreaterThan, 0)
				
				// Should find email and phone
				foundTypes := make(map[string]bool)
				for _, entity := range entities {
					foundTypes[entity.Type] = true
				}
				So(foundTypes[string(EmailEntity)], ShouldBeTrue)
			})
		})
		
		Convey("When extracting by keywords", func() {
			Convey("With person keywords", func() {
				text := "Dr. Johnson is the lead researcher."
				entities := extractor.extractByKeywords(text, "test_source")
				
				So(len(entities), ShouldBeGreaterThan, 0)
				
				// Should find person entity
				foundPerson := false
				for _, entity := range entities {
					if entity.Type == string(PersonEntity) {
						foundPerson = true
					}
				}
				So(foundPerson, ShouldBeTrue)
			})
		})
	})
}

func BenchmarkEntityExtractor(b *testing.B) {
	extractor := NewEntityExtractor()
	
	// Create text with various entity types
	text := `
		Contact Dr. John Smith at john.smith@example.com or call (555) 123-4567.
		Visit our website at https://www.example.com for more information.
		The meeting is scheduled for January 15, 2024 at 2:30 PM.
		The project budget is $1,234,567.89 with a 15% contingency.
		ABC Corporation and XYZ University are collaborating on this research.
	`
	
	b.Run("Extract", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			extractor.Extract(text, "benchmark_source")
		}
	})
	
	b.Run("ExtractByPatterns", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			extractor.extractByPatterns(text, "benchmark_source")
		}
	})
	
	b.Run("ExtractByKeywords", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			extractor.extractByKeywords(text, "benchmark_source")
		}
	})
	
	// Benchmark with large text
	largeText := ""
	for i := 0; i < 100; i++ {
		largeText += text
	}
	
	b.Run("ExtractLargeText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			extractor.Extract(largeText, "benchmark_source")
		}
	})
}