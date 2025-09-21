package main

import (
	"context"
	"fmt"
	"strings"
)

// QueryExpander handles query expansion for improved recall
type QueryExpander struct {
	config *QueryExpanderConfig
}

// QueryExpanderConfig holds configuration for query expansion
type QueryExpanderConfig struct {
	MaxExpansions     int     `json:"max_expansions"`
	EnableSynonyms    bool    `json:"enable_synonyms"`
	EnableParaphrases bool    `json:"enable_paraphrases"`
	EnableSpelling    bool    `json:"enable_spelling"`
	EnableAcronyms    bool    `json:"enable_acronyms"`
	SimilarityThreshold float64 `json:"similarity_threshold"`
}

// ExpansionResult contains the expanded query variations
type ExpansionResult struct {
	Original    string            `json:"original"`
	Expansions  []string          `json:"expansions"`
	Synonyms    []string          `json:"synonyms"`
	Paraphrases []string          `json:"paraphrases"`
	Spellings   []string          `json:"spellings"`
	Acronyms    []string          `json:"acronyms"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewQueryExpander creates a new QueryExpander instance
func NewQueryExpander() *QueryExpander {
	config := &QueryExpanderConfig{
		MaxExpansions:       8,
		EnableSynonyms:      true,
		EnableParaphrases:   true,
		EnableSpelling:      true,
		EnableAcronyms:      true,
		SimilarityThreshold: 0.7,
	}

	return &QueryExpander{
		config: config,
	}
}

// NewQueryExpanderWithConfig creates a QueryExpander with custom configuration
func NewQueryExpanderWithConfig(config *QueryExpanderConfig) *QueryExpander {
	if config == nil {
		return NewQueryExpander()
	}

	return &QueryExpander{
		config: config,
	}
}

// Expand generates query variations for improved recall
func (qe *QueryExpander) Expand(ctx context.Context, originalQuery string, parsed *ParsedQuery) ([]string, error) {
	if strings.TrimSpace(originalQuery) == "" {
		return []string{}, fmt.Errorf("empty query")
	}

	result := &ExpansionResult{
		Original:    originalQuery,
		Expansions:  make([]string, 0),
		Synonyms:    make([]string, 0),
		Paraphrases: make([]string, 0),
		Spellings:   make([]string, 0),
		Acronyms:    make([]string, 0),
		Metadata:    make(map[string]interface{}),
	}

	// Generate synonyms
	if qe.config.EnableSynonyms {
		synonyms := qe.GenerateSynonyms(originalQuery, parsed)
		result.Synonyms = synonyms
		result.Expansions = append(result.Expansions, synonyms...)
	}

	// Generate paraphrases
	if qe.config.EnableParaphrases {
		paraphrases := qe.generateParaphrases(originalQuery, parsed)
		result.Paraphrases = paraphrases
		result.Expansions = append(result.Expansions, paraphrases...)
	}

	// Generate spelling variations
	if qe.config.EnableSpelling {
		spellings := qe.generateSpellingVariations(originalQuery, parsed)
		result.Spellings = spellings
		result.Expansions = append(result.Expansions, spellings...)
	}

	// Generate acronym expansions
	if qe.config.EnableAcronyms {
		acronyms := qe.generateAcronymExpansions(originalQuery, parsed)
		result.Acronyms = acronyms
		result.Expansions = append(result.Expansions, acronyms...)
	}

	// Add context-based expansions only if any expansion type is enabled
	if qe.config.EnableSynonyms || qe.config.EnableParaphrases || qe.config.EnableSpelling || qe.config.EnableAcronyms {
		contextExpansions := qe.AddContext(originalQuery, parsed)
		result.Expansions = append(result.Expansions, contextExpansions...)
	}

	// Deduplicate and limit expansions
	result.Expansions = qe.deduplicateStrings(result.Expansions)
	if len(result.Expansions) > qe.config.MaxExpansions {
		result.Expansions = result.Expansions[:qe.config.MaxExpansions]
	}

	// Add metadata
	result.Metadata["expansion_count"] = len(result.Expansions)
	result.Metadata["synonym_count"] = len(result.Synonyms)
	result.Metadata["paraphrase_count"] = len(result.Paraphrases)
	result.Metadata["spelling_count"] = len(result.Spellings)
	result.Metadata["acronym_count"] = len(result.Acronyms)

	return result.Expansions, nil
}

// GenerateSynonyms creates synonym-based query variations
func (qe *QueryExpander) GenerateSynonyms(query string, parsed *ParsedQuery) []string {
	if parsed == nil {
		return []string{}
	}

	synonymMap := qe.buildSynonymMap()
	var synonymQueries []string

	// Generate synonyms for individual terms
	for _, term := range parsed.Terms {
		if synonyms, exists := synonymMap[strings.ToLower(term)]; exists {
			for _, synonym := range synonyms {
				// Replace term with synonym in original query
				synonymQuery := strings.ReplaceAll(query, term, synonym)
				if synonymQuery != query {
					synonymQueries = append(synonymQueries, synonymQuery)
				}
			}
		}
	}

	// Generate synonyms for phrases
	for _, phrase := range parsed.Phrases {
		words := strings.Fields(phrase)
		for i, word := range words {
			if synonyms, exists := synonymMap[strings.ToLower(word)]; exists {
				for _, synonym := range synonyms {
					// Replace word in phrase
					newWords := make([]string, len(words))
					copy(newWords, words)
					newWords[i] = synonym
					synonymPhrase := strings.Join(newWords, " ")
					
					// Replace phrase in original query
					synonymQuery := strings.ReplaceAll(query, phrase, synonymPhrase)
					if synonymQuery != query {
						synonymQueries = append(synonymQueries, synonymQuery)
					}
				}
			}
		}
	}

	return qe.deduplicateStrings(synonymQueries)
}

// AddContext adds contextual information to expand query understanding
func (qe *QueryExpander) AddContext(query string, parsed *ParsedQuery) []string {
	var contextQueries []string

	// Add temporal context if time range is detected
	if parsed != nil && parsed.TimeRange != nil {
		// Add explicit time-based variations
		timeVariations := []string{
			query + " recent",
			query + " latest",
			query + " current",
			query + " updated",
		}
		contextQueries = append(contextQueries, timeVariations...)
	}

	// Add domain-specific context based on query type
	if parsed != nil {
		switch parsed.QueryType {
		case QueryTypeEntity:
			entityVariations := []string{
				query + " information",
				query + " details",
				query + " overview",
				"about " + query,
			}
			contextQueries = append(contextQueries, entityVariations...)

		case QueryTypeSemantic:
			semanticVariations := []string{
				query + " explanation",
				query + " meaning",
				query + " definition",
				"what is " + query,
			}
			contextQueries = append(contextQueries, semanticVariations...)

		case QueryTypeKeyword:
			keywordVariations := []string{
				query + " examples",
				query + " usage",
				query + " applications",
			}
			contextQueries = append(contextQueries, keywordVariations...)
		}
	}

	// Add question variations
	questionVariations := []string{
		"what is " + query,
		"how does " + query + " work",
		"why " + query,
		"when " + query,
		"where " + query,
	}
	contextQueries = append(contextQueries, questionVariations...)

	return qe.deduplicateStrings(contextQueries)
}

// generateParaphrases creates paraphrased versions of the query
func (qe *QueryExpander) generateParaphrases(query string, parsed *ParsedQuery) []string {
	var paraphrases []string

	// Simple paraphrasing patterns
	paraphrasePatterns := map[string][]string{
		"how to":    {"ways to", "methods to", "steps to"},
		"what is":   {"define", "explain", "describe"},
		"why":       {"reason for", "cause of", "explanation for"},
		"when":      {"time of", "date of", "schedule for"},
		"where":     {"location of", "place of", "position of"},
		"who":       {"person", "individual", "people"},
		"which":     {"what", "that"},
		"best":      {"top", "optimal", "excellent", "superior"},
		"good":      {"effective", "quality", "reliable"},
		"fast":      {"quick", "rapid", "speedy"},
		"easy":      {"simple", "straightforward", "effortless"},
		"difficult": {"hard", "challenging", "complex"},
	}

	queryLower := strings.ToLower(query)
	
	for pattern, replacements := range paraphrasePatterns {
		if strings.Contains(queryLower, pattern) {
			for _, replacement := range replacements {
				paraphrase := strings.ReplaceAll(queryLower, pattern, replacement)
				if paraphrase != queryLower {
					paraphrases = append(paraphrases, paraphrase)
				}
			}
		}
	}

	// Generate structural paraphrases
	if parsed != nil && len(parsed.Terms) > 1 {
		// Reorder terms
		terms := make([]string, len(parsed.Terms))
		copy(terms, parsed.Terms)
		
		// Simple reordering (reverse)
		for i, j := 0, len(terms)-1; i < j; i, j = i+1, j-1 {
			terms[i], terms[j] = terms[j], terms[i]
		}
		reordered := strings.Join(terms, " ")
		if reordered != query {
			paraphrases = append(paraphrases, reordered)
		}
	}

	return qe.deduplicateStrings(paraphrases)
}

// generateSpellingVariations creates spelling variations and corrections
func (qe *QueryExpander) generateSpellingVariations(query string, parsed *ParsedQuery) []string {
	var spellingVariations []string

	// Common spelling corrections
	spellingCorrections := map[string][]string{
		"recieve":    {"receive"},
		"seperate":   {"separate"},
		"definately": {"definitely"},
		"occured":    {"occurred"},
		"begining":   {"beginning"},
		"existance":  {"existence"},
		"maintainance": {"maintenance"},
		"performace": {"performance"},
		"recomend":   {"recommend"},
		"sucessful":  {"successful"},
	}

	queryLower := strings.ToLower(query)
	words := strings.Fields(queryLower)

	for i, word := range words {
		if corrections, exists := spellingCorrections[word]; exists {
			for _, correction := range corrections {
				// Replace word with correction
				correctedWords := make([]string, len(words))
				copy(correctedWords, words)
				correctedWords[i] = correction
				corrected := strings.Join(correctedWords, " ")
				spellingVariations = append(spellingVariations, corrected)
			}
		}
	}

	// Generate common typo variations (simple character substitutions)
	if parsed != nil {
		for _, term := range parsed.Terms {
			if len(term) > 3 { // Only for longer terms
				variations := qe.generateTypoVariations(term)
				for _, variation := range variations {
					variantQuery := strings.ReplaceAll(query, term, variation)
					if variantQuery != query {
						spellingVariations = append(spellingVariations, variantQuery)
					}
				}
			}
		}
	}

	return qe.deduplicateStrings(spellingVariations)
}

// generateAcronymExpansions expands acronyms and creates acronym variations
func (qe *QueryExpander) generateAcronymExpansions(query string, parsed *ParsedQuery) []string {
	var acronymExpansions []string

	// Common acronym expansions
	acronymMap := map[string][]string{
		"ai":   {"artificial intelligence"},
		"ml":   {"machine learning"},
		"nlp":  {"natural language processing"},
		"api":  {"application programming interface"},
		"ui":   {"user interface"},
		"ux":   {"user experience"},
		"db":   {"database"},
		"os":   {"operating system"},
		"cpu":  {"central processing unit"},
		"gpu":  {"graphics processing unit"},
		"ram":  {"random access memory"},
		"ssd":  {"solid state drive"},
		"hdd":  {"hard disk drive"},
		"url":  {"uniform resource locator"},
		"http": {"hypertext transfer protocol"},
		"https": {"hypertext transfer protocol secure"},
		"ftp":  {"file transfer protocol"},
		"ssh":  {"secure shell"},
		"sql":  {"structured query language"},
		"json": {"javascript object notation"},
		"xml":  {"extensible markup language"},
		"html": {"hypertext markup language"},
		"css":  {"cascading style sheets"},
		"js":   {"javascript"},
		"ts":   {"typescript"},
	}

	queryLower := strings.ToLower(query)
	words := strings.Fields(queryLower)

	// Expand acronyms to full forms
	for i, word := range words {
		if expansions, exists := acronymMap[word]; exists {
			for _, expansion := range expansions {
				// Replace acronym with expansion
				expandedWords := make([]string, len(words))
				copy(expandedWords, words)
				expandedWords[i] = expansion
				expanded := strings.Join(expandedWords, " ")
				acronymExpansions = append(acronymExpansions, expanded)
			}
		}
	}

	// Create acronyms from multi-word terms
	if parsed != nil {
		for _, phrase := range parsed.Phrases {
			acronym := qe.createAcronym(phrase)
			if acronym != "" && len(acronym) >= 2 {
				// Replace phrase in original query (handle both quoted and unquoted)
				quotedPhrase := `"` + phrase + `"`
				acronymQuery := strings.ReplaceAll(query, quotedPhrase, acronym)
				if acronymQuery == query {
					// Try without quotes
					acronymQuery = strings.ReplaceAll(query, phrase, acronym)
				}
				if acronymQuery != query {
					acronymExpansions = append(acronymExpansions, acronymQuery)
				}
			}
		}
	}

	return qe.deduplicateStrings(acronymExpansions)
}

// buildSynonymMap creates a basic synonym mapping
func (qe *QueryExpander) buildSynonymMap() map[string][]string {
	return map[string][]string{
		"big":       {"large", "huge", "massive", "enormous"},
		"small":     {"little", "tiny", "miniature", "compact"},
		"fast":      {"quick", "rapid", "speedy", "swift"},
		"slow":      {"sluggish", "gradual", "leisurely"},
		"good":      {"excellent", "great", "fine", "quality"},
		"bad":       {"poor", "terrible", "awful", "inferior"},
		"easy":      {"simple", "effortless", "straightforward"},
		"hard":      {"difficult", "challenging", "tough", "complex"},
		"new":       {"recent", "latest", "modern", "fresh"},
		"old":       {"ancient", "vintage", "outdated", "legacy"},
		"important": {"significant", "crucial", "vital", "essential"},
		"help":      {"assist", "support", "aid", "guide"},
		"find":      {"locate", "discover", "search", "identify"},
		"create":    {"make", "build", "generate", "produce"},
		"delete":    {"remove", "erase", "eliminate", "destroy"},
		"update":    {"modify", "change", "revise", "refresh"},
		"show":      {"display", "present", "exhibit", "demonstrate"},
		"use":       {"utilize", "employ", "apply", "operate"},
		"get":       {"obtain", "acquire", "retrieve", "fetch"},
		"set":       {"configure", "establish", "define", "specify"},
	}
}

// generateTypoVariations creates simple typo variations
func (qe *QueryExpander) generateTypoVariations(term string) []string {
	if len(term) < 4 {
		return []string{}
	}

	var variations []string

	// Character swapping (transpose adjacent characters)
	for i := 0; i < len(term)-1; i++ {
		chars := []rune(term)
		chars[i], chars[i+1] = chars[i+1], chars[i]
		variations = append(variations, string(chars))
	}

	// Single character deletion
	for i := 0; i < len(term); i++ {
		variation := term[:i] + term[i+1:]
		if len(variation) >= 3 {
			variations = append(variations, variation)
		}
	}

	// Limit variations to prevent explosion
	if len(variations) > 3 {
		variations = variations[:3]
	}

	return variations
}

// createAcronym generates an acronym from a multi-word phrase
func (qe *QueryExpander) createAcronym(phrase string) string {
	words := strings.Fields(strings.ToLower(phrase))
	if len(words) < 2 {
		return ""
	}

	var acronym strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			acronym.WriteRune(rune(word[0]))
		}
	}

	return acronym.String()
}

// deduplicateStrings removes duplicate strings while preserving order
func (qe *QueryExpander) deduplicateStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0) // Ensure non-nil slice

	for _, str := range strs {
		normalized := strings.TrimSpace(strings.ToLower(str))
		if normalized != "" && !seen[normalized] {
			seen[normalized] = true
			result = append(result, str)
		}
	}

	return result
}