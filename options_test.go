package main

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRecallOptions(t *testing.T) {
	Convey("Given RecallOptions", t, func() {
		Convey("When creating new RecallOptions", func() {
			options := NewRecallOptions()

			Convey("Then it should have correct default values", func() {
				So(options.MaxResults, ShouldEqual, 10)
				So(options.TimeBudget, ShouldEqual, 5*time.Second)
				So(options.IncludeGraph, ShouldBeFalse)
				So(options.Filters, ShouldNotBeNil)
				So(options.MinConfidence, ShouldEqual, 0.0)
				So(options.SortBy, ShouldEqual, "relevance")
				So(options.SortOrder, ShouldEqual, "desc")
				So(options.ExpandQuery, ShouldBeTrue)
				So(options.UseCache, ShouldBeTrue)
			})
		})

		Convey("When validating valid options", func() {
			options := NewRecallOptions()
			err := options.Validate()

			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating invalid options", func() {
			Convey("With negative max results", func() {
				options := NewRecallOptions()
				options.MaxResults = -1
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "max results must be positive")
				})
			})

			Convey("With too many max results", func() {
				options := NewRecallOptions()
				options.MaxResults = 2000
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "max results cannot exceed 1000")
				})
			})

			Convey("With negative time budget", func() {
				options := NewRecallOptions()
				options.TimeBudget = -1 * time.Second
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "time budget must be positive")
				})
			})

			Convey("With invalid confidence", func() {
				options := NewRecallOptions()
				options.MinConfidence = 1.5
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "min confidence must be between 0 and 1")
				})
			})

			Convey("With invalid sort by", func() {
				options := NewRecallOptions()
				options.SortBy = "invalid"
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "invalid sort by value")
				})
			})

			Convey("With invalid sort order", func() {
				options := NewRecallOptions()
				options.SortOrder = "invalid"
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "invalid sort order value")
				})
			})
		})

		Convey("When setting and getting filters", func() {
			options := NewRecallOptions()
			options.SetFilter("source", "test")

			Convey("Then filter should be set correctly", func() {
				value, exists := options.GetFilter("source")
				So(exists, ShouldBeTrue)
				So(value, ShouldEqual, "test")
			})

			Convey("And getting non-existent filter should return false", func() {
				_, exists := options.GetFilter("nonexistent")
				So(exists, ShouldBeFalse)
			})
		})
	})
}

func TestWriteMetadata(t *testing.T) {
	Convey("Given WriteMetadata", t, func() {
		Convey("When creating new WriteMetadata", func() {
			metadata := NewWriteMetadata("test_source")

			Convey("Then it should have correct default values", func() {
				So(metadata.Source, ShouldEqual, "test_source")
				So(metadata.Timestamp, ShouldNotBeZeroValue)
				So(metadata.Tags, ShouldNotBeNil)
				So(metadata.Confidence, ShouldEqual, 1.0)
				So(metadata.RequireEvidence, ShouldBeFalse)
				So(metadata.Language, ShouldEqual, "en")
				So(metadata.ContentType, ShouldEqual, "text/plain")
				So(metadata.Version, ShouldEqual, "1.0")
				So(metadata.Metadata, ShouldNotBeNil)
			})
		})

		Convey("When validating valid metadata", func() {
			metadata := NewWriteMetadata("test_source")
			err := metadata.Validate()

			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating invalid metadata", func() {
			Convey("With empty source", func() {
				metadata := NewWriteMetadata("")
				err := metadata.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "source cannot be empty")
				})
			})

			Convey("With zero timestamp", func() {
				metadata := NewWriteMetadata("test")
				metadata.Timestamp = time.Time{}
				err := metadata.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "timestamp cannot be zero")
				})
			})

			Convey("With invalid confidence", func() {
				metadata := NewWriteMetadata("test")
				metadata.Confidence = -0.5
				err := metadata.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "confidence must be between 0 and 1")
				})
			})

			Convey("With invalid language", func() {
				metadata := NewWriteMetadata("test")
				metadata.Language = "english"
				err := metadata.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "language must be a 2-character code")
				})
			})
		})

		Convey("When managing tags", func() {
			metadata := NewWriteMetadata("test")

			Convey("Adding a tag", func() {
				metadata.AddTag("important")

				Convey("Then tag should be added", func() {
					So(metadata.HasTag("important"), ShouldBeTrue)
					So(len(metadata.Tags), ShouldEqual, 1)
				})

				Convey("And adding duplicate tag should not increase count", func() {
					metadata.AddTag("important")
					So(len(metadata.Tags), ShouldEqual, 1)
				})
			})

			Convey("Removing a tag", func() {
				metadata.AddTag("temp")
				metadata.RemoveTag("temp")

				Convey("Then tag should be removed", func() {
					So(metadata.HasTag("temp"), ShouldBeFalse)
					So(len(metadata.Tags), ShouldEqual, 0)
				})
			})
		})

		Convey("When managing metadata", func() {
			metadata := NewWriteMetadata("test")

			Convey("Setting metadata", func() {
				metadata.SetMetadata("author", "test_author")

				Convey("Then metadata should be set", func() {
					value, exists := metadata.GetMetadata("author")
					So(exists, ShouldBeTrue)
					So(value, ShouldEqual, "test_author")
				})
			})

			Convey("Getting non-existent metadata", func() {
				_, exists := metadata.GetMetadata("nonexistent")

				Convey("Then should return false", func() {
					So(exists, ShouldBeFalse)
				})
			})
		})
	})
}

