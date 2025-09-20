package main

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestChunk(t *testing.T) {
	Convey("Given a new chunk", t, func() {
		id := "test-chunk-1"
		content := "This is test content"
		source := "test-source"
		
		Convey("When creating a new chunk", func() {
			chunk := NewChunk(id, content, source)
			
			Convey("Then it should have correct initial values", func() {
				So(chunk.ID, ShouldEqual, id)
				So(chunk.Content, ShouldEqual, content)
				So(chunk.Source, ShouldEqual, source)
				So(chunk.Confidence, ShouldEqual, 1.0)
				So(chunk.Metadata, ShouldNotBeNil)
				So(chunk.Claims, ShouldNotBeNil)
				So(chunk.Entities, ShouldNotBeNil)
				So(len(chunk.Claims), ShouldEqual, 0)
				So(len(chunk.Entities), ShouldEqual, 0)
				So(chunk.Timestamp, ShouldHappenWithin, time.Second, time.Now())
			})
		})
		
		Convey("When validating a valid chunk", func() {
			chunk := NewChunk(id, content, source)
			err := chunk.Validate()
			
			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})
		
		Convey("When validating an invalid chunk", func() {
			Convey("With empty ID", func() {
				chunk := NewChunk("", content, source)
				err := chunk.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "chunk ID cannot be empty")
			})
			
			Convey("With empty content", func() {
				chunk := NewChunk(id, "", source)
				err := chunk.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "chunk content cannot be empty")
			})
			
			Convey("With empty source", func() {
				chunk := NewChunk(id, content, "")
				err := chunk.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "chunk source cannot be empty")
			})
			
			Convey("With invalid confidence", func() {
				chunk := NewChunk(id, content, source)
				chunk.Confidence = -0.5
				err := chunk.Validate()
				
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "chunk confidence must be between 0 and 1")
			})
		})
		
		Convey("When marshaling to JSON", func() {
			chunk := NewChunk(id, content, source)
			chunk.Timestamp = time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
			
			data, err := chunk.MarshalJSON()
			
			Convey("Then it should marshal successfully", func() {
				So(err, ShouldBeNil)
				So(data, ShouldNotBeNil)
				
				var result map[string]interface{}
				err = json.Unmarshal(data, &result)
				So(err, ShouldBeNil)
				So(result["timestamp"], ShouldEqual, "2023-01-01T12:00:00Z")
			})
		})
		
		Convey("When unmarshaling from JSON", func() {
			jsonData := `{
				"id": "test-chunk-1",
				"content": "This is test content",
				"source": "test-source",
				"confidence": 0.8,
				"timestamp": "2023-01-01T12:00:00Z",
				"metadata": {},
				"claims": [],
				"entities": []
			}`
			
			var chunk Chunk
			err := chunk.UnmarshalJSON([]byte(jsonData))
			
			Convey("Then it should unmarshal successfully", func() {
				So(err, ShouldBeNil)
				So(chunk.ID, ShouldEqual, "test-chunk-1")
				So(chunk.Content, ShouldEqual, "This is test content")
				So(chunk.Source, ShouldEqual, "test-source")
				So(chunk.Confidence, ShouldEqual, 0.8)
				So(chunk.Timestamp.Year(), ShouldEqual, 2023)
			})
		})
		
		Convey("When adding claims and entities", func() {
			chunk := NewChunk(id, content, source)
			claim := NewClaim("claim-1", "subject", "predicate", "object", source)
			entity := NewEntity("entity-1", "Test Entity", "PERSON", source)
			
			chunk.AddClaim(*claim)
			chunk.AddEntity(*entity)
			
			Convey("Then they should be added correctly", func() {
				So(len(chunk.Claims), ShouldEqual, 1)
				So(len(chunk.Entities), ShouldEqual, 1)
				So(chunk.Claims[0].ID, ShouldEqual, "claim-1")
				So(chunk.Entities[0].ID, ShouldEqual, "entity-1")
			})
		})
		
		Convey("When setting and getting metadata", func() {
			chunk := NewChunk(id, content, source)
			key := "test-key"
			value := "test-value"
			
			chunk.SetMetadata(key, value)
			retrievedValue, exists := chunk.GetMetadata(key)
			
			Convey("Then metadata should be stored and retrieved correctly", func() {
				So(exists, ShouldBeTrue)
				So(retrievedValue, ShouldEqual, value)
			})
			
			Convey("And getting non-existent metadata should return false", func() {
				_, exists := chunk.GetMetadata("non-existent")
				So(exists, ShouldBeFalse)
			})
		})
		
		Convey("When setting embedding", func() {
			chunk := NewChunk(id, content, source)
			embedding := []float32{0.1, 0.2, 0.3, 0.4}
			
			chunk.SetEmbedding(embedding)
			
			Convey("Then embedding should be set correctly", func() {
				So(chunk.Embedding, ShouldResemble, embedding)
			})
		})
	})
}

func BenchmarkChunkCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewChunk("test-id", "test content", "test source")
	}
}

func BenchmarkChunkValidation(b *testing.B) {
	chunk := NewChunk("test-id", "test content", "test source")
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		chunk.Validate()
	}
}

func BenchmarkChunkJSONMarshal(b *testing.B) {
	chunk := NewChunk("test-id", "test content", "test source")
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		chunk.MarshalJSON()
	}
}

func BenchmarkChunkJSONUnmarshal(b *testing.B) {
	jsonData := []byte(`{
		"id": "test-chunk-1",
		"content": "This is test content",
		"source": "test-source",
		"confidence": 0.8,
		"timestamp": "2023-01-01T12:00:00Z",
		"metadata": {},
		"claims": [],
		"entities": []
	}`)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var chunk Chunk
		chunk.UnmarshalJSON(jsonData)
	}
}