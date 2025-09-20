package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ClaimExtractor extracts factual claims from text
type ClaimExtractor struct {
	verbPatterns     []*regexp.Regexp
	factualIndicators []string
	minConfidence    float64
	maxClaimLength   int
}

// ClaimType represents different types of claims
type ClaimType string

const (
	FactualClaim     ClaimType = "FACTUAL"
	OpinionClaim     ClaimType = "OPINION"
	DefinitionClaim  ClaimType = "DEFINITION"
	CausalClaim      ClaimType = "CAUSAL"
	TemporalClaim    ClaimType = "TEMPORAL"
	ComparativeClaim ClaimType = "COMPARATIVE"
)

// ClaimExtractionResult represents a claim extraction result
type ClaimExtractionResult struct {
	Claim     *Claim    `json:"claim"`
	ClaimType ClaimType `json:"claim_type"`
	Context   string    `json:"context"`
	Position  int       `json:"position"`
	Length    int       `json:"length"`
}

// NewClaimExtractor creates a new ClaimExtractor with default settings
func NewClaimExtractor() *ClaimExtractor {
	extractor := &ClaimExtractor{
		minConfidence:  0.6,
		maxClaimLength: 200,
	}
	
	extractor.initializePatterns()
	extractor.initializeIndicators()
	
	return extractor
}

// ExtractClaims extracts claims from the given text
func (ce *ClaimExtractor) ExtractClaims(text, source string) ([]*Claim, error) {
	if text == "" {
		return []*Claim{}, nil
	}

	var claims []*Claim
	
	// Split text into sentences for claim extraction
	sentences := ce.splitIntoSentences(text)
	
	for _, sentence := range sentences {
		sentenceClaims := ce.extractClaimsFromSentence(sentence, source)
		claims = append(claims, sentenceClaims...)
	}
	
	// Filter and validate claims
	validClaims := ce.filterValidClaims(claims)
	
	return validClaims, nil
}

// ValidateClaim checks if a claim meets quality criteria
func (ce *ClaimExtractor) ValidateClaim(claim *Claim) bool {
	if claim == nil {
		return false
	}
	
	// Check basic validation
	if err := claim.Validate(); err != nil {
		return false
	}
	
	// Check confidence threshold
	if claim.Confidence < ce.minConfidence {
		return false
	}
	
	// Check claim length
	claimText := fmt.Sprintf("%s %s %s", claim.Subject, claim.Predicate, claim.Object)
	if len(claimText) > ce.maxClaimLength {
		return false
	}
	
	// Check for meaningful content
	if ce.isTriviaClaim(claim) {
		return false
	}
	
	return true
}

// ScoreClaim calculates a confidence score for a claim
func (ce *ClaimExtractor) ScoreClaim(claim *Claim, context string) float64 {
	score := 0.5 // Base score
	
	// Increase score based on factual indicators
	for _, indicator := range ce.factualIndicators {
		if strings.Contains(strings.ToLower(context), strings.ToLower(indicator)) {
			score += 0.1
			break
		}
	}
	
	// Increase score for specific patterns
	if ce.hasStrongVerb(claim.Predicate) {
		score += 0.2
	}
	
	// Increase score for proper nouns in subject/object
	if ce.hasProperNoun(claim.Subject) {
		score += 0.1
	}
	if ce.hasProperNoun(claim.Object) {
		score += 0.1
	}
	
	// Decrease score for opinion indicators
	opinionIndicators := []string{"think", "believe", "feel", "opinion", "seems", "appears"}
	for _, indicator := range opinionIndicators {
		if strings.Contains(strings.ToLower(context), indicator) {
			score -= 0.2
			break
		}
	}
	
	// Ensure score is within valid range
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}
	
	return score
}

// SetMinConfidence sets the minimum confidence threshold for claims
func (ce *ClaimExtractor) SetMinConfidence(confidence float64) {
	ce.minConfidence = confidence
}

// SetMaxClaimLength sets the maximum length for extracted claims
func (ce *ClaimExtractor) SetMaxClaimLength(length int) {
	ce.maxClaimLength = length
}

// initializePatterns sets up regex patterns for verb detection
func (ce *ClaimExtractor) initializePatterns() {
	verbPatterns := []string{
		`\b(is|are|was|were|am|be|been|being)\b`,           // Copula verbs
		`\b(has|have|had)\b`,                                // Possession verbs
		`\b(does|do|did|done|doing)\b`,                      // Action verbs
		`\b(will|would|shall|should|can|could|may|might)\b`, // Modal verbs
		`\b(says|said|states|stated|reports|reported)\b`,    // Reporting verbs
		`\b(shows|showed|demonstrates|demonstrated)\b`,      // Evidence verbs
		`\b(creates|created|makes|made|produces|produced)\b`, // Creation verbs
		`\b(contains|includes|involves|requires)\b`,         // Relation verbs
		`\b(sits|stands|runs|walks|moves|goes|comes)\b`,     // Common action verbs
		`\b(works|lives|stays|remains|exists)\b`,            // State verbs
		`\b(improves|enhances|increases|decreases|affects)\b`, // Change verbs
	}
	
	for _, pattern := range verbPatterns {
		regex, err := regexp.Compile(`(?i)` + pattern)
		if err == nil {
			ce.verbPatterns = append(ce.verbPatterns, regex)
		}
	}
}

