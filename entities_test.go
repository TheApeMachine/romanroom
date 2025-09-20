package main

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEntity(t *testing.T) {
	Convey("Given a new entity", t, func() {
		id := "test-entity-1"
		name := "Test Entity"
		entityType := "PERSON"
		source := "test-source"
		
		Convey("When creating a new entity", func() {
			entity := NewEntity(id, name, entityType, source)
			
			Convey("Then it should have correct initial values", func() {
				So(entity.ID, ShouldEqual, id)
				So(entity.Name, ShouldEqual, name)
				So(entity.Type, ShouldEqual, entityType)
				So(entity.Source, ShouldEqual, source)
				So(entity.Confidence, ShouldEqual, 1.0)
				So(entity.Properties, ShouldNotBeNil)
				So(len(entity.Properties), ShouldEqual, 0)
				So(entity.CreatedAt, ShouldHappenWithin, time.Second, time.Now())
			})
		})
		
		Convey("When validating a valid entity", func() {
			entity := NewEntity(id, name, entityType, source)
			err := entity.Validate()
			
			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})
		
		Convey("When validating an invalid entity", func() {
			Convey("With empty ID", func() {
				entity := NewEntity("", name, entityType, source)
				err := entity.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "entity ID cannot be empty")
			})
			
			Convey("With empty name", func() {
				entity := NewEntity(id, "", entityType, source)
				err := entity.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "entity name cannot be empty")
			})
			
			Convey("With empty type", func() {
				entity := NewEntity(id, name, "", source)
				err := entity.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "entity type cannot be empty")
			})
			
			Convey("With empty source", func() {
				entity := NewEntity(id, name, entityType, "")
				err := entity.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "entity source cannot be empty")
			})
			
			Convey("With invalid confidence", func() {
				entity := NewEntity(id, name, entityType, source)
				entity.Confidence = 1.5
				err := entity.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "entity confidence must be between 0 and 1")
			})
		})
		
		Convey("When setting and getting properties", func() {
			entity := NewEntity(id, name, entityType, source)
			key := "test-property"
			value := "test-value"
			
			entity.SetProperty(key, value)
			retrievedValue, exists := entity.GetProperty(key)
			
			Convey("Then property should be stored and retrieved correctly", func() {
				So(exists, ShouldBeTrue)
				So(retrievedValue, ShouldEqual, value)
			})
			
			Convey("And getting non-existent property should return false", func() {
				_, exists := entity.GetProperty("non-existent")
				So(exists, ShouldBeFalse)
			})
		})
		
		Convey("When getting string representation", func() {
			entity := NewEntity(id, name, entityType, source)
			entity.Confidence = 0.85
			
			str := entity.String()
			
			Convey("Then it should contain all key information", func() {
				So(str, ShouldContainSubstring, id)
				So(str, ShouldContainSubstring, name)
				So(str, ShouldContainSubstring, entityType)
				So(str, ShouldContainSubstring, "0.85")
			})
		})
		
		Convey("When getting normalized name", func() {
			entity := NewEntity(id, "  Test Entity  ", entityType, source)
			normalized := entity.NormalizedName()
			
			Convey("Then it should be lowercase and trimmed", func() {
				So(normalized, ShouldEqual, "test entity")
			})
		})
	})
}

