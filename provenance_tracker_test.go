package main

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestProvenanceTracker(t *testing.T) {
	Convey("Given a ProvenanceTracker", t, func() {
		tracker := NewProvenanceTracker()
		
		Convey("When creating a new ProvenanceTracker", func() {
			So(tracker, ShouldNotBeNil)
			So(tracker.records, ShouldNotBeNil)
			So(tracker.config, ShouldNotBeNil)
			So(tracker.config.EnableVersioning, ShouldBeTrue)
			So(tracker.config.EnableIntegrityCheck, ShouldBeTrue)
		})

		Convey("When tracking a new memory", func() {
			memoryID := "memory_123"
			metadata := WriteMetadata{
				Source:     "test_source",
				Timestamp:  time.Now(),
				UserID:     "test_user",
				Tags:       []string{"test", "memory"},
				Confidence: 0.8,
			}

			provenanceID, err := tracker.Track(memoryID, metadata)

			Convey("Then it should create a provenance record", func() {
				So(err, ShouldBeNil)
				So(provenanceID, ShouldNotBeEmpty)
				So(provenanceID, ShouldStartWith, "prov_")
				
				// Verify record was stored
				record, err := tracker.GetProvenance(provenanceID)
				So(err, ShouldBeNil)
				So(record.MemoryID, ShouldEqual, memoryID)
				So(record.OriginalSource, ShouldEqual, metadata.Source)
				So(record.CreatedBy, ShouldEqual, metadata.UserID)
				So(record.Version, ShouldEqual, 1)
				So(record.IntegrityHash, ShouldNotBeEmpty)
			})
		})

		Convey("When getting non-existent provenance", func() {
			record, err := tracker.GetProvenance("non_existent")

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(record, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "not found")
			})
		})
	})
}

func TestProvenanceTrackerUpdate(t *testing.T) {
	Convey("Given a ProvenanceTracker with a tracked memory", t, func() {
		tracker := NewProvenanceTracker()
		
		memoryID := "memory_123"
		metadata := WriteMetadata{
			Source:    "test_source",
			Timestamp: time.Now(),
			UserID:    "test_user",
		}
		
		provenanceID, _ := tracker.Track(memoryID, metadata)

		Convey("When updating provenance with a transformation", func() {
			transformation := Transformation{
				ID:          "trans_1",
				Type:        TransformationEmbedding,
				Description: "Generated embeddings",
				Timestamp:   time.Now(),
				Agent:       "embedding_service",
				Parameters: map[string]interface{}{
					"model": "text-embedding-ada-002",
				},
			}

			err := tracker.UpdateProvenance(provenanceID, transformation)

			Convey("Then it should update the record", func() {
				So(err, ShouldBeNil)
				
				record, err := tracker.GetProvenance(provenanceID)
				So(err, ShouldBeNil)
				So(record.Version, ShouldEqual, 2)
				So(len(record.Transformations), ShouldEqual, 1)
				So(record.Transformations[0].Type, ShouldEqual, TransformationEmbedding)
				So(len(record.ParentVersions), ShouldEqual, 1)
			})
		})

		Convey("When tracking a transformation", func() {
			err := tracker.TrackTransformation(
				provenanceID,
				TransformationChunking,
				"Split text into chunks",
				"chunking_service",
				map[string]interface{}{
					"chunk_size": 1000,
					"overlap":    200,
				},
			)

			Convey("Then it should add the transformation", func() {
				So(err, ShouldBeNil)
				
				transformations, err := tracker.GetTransformationHistory(provenanceID)
				So(err, ShouldBeNil)
				So(len(transformations), ShouldEqual, 1)
				So(transformations[0].Type, ShouldEqual, TransformationChunking)
				So(transformations[0].Agent, ShouldEqual, "chunking_service")
			})
		})

		Convey("When updating non-existent provenance", func() {
			transformation := Transformation{
				Type: TransformationUpdate,
			}
			
			err := tracker.UpdateProvenance("non_existent", transformation)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "not found")
			})
		})
	})
}

func TestProvenanceTrackerLineage(t *testing.T) {
	Convey("Given a ProvenanceTracker with multiple versions", t, func() {
		tracker := NewProvenanceTracker()
		
		memoryID := "memory_123"
		
		// Create initial version
		metadata1 := WriteMetadata{
			Source:    "source_v1",
			Timestamp: time.Now(),
			UserID:    "user1",
		}
		_, _ = tracker.Track(memoryID, metadata1)
		
		// Create second version
		metadata2 := WriteMetadata{
			Source:    "source_v2",
			Timestamp: time.Now().Add(time.Hour),
			UserID:    "user2",
		}
		_, _ = tracker.Track(memoryID, metadata2)

		Convey("When getting memory lineage", func() {
			lineage, err := tracker.GetMemoryLineage(memoryID)

			Convey("Then it should return all versions", func() {
				So(err, ShouldBeNil)
				So(len(lineage), ShouldEqual, 2)
				
				// Verify both records are for the same memory
				for _, record := range lineage {
					So(record.MemoryID, ShouldEqual, memoryID)
				}
			})
		})

		Convey("When getting lineage for non-existent memory", func() {
			lineage, err := tracker.GetMemoryLineage("non_existent")

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(lineage, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "no provenance records found")
			})
		})
	})
}

