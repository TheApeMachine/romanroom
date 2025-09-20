package main

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// TextChunker provides various text chunking strategies
type TextChunker struct {
	maxChunkSize     int
	overlapSize      int
	sentencePattern  *regexp.Regexp
	paragraphPattern *regexp.Regexp
}

// ChunkStrategy defines different chunking approaches
type ChunkStrategy int

const (
	ChunkBySize ChunkStrategy = iota
	ChunkBySentence
	ChunkByParagraph
)

// ChunkResult represents the result of chunking text
type ChunkResult struct {
	Text     string `json:"text"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Strategy string `json:"strategy"`
}

// NewTextChunker creates a new TextChunker with default settings
func NewTextChunker(maxChunkSize, overlapSize int) *TextChunker {
	return &TextChunker{
		maxChunkSize:     maxChunkSize,
		overlapSize:      overlapSize,
		sentencePattern:  regexp.MustCompile(`[.!?]+\s+`),
		paragraphPattern: regexp.MustCompile(`\n\s*\n`),
	}
}

// ChunkBySize splits text into chunks of approximately maxChunkSize characters
func (tc *TextChunker) ChunkBySize(text string) []ChunkResult {
	if text == "" {
		return []ChunkResult{}
	}

	var chunks []ChunkResult
	textLen := utf8.RuneCountInString(text)
	
	if textLen <= tc.maxChunkSize {
		return []ChunkResult{{
			Text:     text,
			Start:    0,
			End:      textLen,
			Strategy: "size",
		}}
	}

	start := 0
	for start < textLen {
		end := start + tc.maxChunkSize
		if end > textLen {
			end = textLen
		}

		// Try to break at word boundary
		chunkText := string([]rune(text)[start:end])
		if end < textLen {
			// Look for last space to avoid breaking words
			lastSpace := strings.LastIndex(chunkText, " ")
			if lastSpace > tc.maxChunkSize/2 { // Only if we don't lose too much content
				end = start + lastSpace
				chunkText = string([]rune(text)[start:end])
			}
		}

		chunks = append(chunks, ChunkResult{
			Text:     strings.TrimSpace(chunkText),
			Start:    start,
			End:      end,
			Strategy: "size",
		})

		// Move start position with overlap
		newStart := end - tc.overlapSize
		if newStart <= start {
			// Ensure we make progress
			newStart = start + 1
		}
		start = newStart
	}

	return chunks
}

// ChunkBySentence splits text into chunks based on sentence boundaries
func (tc *TextChunker) ChunkBySentence(text string) []ChunkResult {
	if text == "" {
		return []ChunkResult{}
	}

	sentences := tc.splitIntoSentences(text)
	if len(sentences) == 0 {
		return []ChunkResult{{
			Text:     text,
			Start:    0,
			End:      utf8.RuneCountInString(text),
			Strategy: "sentence",
		}}
	}

	var chunks []ChunkResult
	var currentChunk strings.Builder
	var chunkStart, chunkEnd int
	currentSize := 0

	for i, sentence := range sentences {
		sentenceSize := utf8.RuneCountInString(sentence.text)
		
		// If single sentence exceeds max size, split it by size
		if sentenceSize > tc.maxChunkSize {
			// Finalize current chunk if it has content
			if currentSize > 0 {
				chunks = append(chunks, ChunkResult{
					Text:     strings.TrimSpace(currentChunk.String()),
					Start:    chunkStart,
					End:      chunkEnd,
					Strategy: "sentence",
				})
				currentChunk.Reset()
				currentSize = 0
			}
			
			// Split the oversized sentence by size
			sizeChunks := tc.ChunkBySize(sentence.text)
			for _, sizeChunk := range sizeChunks {
				chunks = append(chunks, ChunkResult{
					Text:     sizeChunk.Text,
					Start:    sentence.start + sizeChunk.Start,
					End:      sentence.start + sizeChunk.End,
					Strategy: "sentence-size",
				})
			}
			continue
		}
		
		// If adding this sentence would exceed max size, finalize current chunk
		if currentSize > 0 && currentSize+sentenceSize > tc.maxChunkSize {
			chunks = append(chunks, ChunkResult{
				Text:     strings.TrimSpace(currentChunk.String()),
				Start:    chunkStart,
				End:      chunkEnd,
				Strategy: "sentence",
			})
			
			// Start new chunk with overlap
			currentChunk.Reset()
			currentSize = 0
			
			// Add overlap from previous sentences if available
			overlapStart := max(0, i-2)
			for j := overlapStart; j < i; j++ {
				if currentSize+utf8.RuneCountInString(sentences[j].text) <= tc.overlapSize {
					currentChunk.WriteString(sentences[j].text)
					currentSize += utf8.RuneCountInString(sentences[j].text)
					if j == overlapStart {
						chunkStart = sentences[j].start
					}
				}
			}
		}

		// Add current sentence
		if currentSize == 0 {
			chunkStart = sentence.start
		}
		currentChunk.WriteString(sentence.text)
		currentSize += sentenceSize
		chunkEnd = sentence.end
	}

	// Add final chunk if there's content
	if currentSize > 0 {
		chunks = append(chunks, ChunkResult{
			Text:     strings.TrimSpace(currentChunk.String()),
			Start:    chunkStart,
			End:      chunkEnd,
			Strategy: "sentence",
		})
	}

	return chunks
}

// ChunkByParagraph splits text into chunks based on paragraph boundaries
func (tc *TextChunker) ChunkByParagraph(text string) []ChunkResult {
	if text == "" {
		return []ChunkResult{}
	}

	paragraphs := tc.splitIntoParagraphs(text)
	if len(paragraphs) == 0 {
		return []ChunkResult{{
			Text:     text,
			Start:    0,
			End:      utf8.RuneCountInString(text),
			Strategy: "paragraph",
		}}
	}

	var chunks []ChunkResult
	var currentChunk strings.Builder
	var chunkStart, chunkEnd int
	currentSize := 0

	for _, paragraph := range paragraphs {
		paragraphSize := utf8.RuneCountInString(paragraph.text)
		
		// If single paragraph exceeds max size, split it by sentences
		if paragraphSize > tc.maxChunkSize {
			// Finalize current chunk if it has content
			if currentSize > 0 {
				chunks = append(chunks, ChunkResult{
					Text:     strings.TrimSpace(currentChunk.String()),
					Start:    chunkStart,
					End:      chunkEnd,
					Strategy: "paragraph",
				})
				currentChunk.Reset()
				currentSize = 0
			}
			
			// Split large paragraph by sentences
			sentenceChunks := tc.ChunkBySentence(paragraph.text)
			for _, chunk := range sentenceChunks {
				chunks = append(chunks, ChunkResult{
					Text:     chunk.Text,
					Start:    paragraph.start + chunk.Start,
					End:      paragraph.start + chunk.End,
					Strategy: "paragraph-sentence",
				})
			}
			continue
		}
		
		// If adding this paragraph would exceed max size, finalize current chunk
		if currentSize > 0 && currentSize+paragraphSize > tc.maxChunkSize {
			chunks = append(chunks, ChunkResult{
				Text:     strings.TrimSpace(currentChunk.String()),
				Start:    chunkStart,
				End:      chunkEnd,
				Strategy: "paragraph",
			})
			
			// Start new chunk
			currentChunk.Reset()
			currentSize = 0
		}

		// Add current paragraph
		if currentSize == 0 {
			chunkStart = paragraph.start
		}
		if currentSize > 0 {
			currentChunk.WriteString("\n\n")
			currentSize += 2
		}
		currentChunk.WriteString(paragraph.text)
		currentSize += paragraphSize
		chunkEnd = paragraph.end
	}

	// Add final chunk if there's content
	if currentSize > 0 {
		chunks = append(chunks, ChunkResult{
			Text:     strings.TrimSpace(currentChunk.String()),
			Start:    chunkStart,
			End:      chunkEnd,
			Strategy: "paragraph",
		})
	}

	return chunks
}

// sentenceInfo holds information about a sentence
type sentenceInfo struct {
	text  string
	start int
	end   int
}

// paragraphInfo holds information about a paragraph
type paragraphInfo struct {
	text  string
	start int
	end   int
}

// splitIntoSentences splits text into sentences with position information
func (tc *TextChunker) splitIntoSentences(text string) []sentenceInfo {
	var sentences []sentenceInfo
	
	matches := tc.sentencePattern.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		// No sentence boundaries found, treat as single sentence
		return []sentenceInfo{{
			text:  text,
			start: 0,
			end:   utf8.RuneCountInString(text),
		}}
	}

	start := 0
	for _, match := range matches {
		end := match[1]
		sentenceText := text[start:end]
		sentences = append(sentences, sentenceInfo{
			text:  sentenceText,
			start: utf8.RuneCountInString(text[:start]),
			end:   utf8.RuneCountInString(text[:end]),
		})
		start = end
	}

	// Add remaining text as final sentence if any
	if start < len(text) {
		sentenceText := text[start:]
		sentences = append(sentences, sentenceInfo{
			text:  sentenceText,
			start: utf8.RuneCountInString(text[:start]),
			end:   utf8.RuneCountInString(text),
		})
	}

	return sentences
}

// splitIntoParagraphs splits text into paragraphs with position information
func (tc *TextChunker) splitIntoParagraphs(text string) []paragraphInfo {
	var paragraphs []paragraphInfo
	
	matches := tc.paragraphPattern.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		// No paragraph boundaries found, treat as single paragraph
		return []paragraphInfo{{
			text:  text,
			start: 0,
			end:   utf8.RuneCountInString(text),
		}}
	}

	start := 0
	for _, match := range matches {
		end := match[0] // Use start of match (before the newlines)
		if end > start {
			paragraphText := strings.TrimSpace(text[start:end])
			if paragraphText != "" {
				paragraphs = append(paragraphs, paragraphInfo{
					text:  paragraphText,
					start: utf8.RuneCountInString(text[:start]),
					end:   utf8.RuneCountInString(text[:end]),
				})
			}
		}
		start = match[1] // Start after the newlines
	}

	// Add remaining text as final paragraph if any
	if start < len(text) {
		paragraphText := strings.TrimSpace(text[start:])
		if paragraphText != "" {
			paragraphs = append(paragraphs, paragraphInfo{
				text:  paragraphText,
				start: utf8.RuneCountInString(text[:start]),
				end:   utf8.RuneCountInString(text),
			})
		}
	}

	return paragraphs
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}