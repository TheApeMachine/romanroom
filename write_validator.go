package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

// WriteArgsValidator validates and sanitizes write arguments
type WriteArgsValidator struct {
	config *WriteValidatorConfig
}

// WriteValidatorConfig holds configuration for argument validation
type WriteValidatorConfig struct {
	MaxContentLength       int      `json:"max_content_length"`
	MinContentLength       int      `json:"min_content_length"`
	MaxSourceLength        int      `json:"max_source_length"`
	MaxTagCount            int      `json:"max_tag_count"`
	MaxTagLength           int      `json:"max_tag_length"`
	RequireSource          bool     `json:"require_source"`
	AllowedContentTypes    []string `json:"allowed_content_types"`
	BlockedPatterns        []string `json:"blocked_patterns"`
	SanitizeHTML           bool     `json:"sanitize_html"`
	ValidateUTF8           bool     `json:"validate_utf8"`
	MaxMetadataKeys        int      `json:"max_metadata_keys"`
	MaxMetadataValueLength int      `json:"max_metadata_value_length"`
}

// WriteValidationError represents a validation error with details
type WriteValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

func (e WriteValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s", e.Field, e.Message)
}

// WriteValidationResult contains validation results and sanitized arguments
type WriteValidationResult struct {
	Valid     bool                   `json:"valid"`
	Errors    []WriteValidationError `json:"errors"`
	Warnings  []string               `json:"warnings"`
	Sanitized WriteArgs              `json:"sanitized"`
}

// NewWriteArgsValidator creates a new validator with default configuration
func NewWriteArgsValidator() *WriteArgsValidator {
	config := &WriteValidatorConfig{
		MaxContentLength:    10000,
		MinContentLength:    1,
		MaxSourceLength:     200,
		MaxTagCount:         10,
		MaxTagLength:        50,
		RequireSource:       true,
		AllowedContentTypes: []string{"text/plain", "text/markdown", "text/html", "application/json"},
		BlockedPatterns: []string{
			`(?i)<script[^>]*>`,
			`(?i)\bjavascript:`,
			`(?i)\bdata:(?:text|image|application)/`,
			`(?i)\bvbscript:`,
		},
		SanitizeHTML:           true,
		ValidateUTF8:           true,
		MaxMetadataKeys:        20,
		MaxMetadataValueLength: 500,
	}

	return &WriteArgsValidator{
		config: config,
	}
}

// NewWriteArgsValidatorWithConfig creates a validator with custom configuration
func NewWriteArgsValidatorWithConfig(config *WriteValidatorConfig) *WriteArgsValidator {
	if config == nil {
		return NewWriteArgsValidator()
	}

	return &WriteArgsValidator{
		config: config,
	}
}