func TestManageOptions(t *testing.T) {
	Convey("Given ManageOptions", t, func() {
		Convey("When creating new ManageOptions", func() {
			options := NewManageOptions("pin")

			Convey("Then it should have correct default values", func() {
				So(options.Operation, ShouldEqual, "pin")
				So(options.MemoryIDs, ShouldNotBeNil)
				So(options.Confidence, ShouldEqual, 0.0)
				So(options.TTL, ShouldEqual, 24*time.Hour)
				So(options.Force, ShouldBeFalse)
				So(options.DryRun, ShouldBeFalse)
				So(options.BatchSize, ShouldEqual, 100)
			})
		})

		Convey("When validating valid options", func() {
			options := NewManageOptions("forget")
			err := options.Validate()

			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating invalid options", func() {
			Convey("With empty operation", func() {
				options := NewManageOptions("")
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "operation cannot be empty")
				})
			})

			Convey("With invalid operation", func() {
				options := NewManageOptions("invalid")
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "invalid operation")
				})
			})

			Convey("With invalid confidence", func() {
				options := NewManageOptions("pin")
				options.Confidence = 2.0
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "confidence must be between 0 and 1")
				})
			})

			Convey("With negative TTL", func() {
				options := NewManageOptions("pin")
				options.TTL = -1 * time.Hour
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "TTL cannot be negative")
				})
			})

			Convey("With invalid batch size", func() {
				options := NewManageOptions("pin")
				options.BatchSize = 0
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "batch size must be positive")
				})
			})
		})
	})
}

func TestSearchOptions(t *testing.T) {
	Convey("Given SearchOptions", t, func() {
		Convey("When creating new SearchOptions", func() {
			options := NewSearchOptions("test query")

			Convey("Then it should have correct default values", func() {
				So(options.Query, ShouldEqual, "test query")
				So(options.MaxResults, ShouldEqual, 10)
				So(options.Offset, ShouldEqual, 0)
				So(options.Filters, ShouldNotBeNil)
				So(options.MinConfidence, ShouldEqual, 0.0)
				So(options.SearchType, ShouldEqual, "hybrid")
				So(options.IncludeScore, ShouldBeTrue)
				So(options.Highlight, ShouldBeFalse)
			})
		})

		Convey("When validating valid options", func() {
			options := NewSearchOptions("test")
			err := options.Validate()

			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating invalid options", func() {
			Convey("With empty query", func() {
				options := NewSearchOptions("")
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "query cannot be empty")
				})
			})

			Convey("With invalid max results", func() {
				options := NewSearchOptions("test")
				options.MaxResults = 0
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "max results must be positive")
				})
			})

			Convey("With too many max results", func() {
				options := NewSearchOptions("test")
				options.MaxResults = 2000
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "max results cannot exceed 1000")
				})
			})

			Convey("With negative offset", func() {
				options := NewSearchOptions("test")
				options.Offset = -1
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "offset cannot be negative")
				})
			})

			Convey("With invalid search type", func() {
				options := NewSearchOptions("test")
				options.SearchType = "invalid"
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "invalid search type")
				})
			})
		})
	})
}

func TestGraphOptions(t *testing.T) {
	Convey("Given GraphOptions", t, func() {
		Convey("When creating new GraphOptions", func() {
			options := NewGraphOptions()

			Convey("Then it should have correct default values", func() {
				So(options.MaxDepth, ShouldEqual, 3)
				So(options.EdgeTypes, ShouldNotBeNil)
				So(options.NodeTypes, ShouldNotBeNil)
				So(options.MinWeight, ShouldEqual, 0.0)
				So(options.MaxNodes, ShouldEqual, 100)
				So(options.IncludeProps, ShouldBeTrue)
			})
		})

		Convey("When validating valid options", func() {
			options := NewGraphOptions()
			err := options.Validate()

			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating invalid options", func() {
			Convey("With invalid max depth", func() {
				options := NewGraphOptions()
				options.MaxDepth = 0
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "max depth must be positive")
				})
			})

			Convey("With too large max depth", func() {
				options := NewGraphOptions()
				options.MaxDepth = 20
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "max depth cannot exceed 10")
				})
			})

			Convey("With negative min weight", func() {
				options := NewGraphOptions()
				options.MinWeight = -1.0
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "min weight cannot be negative")
				})
			})

			Convey("With invalid max nodes", func() {
				options := NewGraphOptions()
				options.MaxNodes = 0
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "max nodes must be positive")
				})
			})

			Convey("With too many max nodes", func() {
				options := NewGraphOptions()
				options.MaxNodes = 20000
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "max nodes cannot exceed 10000")
				})
			})

			Convey("With invalid algorithm", func() {
				options := NewGraphOptions()
				options.Algorithm = "invalid"
				err := options.Validate()

				Convey("Then validation should fail", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "invalid algorithm")
				})
			})
		})
	})
}

// Benchmark tests
func BenchmarkNewRecallOptions(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		options := NewRecallOptions()
		_ = options
	}
}

func BenchmarkRecallOptionsValidation(b *testing.B) {
	options := NewRecallOptions()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := options.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteMetadataValidation(b *testing.B) {
	metadata := NewWriteMetadata("test_source")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := metadata.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkManageOptionsValidation(b *testing.B) {
	options := NewManageOptions("pin")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := options.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}