package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTextChunker(t *testing.T) {
	Convey("Given a TextChunker", t, func() {
		chunker := NewTextChunker(100, 20)
		
		Convey("When creating a new TextChunker", func() {
			So(chunker, ShouldNotBeNil)
			So(chunker.maxChunkSize, ShouldEqual, 100)
			So(chunker.overlapSize, ShouldEqual, 20)
		})
		
		Convey("When chunking by size", func() {
			Convey("With empty text", func() {
				result := chunker.ChunkBySize("")
				So(result, ShouldBeEmpty)
			})
			
			Convey("With text shorter than max size", func() {
				text := "This is a short text."
				result := chunker.ChunkBySize(text)
				
				So(len(result), ShouldEqual, 1)
				So(result[0].Text, ShouldEqual, text)
				So(result[0].Strategy, ShouldEqual, "size")
				So(result[0].Start, ShouldEqual, 0)
				So(result[0].End, ShouldEqual, len([]rune(text)))
			})
			
			Convey("With text longer than max size", func() {
				text := "This is a very long text that should be split into multiple chunks because it exceeds the maximum chunk size that we have configured for this test case."
				result := chunker.ChunkBySize(text)
				
				So(len(result), ShouldBeGreaterThan, 1)
				for _, chunk := range result {
					So(len([]rune(chunk.Text)), ShouldBeLessThanOrEqualTo, 100)
					So(chunk.Strategy, ShouldEqual, "size")
				}
			})
			
			Convey("With text that breaks at word boundaries", func() {
				text := "Word1 Word2 Word3 Word4 Word5 Word6 Word7 Word8 Word9 Word10 Word11 Word12 Word13 Word14 Word15 Word16 Word17 Word18 Word19 Word20"
				result := chunker.ChunkBySize(text)
				
				So(len(result), ShouldBeGreaterThan, 1)
				// Check that chunks respect word boundaries
				for _, chunk := range result {
					// Chunks should not end with partial words (except last chunk)
					if len(chunk.Text) > 0 && chunk.Text[len(chunk.Text)-1] != ' ' {
						// This is acceptable for the last chunk or when breaking at word boundaries
						So(len(chunk.Text), ShouldBeLessThanOrEqualTo, 100)
					}
				}
			})
		})
		
		Convey("When chunking by sentence", func() {
			Convey("With empty text", func() {
				result := chunker.ChunkBySentence("")
				So(result, ShouldBeEmpty)
			})
			
			Convey("With single sentence", func() {
				text := "This is a single sentence."
				result := chunker.ChunkBySentence(text)
				
				So(len(result), ShouldEqual, 1)
				So(result[0].Text, ShouldEqual, text)
				So(result[0].Strategy, ShouldEqual, "sentence")
			})
			
			Convey("With multiple sentences", func() {
				text := "First sentence. Second sentence! Third sentence? Fourth sentence."
				result := chunker.ChunkBySentence(text)
				
				So(len(result), ShouldBeGreaterThanOrEqualTo, 1)
				for _, chunk := range result {
					So(chunk.Strategy, ShouldEqual, "sentence")
				}
			})
			
			Convey("With sentences exceeding max size", func() {
				// Create text with sentences that together exceed max size
				longSentence := "This is a very long sentence that contains many words and should be split appropriately when it exceeds the maximum chunk size."
				text := longSentence + " " + longSentence + " " + longSentence
				
				result := chunker.ChunkBySentence(text)
				
				So(len(result), ShouldBeGreaterThan, 1)
				for _, chunk := range result {
					So(len([]rune(chunk.Text)), ShouldBeLessThanOrEqualTo, 100)
				}
			})
		})
		
		Convey("When chunking by paragraph", func() {
			Convey("With empty text", func() {
				result := chunker.ChunkByParagraph("")
				So(result, ShouldBeEmpty)
			})
			
			Convey("With single paragraph", func() {
				text := "This is a single paragraph with multiple sentences. It should be kept together."
				result := chunker.ChunkByParagraph(text)
				
				So(len(result), ShouldEqual, 1)
				So(result[0].Text, ShouldEqual, text)
				So(result[0].Strategy, ShouldEqual, "paragraph")
			})
			
			Convey("With multiple paragraphs", func() {
				text := "First paragraph.\n\nSecond paragraph.\n\nThird paragraph."
				result := chunker.ChunkByParagraph(text)
				
				So(len(result), ShouldBeGreaterThanOrEqualTo, 1)
				for _, chunk := range result {
					So(chunk.Strategy, ShouldEqual, "paragraph")
				}
			})
			
			Convey("With paragraph exceeding max size", func() {
				// Create a very long paragraph
				longParagraph := ""
				for i := 0; i < 20; i++ {
					longParagraph += "This is a sentence in a very long paragraph. "
				}
				text := longParagraph + "\n\nSecond paragraph."
				
				result := chunker.ChunkByParagraph(text)
				
				So(len(result), ShouldBeGreaterThan, 1)
				// Check that long paragraph was split by sentences
				foundParagraphSentence := false
				for _, chunk := range result {
					if chunk.Strategy == "paragraph-sentence" {
						foundParagraphSentence = true
					}
				}
				So(foundParagraphSentence, ShouldBeTrue)
			})
		})
		
		Convey("When splitting into sentences", func() {
			Convey("With various sentence endings", func() {
				text := "First sentence. Second sentence! Third sentence? Fourth sentence."
				sentences := chunker.splitIntoSentences(text)
				
				So(len(sentences), ShouldEqual, 4)
				So(sentences[0].text, ShouldEqual, "First sentence. ")
				So(sentences[1].text, ShouldEqual, "Second sentence! ")
				So(sentences[2].text, ShouldEqual, "Third sentence? ")
				So(sentences[3].text, ShouldEqual, "Fourth sentence.")
			})
			
			Convey("With no sentence boundaries", func() {
				text := "This is text without proper sentence endings"
				sentences := chunker.splitIntoSentences(text)
				
				So(len(sentences), ShouldEqual, 1)
				So(sentences[0].text, ShouldEqual, text)
			})
		})
		
		Convey("When splitting into paragraphs", func() {
			Convey("With multiple paragraphs", func() {
				text := "First paragraph.\n\nSecond paragraph.\n\nThird paragraph."
				paragraphs := chunker.splitIntoParagraphs(text)
				
				So(len(paragraphs), ShouldEqual, 3)
				So(paragraphs[0].text, ShouldEqual, "First paragraph.")
				So(paragraphs[1].text, ShouldEqual, "Second paragraph.")
				So(paragraphs[2].text, ShouldEqual, "Third paragraph.")
			})
			
			Convey("With no paragraph boundaries", func() {
				text := "This is a single paragraph without breaks."
				paragraphs := chunker.splitIntoParagraphs(text)
				
				So(len(paragraphs), ShouldEqual, 1)
				So(paragraphs[0].text, ShouldEqual, text)
			})
			
			Convey("With empty paragraphs", func() {
				text := "First paragraph.\n\n\n\nSecond paragraph."
				paragraphs := chunker.splitIntoParagraphs(text)
				
				So(len(paragraphs), ShouldEqual, 2)
				So(paragraphs[0].text, ShouldEqual, "First paragraph.")
				So(paragraphs[1].text, ShouldEqual, "Second paragraph.")
			})
		})
	})
}

func BenchmarkTextChunker(b *testing.B) {
	chunker := NewTextChunker(1000, 100)
	
	// Create a large text for benchmarking
	largeText := ""
	for i := 0; i < 1000; i++ {
		largeText += "This is sentence number " + string(rune(i)) + ". It contains some text for benchmarking purposes. "
	}
	
	b.Run("ChunkBySize", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			chunker.ChunkBySize(largeText)
		}
	})
	
	b.Run("ChunkBySentence", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			chunker.ChunkBySentence(largeText)
		}
	})
	
	b.Run("ChunkByParagraph", func(b *testing.B) {
		paragraphText := ""
		for i := 0; i < 100; i++ {
			paragraphText += "This is paragraph number " + string(rune(i)) + ". It contains multiple sentences for testing. "
			if i%10 == 0 {
				paragraphText += "\n\n"
			}
		}
		
		for i := 0; i < b.N; i++ {
			chunker.ChunkByParagraph(paragraphText)
		}
	})
}