// Validate validates write arguments and returns detailed results
func (v *WriteArgsValidator) Validate(args WriteArgs) error {
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
func (v *WriteArgsValidator) ValidateDetailed(args WriteArgs) *WriteValidationResult {
	result := &WriteValidationResult{
		Valid:     true,
		Errors:    make([]WriteValidationError, 0),
		Warnings:  make([]string, 0),
		Sanitized: args, // Start with original args
	}

	// Validate content
	v.validateContent(args.Content, result)

	// Validate source
	v.validateSource(args.Source, result)

	// Validate tags
	v.validateTags(args.Tags, result)

	// Validate metadata
	v.validateMetadata(args.Metadata, result)

	// Check for blocked patterns
	v.checkBlockedPatterns(args.Content, result)

	// Set overall validity
	result.Valid = len(result.Errors) == 0

	return result
}

// CheckContent performs basic content validation
func (v *WriteArgsValidator) CheckContent(content string) []string {
	var issues []string

	if strings.TrimSpace(content) == "" {
		issues = append(issues, "content cannot be empty")
	}

	if len(content) < v.config.MinContentLength {
		issues = append(issues, fmt.Sprintf("content must be at least %d characters", v.config.MinContentLength))
	}

	if len(content) > v.config.MaxContentLength {
		issues = append(issues, fmt.Sprintf("content exceeds maximum length of %d characters", v.config.MaxContentLength))
	}

	if v.config.ValidateUTF8 && !utf8.ValidString(content) {
		issues = append(issues, "content contains invalid UTF-8 characters")
	}

	return issues
}

// ValidateMetadata validates metadata structure and content
func (v *WriteArgsValidator) ValidateMetadata(metadata map[string]interface{}) []string {
	var issues []string

	if metadata == nil {
		return issues
	}

	if len(metadata) > v.config.MaxMetadataKeys {
		issues = append(issues, fmt.Sprintf("metadata exceeds maximum of %d keys", v.config.MaxMetadataKeys))
	}

	for key, value := range metadata {
		// Validate key
		if key == "" {
			issues = append(issues, "metadata key cannot be empty")
			continue
		}

		if len(key) > 100 {
			issues = append(issues, fmt.Sprintf("metadata key '%s' exceeds maximum length of 100 characters", key))
		}

		// Validate value
		if err := v.validateMetadataValue(key, value); err != nil {
			issues = append(issues, err.Error())
		}
	}

	return issues
}

// SanitizeInput sanitizes and normalizes input arguments
func (v *WriteArgsValidator) SanitizeInput(args WriteArgs) (WriteArgs, error) {
	result := v.ValidateDetailed(args)
	if !result.Valid {
		if len(result.Errors) > 0 {
			return WriteArgs{}, result.Errors[0]
		}
		return WriteArgs{}, fmt.Errorf("validation failed")
	}

	sanitized := result.Sanitized

	// Additional sanitization
	sanitized.Content = v.sanitizeContent(sanitized.Content)
	sanitized.Source = v.sanitizeSource(sanitized.Source)
	sanitized.Tags = v.sanitizeTags(sanitized.Tags)
	sanitized.Metadata = v.sanitizeMetadata(sanitized.Metadata)

	return sanitized, nil
}

// validateContent validates the content field
func (v *WriteArgsValidator) validateContent(content string, result *WriteValidationResult) {
	// Check if content is empty
	if strings.TrimSpace(content) == "" {
		result.Errors = append(result.Errors, WriteValidationError{
			Field:   "content",
			Message: "content cannot be empty",
			Value:   content,
		})
		return
	}

	// Check content length
	if len(content) < v.config.MinContentLength {
		result.Errors = append(result.Errors, WriteValidationError{
			Field:   "content",
			Message: fmt.Sprintf("content must be at least %d characters", v.config.MinContentLength),
			Value:   len(content),
		})
	}

	if len(content) > v.config.MaxContentLength {
		result.Errors = append(result.Errors, WriteValidationError{
			Field:   "content",
			Message: fmt.Sprintf("content exceeds maximum length of %d characters", v.config.MaxContentLength),
			Value:   len(content),
		})
	}

	// Validate UTF-8 if enabled
	if v.config.ValidateUTF8 && !utf8.ValidString(content) {
		result.Errors = append(result.Errors, WriteValidationError{
			Field:   "content",
			Message: "content contains invalid UTF-8 characters",
			Value:   content,
		})
	}

	// Check for suspicious patterns
	if v.containsSuspiciousPatterns(content) {
		result.Warnings = append(result.Warnings, "content contains potentially suspicious patterns")
	}

	// Sanitize content
	result.Sanitized.Content = v.sanitizeContent(content)
}

// validateSource validates the source field
func (v *WriteArgsValidator) validateSource(source string, result *WriteValidationResult) {
	// Check if source is required but empty
	if v.config.RequireSource && strings.TrimSpace(source) == "" {
		result.Errors = append(result.Errors, WriteValidationError{
			Field:   "source",
			Message: "source is required but not provided",
			Value:   source,
		})
		return
	}

	// Check source length
	if len(source) > v.config.MaxSourceLength {
		result.Errors = append(result.Errors, WriteValidationError{
			Field:   "source",
			Message: fmt.Sprintf("source exceeds maximum length of %d characters", v.config.MaxSourceLength),
			Value:   len(source),
		})
	}

	// Validate UTF-8 if enabled
	if v.config.ValidateUTF8 && source != "" && !utf8.ValidString(source) {
		result.Errors = append(result.Errors, WriteValidationError{
			Field:   "source",
			Message: "source contains invalid UTF-8 characters",
			Value:   source,
		})
	}

	// Sanitize source
	result.Sanitized.Source = v.sanitizeSource(source)
}

// validateTags validates the tags field
func (v *WriteArgsValidator) validateTags(tags []string, result *WriteValidationResult) {
	if tags == nil {
		return
	}

	// Check tag count
	if len(tags) > v.config.MaxTagCount {
		result.Errors = append(result.Errors, WriteValidationError{
			Field:   "tags",
			Message: fmt.Sprintf("tags exceed maximum count of %d", v.config.MaxTagCount),
			Value:   len(tags),
		})
	}

	// Validate each tag
	var sanitizedTags []string
	for i, tag := range tags {
		if tag == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("empty tag at index %d will be removed", i))
			continue
		}

		if len(tag) > v.config.MaxTagLength {
			result.Errors = append(result.Errors, WriteValidationError{
				Field:   "tags",
				Message: fmt.Sprintf("tag at index %d exceeds maximum length of %d characters", i, v.config.MaxTagLength),
				Value:   len(tag),
			})
			continue
		}

		if v.config.ValidateUTF8 && !utf8.ValidString(tag) {
			result.Errors = append(result.Errors, WriteValidationError{
				Field:   "tags",
				Message: fmt.Sprintf("tag at index %d contains invalid UTF-8 characters", i),
				Value:   tag,
			})
			continue
		}

		sanitizedTag := v.sanitizeTag(tag)
		if sanitizedTag != "" {
			sanitizedTags = append(sanitizedTags, sanitizedTag)
		}
	}

	result.Sanitized.Tags = sanitizedTags
}