// initializeIndicators sets up factual indicators
func (ce *ClaimExtractor) initializeIndicators() {
	ce.factualIndicators = []string{
		"according to", "research shows", "studies indicate", "data reveals",
		"statistics show", "evidence suggests", "findings indicate", "results show",
		"analysis reveals", "survey found", "report states", "documentation shows",
		"records indicate", "measurements show", "observations confirm",
	}
}

// splitIntoSentences splits text into sentences
func (ce *ClaimExtractor) splitIntoSentences(text string) []string {
	// Simple sentence splitting on periods, exclamation marks, and question marks
	sentencePattern := regexp.MustCompile(`[.!?]+\s+`)
	sentences := sentencePattern.Split(text, -1)
	
	var cleanSentences []string
	for _, sentence := range sentences {
		cleaned := strings.TrimSpace(sentence)
		if len(cleaned) > 10 { // Minimum sentence length
			cleanSentences = append(cleanSentences, cleaned)
		}
	}
	
	return cleanSentences
}

// extractClaimsFromSentence extracts claims from a single sentence
func (ce *ClaimExtractor) extractClaimsFromSentence(sentence, source string) []*Claim {
	var claims []*Claim
	
	// Try different claim extraction patterns
	claims = append(claims, ce.extractSubjectVerbObjectClaims(sentence, source)...)
	claims = append(claims, ce.extractDefinitionClaims(sentence, source)...)
	claims = append(claims, ce.extractCausalClaims(sentence, source)...)
	claims = append(claims, ce.extractTemporalClaims(sentence, source)...)
	
	return claims
}

// extractSubjectVerbObjectClaims extracts basic subject-verb-object claims
func (ce *ClaimExtractor) extractSubjectVerbObjectClaims(sentence, source string) []*Claim {
	var claims []*Claim
	
	// Look for verb patterns in the sentence
	for _, verbPattern := range ce.verbPatterns {
		matches := verbPattern.FindAllStringIndex(sentence, -1)
		
		for _, match := range matches {
			verbStart, verbEnd := match[0], match[1]
			verb := strings.TrimSpace(sentence[verbStart:verbEnd])
			
			// Extract subject (text before verb)
			subjectText := strings.TrimSpace(sentence[:verbStart])
			if subjectText == "" {
				continue
			}
			
			// Extract object (text after verb)
			objectText := strings.TrimSpace(sentence[verbEnd:])
			if objectText == "" {
				continue
			}
			
			// Clean up subject and object
			subject := ce.cleanClaimComponent(subjectText)
			object := ce.cleanClaimComponent(objectText)
			
			if subject != "" && object != "" {
				claim := NewClaim(
					fmt.Sprintf("claim_%d", time.Now().UnixNano()),
					subject,
					verb,
					object,
					source,
				)
				
				// Calculate confidence score
				confidence := ce.ScoreClaim(claim, sentence)
				claim.Confidence = confidence
				
				claims = append(claims, claim)
			}
		}
	}
	
	return claims
}

// extractDefinitionClaims extracts definition-style claims
func (ce *ClaimExtractor) extractDefinitionClaims(sentence, source string) []*Claim {
	var claims []*Claim
	
	definitionPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(.+?)\s+is\s+defined\s+as\s+(.+)`),
		regexp.MustCompile(`(?i)(.+?)\s+means\s+(.+)`),
		regexp.MustCompile(`(?i)(.+?)\s+refers\s+to\s+(.+)`),
		regexp.MustCompile(`(?i)(.+?):\s+(.+)`), // Colon-based definitions
	}
	
	for _, pattern := range definitionPatterns {
		matches := pattern.FindStringSubmatch(sentence)
		if len(matches) >= 3 {
			subject := ce.cleanClaimComponent(matches[1])
			object := ce.cleanClaimComponent(matches[2])
			
			if subject != "" && object != "" {
				claim := NewClaim(
					fmt.Sprintf("def_claim_%d", time.Now().UnixNano()),
					subject,
					"is_defined_as",
					object,
					source,
				)
				
				confidence := ce.ScoreClaim(claim, sentence)
				claim.Confidence = confidence + 0.1 // Boost for definition patterns
				
				claims = append(claims, claim)
			}
		}
	}
	
	return claims
}

// extractCausalClaims extracts causal relationship claims
func (ce *ClaimExtractor) extractCausalClaims(sentence, source string) []*Claim {
	var claims []*Claim
	
	causalPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(.+?)\s+causes?\s+(.+)`),
		regexp.MustCompile(`(?i)(.+?)\s+leads?\s+to\s+(.+)`),
		regexp.MustCompile(`(?i)(.+?)\s+results?\s+in\s+(.+)`),
		regexp.MustCompile(`(?i)because\s+of\s+(.+?),\s+(.+)`),
		regexp.MustCompile(`(?i)due\s+to\s+(.+?),\s+(.+)`),
	}
	
	for _, pattern := range causalPatterns {
		matches := pattern.FindStringSubmatch(sentence)
		if len(matches) >= 3 {
			cause := ce.cleanClaimComponent(matches[1])
			effect := ce.cleanClaimComponent(matches[2])
			
			if cause != "" && effect != "" {
				claim := NewClaim(
					fmt.Sprintf("causal_claim_%d", time.Now().UnixNano()),
					cause,
					"causes",
					effect,
					source,
				)
				
				confidence := ce.ScoreClaim(claim, sentence)
				claim.Confidence = confidence
				
				claims = append(claims, claim)
			}
		}
	}
	
	return claims
}

