package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
)

// RecallArgsValidator validates and sanitizes recall arguments
type RecallArgsValidator struct {
	config *RecallValidatorConfig
}

// RecallValidatorConfig holds configuration for argument validation
type RecallValidatorConfig struct {
	MaxQueryLength  int           `json:"max_query_length"`
	MinQueryLength  int           `json:"min_query_length"`
	MaxResults      int           `json:"max_results"`
	MaxTimeBudget   time.Duration `json:"max_time_budget"`
	AllowedFilters  []string      `json:"allowed_filters"`
	BlockedPatterns []string      `json:"blocked_patterns"`
	SanitizeHTML    bool          `json:"sanitize_html"`
	ValidateUTF8    bool          `json:"validate_utf8"`
}

// ValidationError represents a validation error with details
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s", e.Field, e.Message)
}

// ValidationResult contains validation results and sanitized arguments
type ValidationResult struct {
	Valid     bool              `json:"valid"`
	Errors    []ValidationError `json:"errors"`
	Warnings  []string          `json:"warnings"`
	Sanitized RecallArgs        `json:"sanitized"`
}

// NewRecallArgsValidator creates a new validator with default configuration
func NewRecallArgsValidator() *RecallArgsValidator {
	config := &RecallValidatorConfig{
		MaxQueryLength:  1000,
		MinQueryLength:  1,
		MaxResults:      100,
		MaxTimeBudget:   30 * time.Second,
		AllowedFilters:  []string{"source", "type", "confidence", "date", "tag", "user_id"},
		BlockedPatterns: []string{`<script`, `javascript:`, `data:`, `vbscript:`},
		SanitizeHTML:    true,
		ValidateUTF8:    true,
	}

	return &RecallArgsValidator{
		config: config,
	}
}

// NewRecallArgsValidatorWithConfig creates a validator with custom configuration
func NewRecallArgsValidatorWithConfig(config *RecallValidatorConfig) *RecallArgsValidator {
	if config == nil {
		return NewRecallArgsValidator()
	}

	return &RecallArgsValidator{
		config: config,
	}
}

// Validate validates recall arguments and returns detailed results
func (v *RecallArgsValidator) Validate(args RecallArgs) error {
	result := v.ValidateDetailed(args)
	if !result.Valid {
		if len(result.Errors) > 0 {
			return result.Errors[0]
		}
		return fmt.Errorf("validation failed")
	}
	return nil
}

// ValidateDetailed performs comprehensive validation and returns detailed results
func (v *RecallArgsValidator) ValidateDetailed(args RecallArgs) *ValidationResult {
	result := &ValidationResult{
		Valid:     true,
		Errors:    make([]ValidationError, 0),
		Warnings:  make([]string, 0),
		Sanitized: args, // Start with original args
	}

	// Validate query
	v.validateQuery(args.Query, result)

	// Validate max results
	v.validateMaxResults(args.MaxResults, result)

	// Validate time budget
	v.validateTimeBudget(args.TimeBudget, result)

	// Validate filters
	v.validateFilters(args.Filters, result)

	// Check for blocked patterns
	v.checkBlockedPatterns(args.Query, result)

	// Set overall validity
	result.Valid = len(result.Errors) == 0

	return result
}

// CheckArgs performs basic argument validation
func (v *RecallArgsValidator) CheckArgs(args RecallArgs) []string {
	var issues []string

	if strings.TrimSpace(args.Query) == "" {
		issues = append(issues, "query cannot be empty")
	}

	if len(args.Query) > v.config.MaxQueryLength {
		issues = append(issues, fmt.Sprintf("query exceeds maximum length of %d characters", v.config.MaxQueryLength))
	}

	if args.MaxResults < 0 {
		issues = append(issues, "maxResults cannot be negative")
	}

	if args.MaxResults > v.config.MaxResults {
		issues = append(issues, fmt.Sprintf("maxResults exceeds maximum of %d", v.config.MaxResults))
	}

	if args.TimeBudget < 0 {
		issues = append(issues, "timeBudget cannot be negative")
	}

	maxTimeBudgetMs := int(v.config.MaxTimeBudget.Milliseconds())
	if args.TimeBudget > maxTimeBudgetMs {
		issues = append(issues, fmt.Sprintf("timeBudget exceeds maximum of %d milliseconds", maxTimeBudgetMs))
	}

	return issues
}