// validateMetadata validates the metadata field
func (v *WriteArgsValidator) validateMetadata(metadata map[string]interface{}, result *WriteValidationResult) {
	if metadata == nil {
		return
	}

	if len(metadata) > v.config.MaxMetadataKeys {
		result.Errors = append(result.Errors, WriteValidationError{
			Field:   "metadata",
			Message: fmt.Sprintf("metadata exceeds maximum of %d keys", v.config.MaxMetadataKeys),
			Value:   len(metadata),
		})
	}

	sanitizedMetadata := make(map[string]interface{})

	for key, value := range metadata {
		// Validate key
		if key == "" {
			result.Warnings = append(result.Warnings, "empty metadata key will be removed")
			continue
		}

		if len(key) > 100 {
			result.Errors = append(result.Errors, WriteValidationError{
				Field:   "metadata",
				Message: fmt.Sprintf("metadata key '%s' exceeds maximum length of 100 characters", key),
				Value:   len(key),
			})
			continue
		}

		// Validate value
		if err := v.validateMetadataValue(key, value); err != nil {
			result.Errors = append(result.Errors, WriteValidationError{
				Field:   "metadata",
				Message: fmt.Sprintf("metadata value for key '%s': %s", key, err.Error()),
				Value:   value,
			})
			continue
		}

		// Sanitize and add to result
		sanitizedValue := v.sanitizeMetadataValue(value)
		if sanitizedValue != nil {
			sanitizedMetadata[key] = sanitizedValue
		}
	}

	result.Sanitized.Metadata = sanitizedMetadata
}

// checkBlockedPatterns checks for blocked patterns in content
func (v *WriteArgsValidator) checkBlockedPatterns(content string, result *WriteValidationResult) {
	for _, pattern := range v.config.BlockedPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue // skip invalid pattern
		}
		if re.MatchString(content) {
			result.Errors = append(result.Errors, WriteValidationError{
				Field:   "content",
				Message: fmt.Sprintf("content contains blocked pattern: %s", pattern),
				Value:   pattern,
			})
		}
	}
}

// sanitizeContent sanitizes the content string
func (v *WriteArgsValidator) sanitizeContent(content string) string {
	// Trim whitespace
	sanitized := strings.TrimSpace(content)

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

// sanitizeSource sanitizes the source string
func (v *WriteArgsValidator) sanitizeSource(source string) string {
	// Trim whitespace
	sanitized := strings.TrimSpace(source)

	// HTML escape if enabled
	if v.config.SanitizeHTML {
		sanitized = html.EscapeString(sanitized)
	}

	// Remove control characters
	sanitized = v.removeControlCharacters(sanitized)

	return sanitized
}

// sanitizeTags sanitizes the tags slice
func (v *WriteArgsValidator) sanitizeTags(tags []string) []string {
	if tags == nil {
		return nil
	}

	var sanitized []string
	for _, tag := range tags {
		sanitizedTag := v.sanitizeTag(tag)
		if sanitizedTag != "" {
			sanitized = append(sanitized, sanitizedTag)
		}
	}

	return sanitized
}

// sanitizeTag sanitizes a single tag
func (v *WriteArgsValidator) sanitizeTag(tag string) string {
	// Trim whitespace
	sanitized := strings.TrimSpace(tag)

	// Convert to lowercase for consistency
	sanitized = strings.ToLower(sanitized)

	// Remove special characters except hyphens and underscores
	re := regexp.MustCompile(`[^a-z0-9\-_]`)
	sanitized = re.ReplaceAllString(sanitized, "")

	// Remove leading/trailing hyphens and underscores
	sanitized = strings.Trim(sanitized, "-_")

	return sanitized
}

// sanitizeMetadata sanitizes the metadata map
func (v *WriteArgsValidator) sanitizeMetadata(metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		return nil
	}

	sanitized := make(map[string]interface{})

	for key, value := range metadata {
		if key != "" {
			sanitizedValue := v.sanitizeMetadataValue(value)
			if sanitizedValue != nil {
				sanitized[key] = sanitizedValue
			}
		}
	}

	return sanitized
}

