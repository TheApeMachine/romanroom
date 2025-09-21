package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// EntityExtractor extracts entities from text using regex patterns and keyword matching
type EntityExtractor struct {
	patterns      map[string]*regexp.Regexp
	keywords      map[string][]string
	minConfidence float64
}

// EntityType represents different types of entities
type EntityType string

const (
	PersonEntity       EntityType = "PERSON"
	OrganizationEntity EntityType = "ORGANIZATION"
	LocationEntity     EntityType = "LOCATION"
	DateEntity         EntityType = "DATE"
	NumberEntity       EntityType = "NUMBER"
	EmailEntity        EntityType = "EMAIL"
	URLEntity          EntityType = "URL"
	PhoneEntity        EntityType = "PHONE"
	ConceptEntity      EntityType = "CONCEPT"
)

// ExtractionResult represents an entity extraction result
type ExtractionResult struct {
	Entity   *Entity `json:"entity"`
	Position int     `json:"position"`
	Length   int     `json:"length"`
	Context  string  `json:"context"`
}

// NewEntityExtractor creates a new EntityExtractor with default patterns
func NewEntityExtractor() *EntityExtractor {
	extractor := &EntityExtractor{
		patterns:      make(map[string]*regexp.Regexp),
		keywords:      make(map[string][]string),
		minConfidence: 0.5,
	}

	extractor.initializePatterns()
	extractor.initializeKeywords()

	return extractor
}

// Extract extracts entities from the given text
func (ee *EntityExtractor) Extract(text, source string) ([]*Entity, error) {
	if text == "" {
		return []*Entity{}, nil
	}

	var entities []*Entity

	// Extract using regex patterns
	patternEntities := ee.extractByPatterns(text, source)
	entities = append(entities, patternEntities...)

	// Extract using keyword matching
	keywordEntities := ee.extractByKeywords(text, source)
	entities = append(entities, keywordEntities...)

	// Filter and validate entities
	validEntities := ee.FilterEntities(entities)

	return validEntities, nil
}

// ValidateEntity checks if an entity meets quality criteria
func (ee *EntityExtractor) ValidateEntity(entity *Entity) bool {
	if entity == nil {
		return false
	}

	// Check basic validation
	if err := entity.Validate(); err != nil {
		return false
	}

	// Check confidence threshold
	if entity.Confidence < ee.minConfidence {
		return false
	}

	// Check name length and content
	name := strings.TrimSpace(entity.Name)
	if len(name) < 2 || len(name) > 100 {
		return false
	}

	// Check for common false positives
	if ee.isFalsePositive(name, entity.Type) {
		return false
	}

	return true
}

// FilterEntities removes invalid and duplicate entities
func (ee *EntityExtractor) FilterEntities(entities []*Entity) []*Entity {
	var filtered []*Entity
	seen := make(map[string]bool)

	for _, entity := range entities {
		if !ee.ValidateEntity(entity) {
			continue
		}

		// Create unique key for deduplication
		key := fmt.Sprintf("%s:%s", strings.ToLower(entity.Name), entity.Type)
		if seen[key] {
			continue
		}

		seen[key] = true
		filtered = append(filtered, entity)
	}

	return filtered
}

// SetMinConfidence sets the minimum confidence threshold for entities
func (ee *EntityExtractor) SetMinConfidence(confidence float64) {
	ee.minConfidence = confidence
}

// AddPattern adds a custom regex pattern for entity extraction
func (ee *EntityExtractor) AddPattern(entityType string, pattern string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %v", err)
	}

	ee.patterns[entityType] = regex
	return nil
}

// AddKeywords adds keywords for entity type detection
func (ee *EntityExtractor) AddKeywords(entityType string, keywords []string) {
	ee.keywords[entityType] = append(ee.keywords[entityType], keywords...)
}