// SanitizeInput sanitizes and normalizes input arguments
func (v *RecallArgsValidator) SanitizeInput(args RecallArgs) (RecallArgs, error) {
	result := v.ValidateDetailed(args)
	if !result.Valid {
		if len(result.Errors) > 0 {
			return RecallArgs{}, result.Errors[0]
		}
		return RecallArgs{}, fmt.Errorf("validation failed")
	}

	// Return the already sanitized result from ValidateDetailed
	return result.Sanitized, nil
}

// validateQuery validates the query string
func (v *RecallArgsValidator) validateQuery(query string, result *ValidationResult) {
	// Check if query is empty
	if strings.TrimSpace(query) == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "query",
			Message: "query cannot be empty",
			Value:   query,
		})
		return
	}

	// Check query length
	if len(query) < v.config.MinQueryLength {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "query",
			Message: fmt.Sprintf("query must be at least %d characters", v.config.MinQueryLength),
			Value:   len(query),
		})
	}

	if len(query) > v.config.MaxQueryLength {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "query",
			Message: fmt.Sprintf("query exceeds maximum length of %d characters", v.config.MaxQueryLength),
			Value:   len(query),
		})
	}

	// Validate UTF-8 if enabled
	if v.config.ValidateUTF8 && !v.isValidUTF8(query) {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "query",
			Message: "query contains invalid UTF-8 characters",
			Value:   query,
		})
	}

	// Check for suspicious patterns
	if v.containsSuspiciousPatterns(query) {
		result.Warnings = append(result.Warnings, "query contains potentially suspicious patterns")
	}

	// Sanitize query
	result.Sanitized.Query = v.sanitizeQuery(query)
}

// validateMaxResults validates the maxResults parameter
func (v *RecallArgsValidator) validateMaxResults(maxResults int, result *ValidationResult) {
	if maxResults < 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "maxResults",
			Message: "maxResults cannot be negative",
			Value:   maxResults,
		})
		return
	}

	if maxResults > v.config.MaxResults {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "maxResults",
			Message: fmt.Sprintf("maxResults exceeds maximum of %d", v.config.MaxResults),
			Value:   maxResults,
		})
		// Cap at maximum
		result.Sanitized.MaxResults = v.config.MaxResults
		result.Warnings = append(result.Warnings, fmt.Sprintf("maxResults capped at %d", v.config.MaxResults))
	}

	// Set default if zero
	if maxResults == 0 {
		result.Sanitized.MaxResults = 10 // Default value
		result.Warnings = append(result.Warnings, "maxResults set to default value of 10")
	}
}

// validateTimeBudget validates the timeBudget parameter
func (v *RecallArgsValidator) validateTimeBudget(timeBudget int, result *ValidationResult) {
	if timeBudget < 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "timeBudget",
			Message: "timeBudget cannot be negative",
			Value:   timeBudget,
		})
		return
	}

	maxTimeBudgetMs := int(v.config.MaxTimeBudget.Milliseconds())
	if timeBudget > maxTimeBudgetMs {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "timeBudget",
			Message: fmt.Sprintf("timeBudget exceeds maximum of %d milliseconds", maxTimeBudgetMs),
			Value:   timeBudget,
		})
		// Cap at maximum
		result.Sanitized.TimeBudget = maxTimeBudgetMs
		result.Warnings = append(result.Warnings, fmt.Sprintf("timeBudget capped at %d milliseconds", maxTimeBudgetMs))
	}

	// Set default if zero
	if timeBudget == 0 {
		result.Sanitized.TimeBudget = 5000 // Default 5 seconds
		result.Warnings = append(result.Warnings, "timeBudget set to default value of 5000ms")
	}
}

// validateFilters validates the filters parameter
func (v *RecallArgsValidator) validateFilters(filters map[string]interface{}, result *ValidationResult) {
	if filters == nil {
		return
	}

	sanitizedFilters := make(map[string]interface{})

	for key, value := range filters {
		// Check if filter key is allowed
		if !v.isAllowedFilter(key) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("filter key '%s' is not in allowed list", key))
			continue
		}

		// Sanitize filter value
		sanitizedValue := v.sanitizeFilterValue(value)
		if sanitizedValue != nil {
			sanitizedFilters[key] = sanitizedValue
		} else {
			result.Warnings = append(result.Warnings, fmt.Sprintf("filter value for key '%s' could not be sanitized", key))
		}
	}

	result.Sanitized.Filters = sanitizedFilters
}