// sanitizeMetadataValue sanitizes a metadata value
func (v *WriteArgsValidator) sanitizeMetadataValue(value interface{}) interface{} {
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
			if sanitizedItem := v.sanitizeMetadataValue(item); sanitizedItem != nil {
				sanitizedSlice = append(sanitizedSlice, sanitizedItem)
			}
		}
		return sanitizedSlice
	default:
		// For unknown types, convert to string and sanitize
		str := fmt.Sprintf("%v", val)
		return v.sanitizeMetadataValue(str)
	}
}

// validateMetadataValue validates a metadata value
func (v *WriteArgsValidator) validateMetadataValue(key string, value interface{}) error {
	switch val := value.(type) {
	case string:
		if len(val) > v.config.MaxMetadataValueLength {
			return fmt.Errorf("value exceeds maximum length of %d characters", v.config.MaxMetadataValueLength)
		}
		if v.config.ValidateUTF8 && !utf8.ValidString(val) {
			return fmt.Errorf("value contains invalid UTF-8 characters")
		}
	case []interface{}:
		if len(val) > 100 {
			return fmt.Errorf("array value exceeds maximum length of 100 items")
		}
		for i, item := range val {
			if err := v.validateMetadataValue(fmt.Sprintf("%s[%d]", key, i), item); err != nil {
				return err
			}
		}
	case map[string]interface{}:
		if len(val) > 50 {
			return fmt.Errorf("object value exceeds maximum of 50 keys")
		}
		for subKey, subValue := range val {
			if err := v.validateMetadataValue(fmt.Sprintf("%s.%s", key, subKey), subValue); err != nil {
				return err
			}
		}
	default:
		// Fallback: validate stringified representation
		s := fmt.Sprintf("%v", val)
		if len(s) > v.config.MaxMetadataValueLength {
			return fmt.Errorf("value exceeds maximum length of %d characters", v.config.MaxMetadataValueLength)
		}
		if v.config.ValidateUTF8 && !utf8.ValidString(s) {
			return fmt.Errorf("value contains invalid UTF-8 characters")
		}
	}

	return nil
}

// containsSuspiciousPatterns checks for suspicious patterns
func (v *WriteArgsValidator) containsSuspiciousPatterns(content string) bool {
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
		matched, _ := regexp.MatchString(pattern, content)
		if matched {
			return true
		}
	}

	return false
}

// removeControlCharacters removes control characters from a string
func (v *WriteArgsValidator) removeControlCharacters(s string) string {
	// Remove control characters except tab, newline, and carriage return
	re := regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)
	return re.ReplaceAllString(s, "")
}

// normalizeWhitespace normalizes whitespace in a string
func (v *WriteArgsValidator) normalizeWhitespace(s string) string {
	// Replace multiple whitespace characters with single space
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, " ")
}

// GetConfig returns the current configuration
func (v *WriteArgsValidator) GetConfig() *WriteValidatorConfig {
	return v.config
}

// UpdateConfig updates the configuration
func (v *WriteArgsValidator) UpdateConfig(config *WriteValidatorConfig) {
	if config != nil {
		v.config = config
	}
}

// ValidateAndSanitize is a convenience method that validates and sanitizes in one call
func (v *WriteArgsValidator) ValidateAndSanitize(args WriteArgs) (WriteArgs, []string, error) {
	result := v.ValidateDetailed(args)

	if !result.Valid {
		if len(result.Errors) > 0 {
			return WriteArgs{}, result.Warnings, result.Errors[0]
		}
		return WriteArgs{}, result.Warnings, fmt.Errorf("validation failed")
	}

	return result.Sanitized, result.Warnings, nil
}
