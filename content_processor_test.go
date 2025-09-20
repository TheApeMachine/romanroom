package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestContentProcessor(t *testing.T) {
	Convey("Given a ContentProcessor", t, func() {
		processor := NewContentProcessor()
		
		Convey("When creating a new ContentProcessor", func() {
			So(processor, ShouldNotBeNil)
			So(processor.config, ShouldNotBeNil)
			So(processor.textChunker, ShouldNotBeNil)
			So(processor.entityExtractor, ShouldNotBeNil)
			So(processor.claimExtractor, ShouldNotBeNil)
			So(processor.config.MaxChunkSize, ShouldEqual, 1000)
			So(processor.config.ChunkOverlap, ShouldEqual, 100)
			So(processor.config.ChunkStrategy, ShouldEqual, "sentence")
		})
		
		Convey("When creating with custom config", func() {
			config := &ContentProcessingConfig{
				MaxChunkSize:        500,
				ChunkOverlap:        50,
				ChunkStrategy:       "paragraph",
				MinEntityConfidence: 0.7,
				MinClaimConfidence:  0.8,
				EnablePreprocessing: false,
			}
			
			customProcessor := NewContentProcessorWithConfig(config)
			
			So(customProcessor.config.MaxChunkSize, ShouldEqual, 500)
			So(customProcessor.config.ChunkStrategy, ShouldEqual, "paragraph")
			So(customProcessor.config.MinEntityConfidence, ShouldEqual, 0.7)
		})
		
		Convey("When processing content", func() {
			Convey("With empty content", func() {
				result, err := processor.Process("", "test_source")
				
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(result.Chunks, ShouldBeEmpty)
				So(result.Entities, ShouldBeEmpty)
				So(result.Claims, ShouldBeEmpty)
				So(result.Stats.OriginalLength, ShouldEqual, 0)
			})
			
			Convey("With simple content", func() {
				content := "The sky is blue. Dr. Smith works at ABC Corporation. Contact him at smith@example.com."
				result, err := processor.Process(content, "test_source")
				
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(len(result.Chunks), ShouldBeGreaterThan, 0)
				So(len(result.Entities), ShouldBeGreaterThan, 0)
				So(len(result.Claims), ShouldBeGreaterThan, 0)
				
				// Check stats
				So(result.Stats.OriginalLength, ShouldEqual, len(content))
				So(result.Stats.ChunkCount, ShouldEqual, len(result.Chunks))
				So(result.Stats.EntityCount, ShouldEqual, len(result.Entities))
				So(result.Stats.ClaimCount, ShouldEqual, len(result.Claims))
				So(result.Stats.ProcessingTime, ShouldBeGreaterThan, 0)
			})
			
			Convey("With long content requiring chunking", func() {
				// Create content longer than max chunk size
				longContent := ""
				for i := 0; i < 50; i++ {
					longContent += "This is sentence number " + string(rune(i+'0')) + " in a very long document. "
				}
				
				result, err := processor.Process(longContent, "test_source")
				
				So(err, ShouldBeNil)
				So(len(result.Chunks), ShouldBeGreaterThan, 1)
				
				// Check that chunks have proper metadata
				for i, chunk := range result.Chunks {
					chunkIndex, exists := chunk.GetMetadata("chunk_index")
					So(exists, ShouldBeTrue)
					So(chunkIndex, ShouldEqual, i)
					
					strategy, exists := chunk.GetMetadata("chunk_strategy")
					So(exists, ShouldBeTrue)
					So(strategy, ShouldEqual, "sentence")
				}
			})
			
			Convey("With content containing entities and claims", func() {
				content := `
					Dr. John Smith is a researcher at MIT University. 
					According to his research, artificial intelligence is defined as machine intelligence.
					He can be contacted at john.smith@mit.edu or by phone at (617) 555-0123.
					The study was published in 2024 and shows that AI improves productivity by 25%.
				`
				
				result, err := processor.Process(content, "test_source")
				
				So(err, ShouldBeNil)
				
				// Should extract various entity types
				entityTypes := make(map[string]bool)
				for _, entity := range result.Entities {
					entityTypes[entity.Type] = true
				}
				So(len(entityTypes), ShouldBeGreaterThan, 1)
				
				// Should extract claims
				So(len(result.Claims), ShouldBeGreaterThan, 0)
				
				// Chunks should contain entities and claims
				for _, chunk := range result.Chunks {
					entityCount, _ := chunk.GetMetadata("entity_count")
					claimCount, _ := chunk.GetMetadata("claim_count")
					So(entityCount.(int) + claimCount.(int), ShouldBeGreaterThan, 0)
				}
			})
		})
		
		Convey("When chunking content", func() {
			Convey("With size strategy", func() {
				processor.SetChunkStrategy("size")
				content := "This is a test document with multiple sentences. Each sentence should be processed correctly."
				
				chunks := processor.Chunk(content)
				
				So(len(chunks), ShouldBeGreaterThan, 0)
				for _, chunk := range chunks {
					So(chunk.Strategy, ShouldEqual, "size")
				}
			})
			
			Convey("With sentence strategy", func() {
				processor.SetChunkStrategy("sentence")
				content := "First sentence. Second sentence! Third sentence?"
				
				chunks := processor.Chunk(content)
				
				So(len(chunks), ShouldBeGreaterThan, 0)
				for _, chunk := range chunks {
					So(chunk.Strategy, ShouldEqual, "sentence")
				}
			})
			
			Convey("With paragraph strategy", func() {
				processor.SetChunkStrategy("paragraph")
				content := "First paragraph.\n\nSecond paragraph.\n\nThird paragraph."
				
				chunks := processor.Chunk(content)
				
				So(len(chunks), ShouldBeGreaterThan, 0)
				for _, chunk := range chunks {
					So(chunk.Strategy, ShouldEqual, "paragraph")
				}
			})
			
			Convey("With unknown strategy defaults to sentence", func() {
				processor.SetChunkStrategy("unknown")
				content := "Test content for unknown strategy."
				
				chunks := processor.Chunk(content)
				
				So(len(chunks), ShouldBeGreaterThan, 0)
				for _, chunk := range chunks {
					So(chunk.Strategy, ShouldEqual, "sentence")
				}
			})
		})
		
		Convey("When preprocessing content", func() {
			Convey("With whitespace normalization", func() {
				content := "Text  with   multiple    spaces\t\tand\ttabs\n\n\n\nand newlines."
				processed := processor.Preprocess(content)
				
				So(processed, ShouldNotContainSubstring, "  ")
				So(processed, ShouldNotContainSubstring, "\t")
				So(processed, ShouldNotContainSubstring, "\n\n\n")
			})
			
			Convey("With encoding issues", func() {
				content := "Text with â€™ and â€œ encoding issues â€."
				processed := processor.Preprocess(content)
				
				So(processed, ShouldContainSubstring, "'")
				So(processed, ShouldContainSubstring, "\"")
			})
			
			Convey("With special characters", func() {
				content := "Text with \"smart quotes\" and — em dashes."
				processed := processor.Preprocess(content)
				
				So(processed, ShouldContainSubstring, "\"")
				So(processed, ShouldContainSubstring, " - ")
			})
			
			Convey("With punctuation issues", func() {
				content := "Text with,bad spacing .And missing spaces."
				processed := processor.Preprocess(content)
				
				So(processed, ShouldContainSubstring, ", ")
				So(processed, ShouldContainSubstring, ". ")
			})
			
			Convey("With empty content", func() {
				processed := processor.Preprocess("")
				So(processed, ShouldEqual, "")
			})
		})
		
		Convey("When configuring processor", func() {
			Convey("Setting chunk strategy", func() {
				processor.SetChunkStrategy("paragraph")
				So(processor.config.ChunkStrategy, ShouldEqual, "paragraph")
			})
			
			Convey("Setting max chunk size", func() {
				processor.SetMaxChunkSize(500)
				So(processor.config.MaxChunkSize, ShouldEqual, 500)
			})
			
			Convey("Setting chunk overlap", func() {
				processor.SetChunkOverlap(50)
				So(processor.config.ChunkOverlap, ShouldEqual, 50)
			})
			
			Convey("Setting min entity confidence", func() {
				processor.SetMinEntityConfidence(0.8)
				So(processor.config.MinEntityConfidence, ShouldEqual, 0.8)
			})
			
			Convey("Setting min claim confidence", func() {
				processor.SetMinClaimConfidence(0.9)
				So(processor.config.MinClaimConfidence, ShouldEqual, 0.9)
			})
			
			Convey("Enabling/disabling preprocessing", func() {
				processor.EnablePreprocessing(false)
				So(processor.config.EnablePreprocessing, ShouldBeFalse)
				
				processor.EnablePreprocessing(true)
				So(processor.config.EnablePreprocessing, ShouldBeTrue)
			})
			
			Convey("Getting config", func() {
				config := processor.GetConfig()
				So(config, ShouldNotBeNil)
				So(config, ShouldEqual, processor.config)
			})
		})
		
		Convey("When processing with preprocessing disabled", func() {
			processor.EnablePreprocessing(false)
			content := "Text  with   bad    formatting."
			
			result, err := processor.Process(content, "test_source")
			
			So(err, ShouldBeNil)
			// Content should not be preprocessed
			So(result.Chunks[0].Content, ShouldContainSubstring, "   ")
		})
		
		Convey("When measuring processing performance", func() {
			content := `
				This is a comprehensive test document with multiple sentences and paragraphs.
				Dr. Jane Doe works at XYZ Corporation and can be reached at jane.doe@xyz.com.
				According to recent studies, machine learning improves efficiency by 30%.
				The research was conducted in 2024 and published in Nature journal.
				
				The second paragraph contains additional information about the methodology.
				Statistical analysis shows significant improvements in all measured parameters.
				Contact the research team at research@xyz.com for more details.
			`
			
			result, err := processor.Process(content, "test_source")
			
			So(err, ShouldBeNil)
			So(result.Stats.ProcessingTime, ShouldBeGreaterThan, 0)
			So(result.Stats.ChunkingTime, ShouldBeGreaterThan, 0)
			So(result.Stats.EntityExtractionTime, ShouldBeGreaterThan, 0)
			So(result.Stats.ClaimExtractionTime, ShouldBeGreaterThan, 0)
		})
	})
}