// initializePatterns sets up default regex patterns for entity extraction
func (ee *EntityExtractor) initializePatterns() {
	patterns := map[string]string{
		string(EmailEntity):  `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
		string(URLEntity):    `https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`,
		string(PhoneEntity):  `\b(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}\b`,
		string(DateEntity):   `\b(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{1,2},?\s+\d{4}\b|\b\d{1,2}[/-]\d{1,2}[/-]\d{2,4}\b|\b\d{4}-\d{2}-\d{2}\b`,
		string(NumberEntity): `\b\d{1,3}(?:,\d{3})*(?:\.\d+)?\b|\b\d+(?:\.\d+)?%\b`,
	}

	for entityType, pattern := range patterns {
		regex, err := regexp.Compile(pattern)
		if err == nil {
			ee.patterns[entityType] = regex
		}
	}
}

// initializeKeywords sets up default keywords for entity type detection
func (ee *EntityExtractor) initializeKeywords() {
	ee.keywords = map[string][]string{
		string(PersonEntity): {
			"Mr.", "Mrs.", "Ms.", "Dr.", "Prof.", "CEO", "President", "Director",
			"Manager", "Engineer", "Developer", "Analyst", "Consultant",
		},
		string(OrganizationEntity): {
			"Inc.", "Corp.", "LLC", "Ltd.", "Company", "Corporation", "University",
			"College", "School", "Hospital", "Bank", "Agency", "Department",
		},
		string(LocationEntity): {
			"Street", "St.", "Avenue", "Ave.", "Road", "Rd.", "Boulevard", "Blvd.",
			"City", "State", "Country", "County", "Province", "District",
		},
		string(ConceptEntity): {
			"algorithm", "system", "process", "method", "technique", "approach",
			"framework", "model", "theory", "concept", "principle", "strategy",
		},
	}
}

// extractByPatterns extracts entities using regex patterns
func (ee *EntityExtractor) extractByPatterns(text, source string) []*Entity {
	var entities []*Entity

	for entityType, pattern := range ee.patterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		indices := pattern.FindAllStringIndex(text, -1)

		for i, match := range matches {
			if len(match) > 0 {
				entityText := strings.TrimSpace(match[0])
				if entityText == "" {
					continue
				}

				// Calculate confidence based on pattern match quality
				confidence := ee.calculatePatternConfidence(entityText, entityType)

				// Get context around the match
				context := ee.getContext(text, indices[i][0], indices[i][1])

				entity := NewEntity(
					ee.generateEntityID(entityType, entityText),
					entityText,
					entityType,
					source,
				)
				entity.Confidence = confidence
				entity.SetProperty("context", context)
				entity.SetProperty("extraction_method", "pattern")

				entities = append(entities, entity)
			}
		}
	}

	return entities
}

// extractByKeywords extracts entities using keyword matching
func (ee *EntityExtractor) extractByKeywords(text, source string) []*Entity {
	var entities []*Entity
	words := strings.Fields(text)

	for i, word := range words {
		cleanWord := strings.Trim(word, ".,!?;:()[]{}\"'")

		for entityType, keywords := range ee.keywords {
			for _, keyword := range keywords {
				// Check both exact match and trimmed match (for titles like "Dr.", "Mr.")
				keywordMatch := strings.EqualFold(cleanWord, keyword) ||
					strings.EqualFold(cleanWord, strings.Trim(keyword, "."))

				if keywordMatch {
					// Look for potential entity name near the keyword
					entityName := ee.findEntityNearKeyword(words, i, entityType)
					if entityName != "" {
						confidence := ee.calculateKeywordConfidence(entityName, keyword, entityType)

						entity := NewEntity(
							ee.generateEntityID(entityType, entityName),
							entityName,
							entityType,
							source,
						)
						entity.Confidence = confidence
						entity.SetProperty("keyword", keyword)
						entity.SetProperty("extraction_method", "keyword")

						entities = append(entities, entity)
					}
				}
			}
		}
	}

	return entities
}

// findEntityNearKeyword finds potential entity names near a keyword
func (ee *EntityExtractor) findEntityNearKeyword(words []string, keywordIndex int, entityType string) string {
	// Look for capitalized words near the keyword
	searchRange := 3 // Look 3 words before and after

	for offset := -searchRange; offset <= searchRange; offset++ {
		index := keywordIndex + offset
		if index < 0 || index >= len(words) || index == keywordIndex {
			continue
		}

		word := strings.Trim(words[index], ".,!?;:()[]{}\"'")

		// Check if word looks like an entity name
		if ee.looksLikeEntityName(word, entityType) {
			// Try to get full name by looking at adjacent capitalized words
			return ee.expandEntityName(words, index)
		}
	}

	return ""
}

// looksLikeEntityName checks if a word looks like an entity name
func (ee *EntityExtractor) looksLikeEntityName(word, entityType string) bool {
	if len(word) < 2 {
		return false
	}

	// Check if first letter is capitalized
	if strings.Title(word) != word {
		return false
	}

	// Additional checks based on entity type
	switch EntityType(entityType) {
	case PersonEntity:
		return ee.looksLikePersonName(word)
	case OrganizationEntity:
		return ee.looksLikeOrganizationName(word)
	case LocationEntity:
		return ee.looksLikeLocationName(word)
	default:
		return true
	}
}

// expandEntityName expands a single word to a full entity name
func (ee *EntityExtractor) expandEntityName(words []string, startIndex int) string {
	var nameParts []string

	// Start with the word at startIndex
	startWord := strings.Trim(words[startIndex], ".,!?;:()[]{}\"'")
	if strings.Title(startWord) == startWord && len(startWord) > 1 {
		nameParts = append(nameParts, startWord)
	}

	// Add words before if they're capitalized
	for i := startIndex - 1; i >= 0; i-- {
		word := strings.Trim(words[i], ".,!?;:()[]{}\"'")
		if strings.Title(word) == word && len(word) > 1 {
			nameParts = append([]string{word}, nameParts...)
		} else {
			break
		}
	}

	// Add words after if they're capitalized
	for i := startIndex + 1; i < len(words); i++ {
		word := strings.Trim(words[i], ".,!?;:()[]{}\"'")
		if strings.Title(word) == word && len(word) > 1 {
			nameParts = append(nameParts, word)
		} else {
			break
		}
	}

	return strings.Join(nameParts, " ")
}

// calculatePatternConfidence calculates confidence for pattern-based extraction
func (ee *EntityExtractor) calculatePatternConfidence(text, entityType string) float64 {
	baseConfidence := 0.8

	// Adjust based on text length and characteristics
	if len(text) < 3 {
		baseConfidence -= 0.2
	}

	// Type-specific adjustments
	switch EntityType(entityType) {
	case EmailEntity, URLEntity, PhoneEntity:
		return 0.95 // High confidence for structured patterns
	case DateEntity, NumberEntity:
		return 0.85 // Good confidence for numeric patterns
	default:
		return baseConfidence
	}
}

// calculateKeywordConfidence calculates confidence for keyword-based extraction
func (ee *EntityExtractor) calculateKeywordConfidence(entityName, keyword, _ string) float64 {
	baseConfidence := 0.6

	// Adjust based on entity name quality
	if len(entityName) > 10 {
		baseConfidence += 0.1
	}

	// Adjust based on keyword specificity
	if len(keyword) > 5 {
		baseConfidence += 0.1
	}

	return baseConfidence
}

// getContext extracts context around a match
func (ee *EntityExtractor) getContext(text string, start, end int) string {
	contextSize := 50

	contextStart := start - contextSize
	if contextStart < 0 {
		contextStart = 0
	}

	contextEnd := end + contextSize
	if contextEnd > len(text) {
		contextEnd = len(text)
	}

	return strings.TrimSpace(text[contextStart:contextEnd])
}

// isFalsePositive checks for common false positive patterns
func (ee *EntityExtractor) isFalsePositive(name, _ string) bool {
	name = strings.ToLower(name)

	// Common false positives
	falsePositives := []string{
		"the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with",
		"by", "from", "up", "about", "into", "through", "during", "before",
		"after", "above", "below", "between", "among", "this", "that", "these",
		"those", "i", "you", "he", "she", "it", "we", "they", "me", "him", "her",
		"us", "them", "my", "your", "his", "her", "its", "our", "their",
	}

	for _, fp := range falsePositives {
		if name == fp {
			return true
		}
	}

	return false
}

// Helper methods for entity type checking
func (ee *EntityExtractor) looksLikePersonName(word string) bool {
	// Simple heuristic: check if it's not a common word
	commonWords := []string{"The", "And", "Or", "But", "In", "On", "At", "To", "For"}
	for _, common := range commonWords {
		if word == common {
			return false
		}
	}
	return true
}

func (ee *EntityExtractor) looksLikeOrganizationName(word string) bool {
	// Organizations often have specific suffixes or are proper nouns
	return len(word) > 2
}

func (ee *EntityExtractor) looksLikeLocationName(word string) bool {
	// Locations are typically proper nouns
	return len(word) > 2
}

// generateEntityID generates a unique ID for an entity using a hash-based approach
// This prevents collisions that can occur with time-based IDs in fast tests
func (ee *EntityExtractor) generateEntityID(entityType, entityName string) string {
	// Use a combination of type, name, and current time with more entropy
	input := fmt.Sprintf("%s:%s:%d:%d", entityType, entityName, time.Now().UnixNano(), time.Now().Unix())

	// Simple hash to create a more unique ID
	hash := 0
	for _, c := range input {
		hash = hash*31 + int(c)
	}

	// Make it positive and add more entropy
	if hash < 0 {
		hash = -hash
	}

	return fmt.Sprintf("%s_%d", entityType, hash)
}