// checkBlockedPatterns checks for blocked patterns in the query
func (v *RecallArgsValidator) checkBlockedPatterns(query string, result *ValidationResult) {
	lowerQuery := strings.ToLower(query)

	for _, pattern := range v.config.BlockedPatterns {
		if strings.Contains(lowerQuery, strings.ToLower(pattern)) {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "query",
				Message: fmt.Sprintf("query contains blocked pattern: %s", pattern),
				Value:   pattern,
			})
		}
	}
}

// sanitizeQuery sanitizes the query string
func (v *RecallArgsValidator) sanitizeQuery(query string) string {
	// Trim whitespace
	sanitized := strings.TrimSpace(query)

	// HTML escape if enabled
	if v.config.SanitizeHTML {
		sanitized = html.EscapeString(sanitized)
	}

	// Remove control characters
	sanitized = v.removeControlCharacters(sanitized)

	// Normalize whitespace
	sanitized = v.normalizeWhitespace(sanitized)

	return sanitized
}

// sanitizeFilters sanitizes the filters map
func (v *RecallArgsValidator) sanitizeFilters(filters map[string]interface{}) map[string]interface{} {
	if filters == nil {
		return nil
	}

	sanitized := make(map[string]interface{})

	for key, value := range filters {
		if v.isAllowedFilter(key) {
			sanitizedValue := v.sanitizeFilterValue(value)
			if sanitizedValue != nil {
				sanitized[key] = sanitizedValue
			}
		}
	}

	return sanitized
}

// sanitizeFilterValue sanitizes a filter value
func (v *RecallArgsValidator) sanitizeFilterValue(value interface{}) interface{} {
	switch val := value.(type) {
	case string:
		sanitized := strings.TrimSpace(val)
		if v.config.SanitizeHTML {
			sanitized = html.EscapeString(sanitized)
		}
		sanitized = v.removeControlCharacters(sanitized)
		return sanitized
	case int, int32, int64, float32, float64, bool:
		return val
	case []interface{}:
		var sanitizedSlice []interface{}
		for _, item := range val {
			if sanitizedItem := v.sanitizeFilterValue(item); sanitizedItem != nil {
				sanitizedSlice = append(sanitizedSlice, sanitizedItem)
			}
		}
		return sanitizedSlice
	default:
		// For unknown types, convert to string and sanitize
		str := fmt.Sprintf("%v", val)
		return v.sanitizeFilterValue(str)
	}
}

// isAllowedFilter checks if a filter key is allowed
func (v *RecallArgsValidator) isAllowedFilter(key string) bool {
	for _, allowed := range v.config.AllowedFilters {
		if key == allowed {
			return true
		}
	}
	return false
}

// isValidUTF8 checks if a string is valid UTF-8
func (v *RecallArgsValidator) isValidUTF8(s string) bool {
	return strings.ToValidUTF8(s, "") == s
}

// containsSuspiciousPatterns checks for suspicious patterns
func (v *RecallArgsValidator) containsSuspiciousPatterns(query string) bool {
	suspiciousPatterns := []string{
		`(?i)<script`,
		`(?i)javascript:`,
		`(?i)data:text/html`,
		`(?i)vbscript:`,
		`(?i)onload=`,
		`(?i)onerror=`,
		`\x00`, // null bytes
	}

	for _, pattern := range suspiciousPatterns {
		matched, _ := regexp.MatchString(pattern, query)
		if matched {
			return true
		}
	}

	return false
}

// removeControlCharacters removes control characters from a string
func (v *RecallArgsValidator) removeControlCharacters(s string) string {
	// Remove control characters except tab, newline, and carriage return
	re := regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)
	return re.ReplaceAllString(s, "")
}

// normalizeWhitespace normalizes whitespace in a string
func (v *RecallArgsValidator) normalizeWhitespace(s string) string {
	// Replace multiple whitespace characters with single space
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, " ")
}

// GetConfig returns the current configuration
func (v *RecallArgsValidator) GetConfig() *RecallValidatorConfig {
	return v.config
}

// UpdateConfig updates the configuration
func (v *RecallArgsValidator) UpdateConfig(config *RecallValidatorConfig) {
	if config != nil {
		v.config = config
	}
}

// ValidateAndSanitize is a convenience method that validates and sanitizes in one call
func (v *RecallArgsValidator) ValidateAndSanitize(args RecallArgs) (RecallArgs, []string, error) {
	result := v.ValidateDetailed(args)

	if !result.Valid {
		if len(result.Errors) > 0 {
			return RecallArgs{}, result.Warnings, result.Errors[0]
		}
		return RecallArgs{}, result.Warnings, fmt.Errorf("validation failed")
	}

	return result.Sanitized, result.Warnings, nil
}