func BenchmarkContentProcessor(b *testing.B) {
	processor := NewContentProcessor()
	
	// Create test content with various elements
	content := `
		Dr. John Smith is a leading researcher at MIT University in the field of artificial intelligence.
		According to his latest research published in 2024, machine learning algorithms can improve
		productivity by up to 40% in manufacturing environments. The study analyzed data from over
		1,000 companies worldwide and found significant correlations between AI adoption and efficiency.
		
		Contact Dr. Smith at john.smith@mit.edu or call (617) 555-0123 for more information.
		Visit the research website at https://ai-research.mit.edu for detailed findings.
		
		The implications of this research are far-reaching. Companies that implement AI solutions
		see immediate benefits in cost reduction and quality improvement. The technology is particularly
		effective in predictive maintenance and quality control applications.
	`
	
	b.Run("Process", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(content, "benchmark_source")
		}
	})
	
	b.Run("Chunk", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Chunk(content)
		}
	})
	
	b.Run("Preprocess", func(b *testing.B) {
		messyContent := "Text  with   bad    formatting\t\tand\tencoding â€™ issues."
		for i := 0; i < b.N; i++ {
			processor.Preprocess(messyContent)
		}
	})
	
	// Benchmark with large content
	largeContent := ""
	for i := 0; i < 100; i++ {
		largeContent += content
	}
	
	b.Run("ProcessLargeContent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(largeContent, "benchmark_source")
		}
	})
	
	// Benchmark different chunking strategies
	b.Run("ChunkBySize", func(b *testing.B) {
		processor.SetChunkStrategy("size")
		for i := 0; i < b.N; i++ {
			processor.Chunk(content)
		}
	})
	
	b.Run("ChunkBySentence", func(b *testing.B) {
		processor.SetChunkStrategy("sentence")
		for i := 0; i < b.N; i++ {
			processor.Chunk(content)
		}
	})
	
	b.Run("ChunkByParagraph", func(b *testing.B) {
		processor.SetChunkStrategy("paragraph")
		for i := 0; i < b.N; i++ {
			processor.Chunk(content)
		}
	})
}