func TestProvenanceTrackerIntegrity(t *testing.T) {
	Convey("Given a ProvenanceTracker with integrity checking", t, func() {
		tracker := NewProvenanceTracker()
		
		memoryID := "memory_123"
		metadata := WriteMetadata{
			Source:    "test_source",
			Timestamp: time.Now(),
			UserID:    "test_user",
		}
		
		provenanceID, _ := tracker.Track(memoryID, metadata)

		Convey("When verifying integrity of valid records", func() {
			err := tracker.VerifyIntegrity()

			Convey("Then it should pass", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When manually corrupting a record", func() {
			// Manually corrupt the record
			record := tracker.records[provenanceID]
			record.IntegrityHash = "corrupted_hash"

			err := tracker.VerifyIntegrity()

			Convey("Then it should detect corruption", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "integrity check failed")
			})
		})

		Convey("When integrity checking is disabled", func() {
			tracker.config.EnableIntegrityCheck = false
			
			// Corrupt the record
			record := tracker.records[provenanceID]
			record.IntegrityHash = "corrupted_hash"

			err := tracker.VerifyIntegrity()

			Convey("Then it should pass without checking", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestProvenanceTrackerVersioning(t *testing.T) {
	Convey("Given a ProvenanceTracker with versioning enabled", t, func() {
		tracker := NewProvenanceTracker()
		
		memoryID := "memory_123"
		metadata := WriteMetadata{
			Source:    "test_source",
			Timestamp: time.Now(),
			UserID:    "test_user",
		}
		
		provenanceID, _ := tracker.Track(memoryID, metadata)

		Convey("When making multiple updates", func() {
			// Add multiple transformations
			for i := 0; i < 5; i++ {
				transformation := Transformation{
					Type:        TransformationUpdate,
					Description: "Update " + string(rune(i)),
					Timestamp:   time.Now(),
					Agent:       "test_agent",
				}
				tracker.UpdateProvenance(provenanceID, transformation)
			}

			record, _ := tracker.GetProvenance(provenanceID)

			Convey("Then it should track versions", func() {
				So(record.Version, ShouldEqual, 6) // Initial + 5 updates
				So(len(record.ParentVersions), ShouldEqual, 5)
				So(len(record.Transformations), ShouldEqual, 5)
			})
		})

		Convey("When exceeding max version history", func() {
			// Set low max version history
			tracker.config.MaxVersionHistory = 2
			
			// Add multiple transformations
			for i := 0; i < 5; i++ {
				transformation := Transformation{
					Type:      TransformationUpdate,
					Timestamp: time.Now(),
					Agent:     "test_agent",
				}
				tracker.UpdateProvenance(provenanceID, transformation)
			}

			record, _ := tracker.GetProvenance(provenanceID)

			Convey("Then it should limit version history", func() {
				So(len(record.ParentVersions), ShouldBeLessThanOrEqualTo, 2)
			})
		})

		Convey("When versioning is disabled", func() {
			tracker.config.EnableVersioning = false
			
			transformation := Transformation{
				Type:      TransformationUpdate,
				Timestamp: time.Now(),
				Agent:     "test_agent",
			}
			tracker.UpdateProvenance(provenanceID, transformation)

			record, _ := tracker.GetProvenance(provenanceID)

			Convey("Then it should not increment version", func() {
				So(record.Version, ShouldEqual, 1)
				So(len(record.ParentVersions), ShouldEqual, 0)
			})
		})
	})
}

func TestProvenanceTrackerStats(t *testing.T) {
	Convey("Given a ProvenanceTracker with tracked memories", t, func() {
		tracker := NewProvenanceTracker()
		
		// Track multiple memories with transformations
		for i := 0; i < 3; i++ {
			memoryID := "memory_" + string(rune(i))
			metadata := WriteMetadata{
				Source:    "test_source",
				Timestamp: time.Now(),
				UserID:    "test_user",
			}
			
			provenanceID, _ := tracker.Track(memoryID, metadata)
			
			// Add some transformations
			tracker.TrackTransformation(provenanceID, TransformationChunking, "chunk", "chunker", nil)
			tracker.TrackTransformation(provenanceID, TransformationEmbedding, "embed", "embedder", nil)
		}

		Convey("When getting stats", func() {
			stats := tracker.GetStats()

			Convey("Then it should return accurate statistics", func() {
				So(stats["total_records"], ShouldEqual, 3)
				So(stats["total_transformations"], ShouldEqual, 6)
				So(stats["versioning_enabled"], ShouldBeTrue)
				So(stats["integrity_enabled"], ShouldBeTrue)
				
				transformationCounts := stats["transformation_counts"].(map[TransformationType]int)
				So(transformationCounts[TransformationChunking], ShouldEqual, 3)
				So(transformationCounts[TransformationEmbedding], ShouldEqual, 3)
			})
		})
	})
}