func TestClaim(t *testing.T) {
	Convey("Given a new claim", t, func() {
		id := "test-claim-1"
		subject := "John Doe"
		predicate := "works at"
		object := "Acme Corp"
		source := "test-source"
		
		Convey("When creating a new claim", func() {
			claim := NewClaim(id, subject, predicate, object, source)
			
			Convey("Then it should have correct initial values", func() {
				So(claim.ID, ShouldEqual, id)
				So(claim.Subject, ShouldEqual, subject)
				So(claim.Predicate, ShouldEqual, predicate)
				So(claim.Object, ShouldEqual, object)
				So(claim.Source, ShouldEqual, source)
				So(claim.Confidence, ShouldEqual, 1.0)
				So(claim.Evidence, ShouldNotBeNil)
				So(len(claim.Evidence), ShouldEqual, 0)
				So(claim.CreatedAt, ShouldHappenWithin, time.Second, time.Now())
			})
		})
		
		Convey("When validating a valid claim", func() {
			claim := NewClaim(id, subject, predicate, object, source)
			err := claim.Validate()
			
			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})
		
		Convey("When validating an invalid claim", func() {
			Convey("With empty ID", func() {
				claim := NewClaim("", subject, predicate, object, source)
				err := claim.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "claim ID cannot be empty")
			})
			
			Convey("With empty subject", func() {
				claim := NewClaim(id, "", predicate, object, source)
				err := claim.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "claim subject cannot be empty")
			})
			
			Convey("With empty predicate", func() {
				claim := NewClaim(id, subject, "", object, source)
				err := claim.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "claim predicate cannot be empty")
			})
			
			Convey("With empty object", func() {
				claim := NewClaim(id, subject, predicate, "", source)
				err := claim.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "claim object cannot be empty")
			})
			
			Convey("With empty source", func() {
				claim := NewClaim(id, subject, predicate, object, "")
				err := claim.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "claim source cannot be empty")
			})
			
			Convey("With invalid confidence", func() {
				claim := NewClaim(id, subject, predicate, object, source)
				claim.Confidence = -0.1
				err := claim.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "claim confidence must be between 0 and 1")
			})
		})
		
		Convey("When adding evidence", func() {
			claim := NewClaim(id, subject, predicate, object, source)
			evidence1 := "Evidence 1"
			evidence2 := "Evidence 2"
			
			claim.AddEvidence(evidence1)
			claim.AddEvidence(evidence2)
			
			Convey("Then evidence should be added correctly", func() {
				So(len(claim.Evidence), ShouldEqual, 2)
				So(claim.Evidence[0], ShouldEqual, evidence1)
				So(claim.Evidence[1], ShouldEqual, evidence2)
				So(claim.HasEvidence(), ShouldBeTrue)
				So(claim.EvidenceCount(), ShouldEqual, 2)
			})
		})
		
		Convey("When getting string representation", func() {
			claim := NewClaim(id, subject, predicate, object, source)
			claim.Confidence = 0.75
			
			str := claim.String()
			
			Convey("Then it should contain all key information", func() {
				So(str, ShouldContainSubstring, id)
				So(str, ShouldContainSubstring, subject)
				So(str, ShouldContainSubstring, predicate)
				So(str, ShouldContainSubstring, object)
				So(str, ShouldContainSubstring, "0.75")
			})
		})
		
		Convey("When getting triple representation", func() {
			claim := NewClaim(id, subject, predicate, object, source)
			triple := claim.Triple()
			
			Convey("Then it should be formatted as subject-predicate-object", func() {
				expected := subject + " " + predicate + " " + object
				So(triple, ShouldEqual, expected)
			})
		})
		
		Convey("When checking evidence status", func() {
			claim := NewClaim(id, subject, predicate, object, source)
			
			Convey("Initially should have no evidence", func() {
				So(claim.HasEvidence(), ShouldBeFalse)
				So(claim.EvidenceCount(), ShouldEqual, 0)
			})
			
			Convey("After adding evidence should have evidence", func() {
				claim.AddEvidence("test evidence")
				So(claim.HasEvidence(), ShouldBeTrue)
				So(claim.EvidenceCount(), ShouldEqual, 1)
			})
		})
	})
}

func BenchmarkEntityCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewEntity("test-id", "Test Entity", "PERSON", "test-source")
	}
}

func BenchmarkEntityValidation(b *testing.B) {
	entity := NewEntity("test-id", "Test Entity", "PERSON", "test-source")
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		entity.Validate()
	}
}

func BenchmarkClaimCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewClaim("test-id", "subject", "predicate", "object", "test-source")
	}
}

func BenchmarkClaimValidation(b *testing.B) {
	claim := NewClaim("test-id", "subject", "predicate", "object", "test-source")
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		claim.Validate()
	}
}

func BenchmarkEntityPropertyOperations(b *testing.B) {
	entity := NewEntity("test-id", "Test Entity", "PERSON", "test-source")
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		entity.SetProperty("key", "value")
		entity.GetProperty("key")
	}
}

func BenchmarkClaimEvidenceOperations(b *testing.B) {
	claim := NewClaim("test-id", "subject", "predicate", "object", "test-source")
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		claim.AddEvidence("test evidence")
		claim.HasEvidence()
		claim.EvidenceCount()
	}
}