// extractTemporalClaims extracts temporal relationship claims
func (ce *ClaimExtractor) extractTemporalClaims(sentence, source string) []*Claim {
	var claims []*Claim
	
	temporalPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(.+?)\s+before\s+(.+)`),
		regexp.MustCompile(`(?i)(.+?)\s+after\s+(.+)`),
		regexp.MustCompile(`(?i)(.+?)\s+during\s+(.+)`),
		regexp.MustCompile(`(?i)in\s+(\d{4}),\s+(.+)`),
		regexp.MustCompile(`(?i)on\s+([A-Za-z]+\s+\d{1,2},?\s+\d{4}),\s+(.+)`),
	}
	
	for _, pattern := range temporalPatterns {
		matches := pattern.FindStringSubmatch(sentence)
		if len(matches) >= 3 {
			temporal := ce.cleanClaimComponent(matches[1])
			event := ce.cleanClaimComponent(matches[2])
			
			if temporal != "" && event != "" {
				claim := NewClaim(
					fmt.Sprintf("temporal_claim_%d", time.Now().UnixNano()),
					event,
					"occurs_during",
					temporal,
					source,
				)
				
				confidence := ce.ScoreClaim(claim, sentence)
				claim.Confidence = confidence
				
				claims = append(claims, claim)
			}
		}
	}
	
	return claims
}

// cleanClaimComponent cleans and normalizes claim components
func (ce *ClaimExtractor) cleanClaimComponent(text string) string {
	// Remove extra whitespace
	text = strings.TrimSpace(text)
	
	// Remove common prefixes and suffixes
	text = strings.TrimPrefix(text, "the ")
	text = strings.TrimPrefix(text, "a ")
	text = strings.TrimPrefix(text, "an ")
	text = strings.TrimSuffix(text, ".")
	text = strings.TrimSuffix(text, ",")
	text = strings.TrimSuffix(text, ";")
	
	// Limit length
	if len(text) > 100 {
		text = text[:100] + "..."
	}
	
	return strings.TrimSpace(text)
}

// filterValidClaims filters out invalid and low-quality claims
func (ce *ClaimExtractor) filterValidClaims(claims []*Claim) []*Claim {
	var validClaims []*Claim
	seen := make(map[string]bool)
	
	for _, claim := range claims {
		if !ce.ValidateClaim(claim) {
			continue
		}
		
		// Create unique key for deduplication
		key := fmt.Sprintf("%s|%s|%s", 
			strings.ToLower(claim.Subject),
			strings.ToLower(claim.Predicate),
			strings.ToLower(claim.Object))
		
		if seen[key] {
			continue
		}
		
		seen[key] = true
		validClaims = append(validClaims, claim)
	}
	
	return validClaims
}

// hasStrongVerb checks if the predicate contains a strong factual verb
func (ce *ClaimExtractor) hasStrongVerb(predicate string) bool {
	strongVerbs := []string{"is", "are", "was", "were", "has", "have", "contains", "includes", "improves", "enhances", "increases", "decreases", "affects", "shows", "demonstrates"}
	predicate = strings.ToLower(predicate)
	
	for _, verb := range strongVerbs {
		if strings.Contains(predicate, verb) {
			return true
		}
	}
	
	return false
}

// hasProperNoun checks if text contains proper nouns (capitalized words)
func (ce *ClaimExtractor) hasProperNoun(text string) bool {
	words := strings.Fields(text)
	for _, word := range words {
		if len(word) > 1 && strings.Title(word) == word {
			return true
		}
	}
	return false
}

// isTriviaClaim checks if a claim is too trivial or obvious
func (ce *ClaimExtractor) isTriviaClaim(claim *Claim) bool {
	// Check for very short components
	if len(claim.Subject) < 3 || len(claim.Object) < 3 {
		return true
	}
	
	// Check for common trivial patterns
	trivialPatterns := []string{
		"this is", "that is", "it is", "there is", "here is",
		"i am", "you are", "we are", "they are",
	}
	
	claimText := strings.ToLower(fmt.Sprintf("%s %s %s", 
		claim.Subject, claim.Predicate, claim.Object))
	
	for _, pattern := range trivialPatterns {
		if strings.Contains(claimText, pattern) {
			return true
		}
	}
	
	return false
}