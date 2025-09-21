package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// WriteResponseFormatter formats write responses for MCP output
type WriteResponseFormatter struct {
	config *WriteFormatterConfig
}

// WriteFormatterConfig holds configuration for response formatting
type WriteFormatterConfig struct {
	IncludeMetadata         bool `json:"include_metadata"`
	IncludeDebugInfo        bool `json:"include_debug_info"`
	FormatTimestamps        bool `json:"format_timestamps"`
	TruncateIDs             bool `json:"truncate_ids"`
	MaxIDLength             int  `json:"max_id_length"`
	SortConflictsBySeverity bool `json:"sort_conflicts_by_severity"`
	IncludeProcessingStats  bool `json:"include_processing_stats"`
	ShowWarnings            bool `json:"show_warnings"`
}

// WriteFormattingContext provides context for formatting decisions
type WriteFormattingContext struct {
	OriginalContent string                 `json:"original_content"`
	RequestTime     time.Time              `json:"request_time"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	UserPreferences map[string]interface{} `json:"user_preferences"`
}

// NewWriteResponseFormatter creates a new formatter with default configuration
func NewWriteResponseFormatter() *WriteResponseFormatter {
	config := &WriteFormatterConfig{
		IncludeMetadata:         true,
		IncludeDebugInfo:        false,
		FormatTimestamps:        true,
		TruncateIDs:             true,
		MaxIDLength:             16,
		SortConflictsBySeverity: true,
		IncludeProcessingStats:  true,
		ShowWarnings:            true,
	}

	return &WriteResponseFormatter{
		config: config,
	}
}

// NewWriteResponseFormatterWithConfig creates a formatter with custom configuration
func NewWriteResponseFormatterWithConfig(config *WriteFormatterConfig) *WriteResponseFormatter {
	if config == nil {
		return NewWriteResponseFormatter()
	}

	return &WriteResponseFormatter{
		config: config,
	}
}

// Format formats a WriteResponse into a WriteResult for MCP output
func (f *WriteResponseFormatter) Format(response *WriteResponse, args WriteArgs) (WriteResult, error) {
	if response == nil {
		return WriteResult{}, fmt.Errorf("response cannot be nil")
	}

	context := &WriteFormattingContext{
		OriginalContent: args.Content,
		RequestTime:     time.Now(),
		ProcessingTime:  response.ProcessingTime,
	}

	// Format memory ID
	formattedMemoryID := f.formatMemoryID(response.MemoryID)

	// Format entities linked
	formattedEntities := f.formatEntitiesLinked(response.EntitiesLinked, context)

	// Format conflicts
	formattedConflicts := f.formatConflicts(response.ConflictsFound, context)

	// Format provenance ID
	formattedProvenanceID := f.formatProvenanceID(response.ProvenanceID)

	result := WriteResult{
		MemoryID:       formattedMemoryID,
		CandidateCount: response.CandidateCount,
		ConflictsFound: formattedConflicts,
		EntitiesLinked: formattedEntities,
		ProvenanceID:   formattedProvenanceID,
	}

	return result, nil
}

// BuildWriteResult builds a WriteResult from individual components
func (f *WriteResponseFormatter) BuildWriteResult(memoryID string, candidateCount int, entitiesLinked []string, provenanceID string) WriteResult {
	context := &WriteFormattingContext{
		RequestTime: time.Now(),
	}

	// Format each component
	formattedMemoryID := f.formatMemoryID(memoryID)
	formattedEntities := f.formatEntitiesLinked(entitiesLinked, context)
	formattedProvenanceID := f.formatProvenanceID(provenanceID)

	return WriteResult{
		MemoryID:       formattedMemoryID,
		CandidateCount: candidateCount,
		ConflictsFound: []ConflictInfo{},
		EntitiesLinked: formattedEntities,
		ProvenanceID:   formattedProvenanceID,
	}
}

// AddConflictInfo adds conflict information to the formatted result
func (f *WriteResponseFormatter) AddConflictInfo(result *WriteResult, conflicts []ConflictInfo) {
	if conflicts == nil {
		return
	}

	context := &WriteFormattingContext{
		RequestTime: time.Now(),
	}

	formattedConflicts := f.formatConflicts(conflicts, context)
	result.ConflictsFound = formattedConflicts
}

// formatMemoryID formats the memory ID
func (f *WriteResponseFormatter) formatMemoryID(memoryID string) string {
	if memoryID == "" {
		return ""
	}

	// Truncate ID if configured
	if f.config.TruncateIDs && len(memoryID) > f.config.MaxIDLength {
		return memoryID[:f.config.MaxIDLength] + "..."
	}

	return memoryID
}

// formatEntitiesLinked formats the entities linked list
func (f *WriteResponseFormatter) formatEntitiesLinked(entities []string, _ *WriteFormattingContext) []string {
	if len(entities) == 0 {
		return []string{}
	}

	var formatted []string
	for _, entity := range entities {
		formattedEntity := f.formatEntityID(entity)
		if formattedEntity != "" {
			formatted = append(formatted, formattedEntity)
		}
	}

	// Remove duplicates
	formatted = f.removeDuplicateStrings(formatted)

	// Sort alphabetically for consistency
	sort.Strings(formatted)

	return formatted
}

// formatEntityID formats a single entity ID
func (f *WriteResponseFormatter) formatEntityID(entityID string) string {
	if entityID == "" {
		return ""
	}

	// Truncate ID if configured
	if f.config.TruncateIDs && len(entityID) > f.config.MaxIDLength {
		return entityID[:f.config.MaxIDLength] + "..."
	}

	return entityID
}

// formatConflicts formats conflict information
func (f *WriteResponseFormatter) formatConflicts(conflicts []ConflictInfo, context *WriteFormattingContext) []ConflictInfo {
	if len(conflicts) == 0 {
		return []ConflictInfo{}
	}

	formatted := make([]ConflictInfo, len(conflicts))
	for i, conflict := range conflicts {
		formatted[i] = f.formatConflict(conflict, context)
	}

	// Sort by severity if configured
	if f.config.SortConflictsBySeverity {
		f.sortConflictsBySeverity(formatted)
	}

	return formatted
}

// formatConflict formats a single conflict
func (f *WriteResponseFormatter) formatConflict(conflict ConflictInfo, context *WriteFormattingContext) ConflictInfo {
	formatted := conflict

	// Ensure ID is not empty
	if formatted.ID == "" {
		formatted.ID = fmt.Sprintf("conflict_%d", time.Now().UnixNano())
	}

	// Truncate ID if configured
	if f.config.TruncateIDs && len(formatted.ID) > f.config.MaxIDLength {
		formatted.ID = formatted.ID[:f.config.MaxIDLength] + "..."
	}

	// Enhance description with context
	if context.OriginalContent != "" && len(context.OriginalContent) > 0 {
		contentPreview := f.truncateContent(context.OriginalContent, 50)
		formatted.Description = fmt.Sprintf("%s (content: \"%s\")",
			conflict.Description, contentPreview)
	}

	// Format conflicting IDs
	if len(formatted.ConflictingIDs) > 0 {
		formattedIDs := make([]string, len(formatted.ConflictingIDs))
		for i, id := range formatted.ConflictingIDs {
			formattedIDs[i] = f.formatEntityID(id)
		}
		formatted.ConflictingIDs = formattedIDs
	}

	// Normalize severity
	formatted.Severity = f.normalizeSeverity(formatted.Severity)

	return formatted
}

// formatProvenanceID formats the provenance ID
func (f *WriteResponseFormatter) formatProvenanceID(provenanceID string) string {
	if provenanceID == "" {
		return ""
	}

	// Truncate ID if configured
	if f.config.TruncateIDs && len(provenanceID) > f.config.MaxIDLength {
		return provenanceID[:f.config.MaxIDLength] + "..."
	}

	return provenanceID
}

// sortConflictsBySeverity sorts conflicts by severity level
func (f *WriteResponseFormatter) sortConflictsBySeverity(conflicts []ConflictInfo) {
	sort.Slice(conflicts, func(i, j int) bool {
		return f.getSeverityWeight(conflicts[i].Severity) > f.getSeverityWeight(conflicts[j].Severity)
	})
}

// getSeverityWeight returns a numeric weight for severity levels
func (f *WriteResponseFormatter) getSeverityWeight(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

// normalizeSeverity normalizes severity strings
func (f *WriteResponseFormatter) normalizeSeverity(severity string) string {
	normalized := strings.ToLower(strings.TrimSpace(severity))

	switch normalized {
	case "critical", "high", "medium", "low":
		return normalized
	case "error", "severe":
		return "high"
	case "warning", "warn":
		return "medium"
	case "info", "information":
		return "low"
	default:
		return "medium" // Default to medium if unknown
	}
}

// truncateContent truncates content for display purposes
func (f *WriteResponseFormatter) truncateContent(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength-3] + "..."
}

// removeDuplicateStrings removes duplicate strings from a slice
func (f *WriteResponseFormatter) removeDuplicateStrings(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// GetConfig returns the current configuration
func (f *WriteResponseFormatter) GetConfig() *WriteFormatterConfig {
	return f.config
}

// UpdateConfig updates the configuration
func (f *WriteResponseFormatter) UpdateConfig(config *WriteFormatterConfig) {
	if config != nil {
		f.config = config
	}
}

// FormatForDisplay formats a WriteResult for human-readable display
func (f *WriteResponseFormatter) FormatForDisplay(result WriteResult) string {
	var builder strings.Builder

	builder.WriteString("=== Memory Write Results ===\n")
	builder.WriteString(fmt.Sprintf("Memory ID: %s\n", result.MemoryID))
	builder.WriteString(fmt.Sprintf("Candidates Created: %d\n", result.CandidateCount))
	builder.WriteString(fmt.Sprintf("Entities Linked: %d\n", len(result.EntitiesLinked)))
	builder.WriteString(fmt.Sprintf("Provenance ID: %s\n\n", result.ProvenanceID))

	// Format entities
	if len(result.EntitiesLinked) > 0 {
		builder.WriteString("=== Linked Entities ===\n")
		for i, entity := range result.EntitiesLinked {
			builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, entity))
		}
		builder.WriteString("\n")
	}

	// Format conflicts if any
	if len(result.ConflictsFound) > 0 {
		builder.WriteString("=== Conflicts Detected ===\n")
		for _, conflict := range result.ConflictsFound {
			builder.WriteString(fmt.Sprintf("- %s (%s): %s\n",
				conflict.Type, conflict.Severity, conflict.Description))
			if len(conflict.ConflictingIDs) > 0 {
				builder.WriteString(fmt.Sprintf("  Conflicting IDs: %s\n",
					strings.Join(conflict.ConflictingIDs, ", ")))
			}
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// FormatSummary creates a brief summary of the write operation
func (f *WriteResponseFormatter) FormatSummary(result WriteResult) string {
	summary := fmt.Sprintf("Memory stored (ID: %s)", result.MemoryID)

	if result.CandidateCount > 1 {
		summary += fmt.Sprintf(", %d candidates created", result.CandidateCount)
	}

	if len(result.EntitiesLinked) > 0 {
		summary += fmt.Sprintf(", %d entities linked", len(result.EntitiesLinked))
	}

	if len(result.ConflictsFound) > 0 {
		summary += fmt.Sprintf(", %d conflicts detected", len(result.ConflictsFound))
	}

	return summary
}

// ValidateResult validates a WriteResult for consistency
func (f *WriteResponseFormatter) ValidateResult(result WriteResult) error {
	if result.MemoryID == "" {
		return fmt.Errorf("memory ID cannot be empty")
	}

	if result.CandidateCount < 0 {
		return fmt.Errorf("candidate count cannot be negative")
	}

	if result.EntitiesLinked == nil {
		return fmt.Errorf("entities linked cannot be nil")
	}

	if result.ConflictsFound == nil {
		return fmt.Errorf("conflicts found cannot be nil")
	}

	// Validate conflicts
	for i, conflict := range result.ConflictsFound {
		if conflict.ID == "" {
			return fmt.Errorf("conflict at index %d has empty ID", i)
		}
		if conflict.Type == "" {
			return fmt.Errorf("conflict at index %d has empty type", i)
		}
		if conflict.Severity == "" {
			return fmt.Errorf("conflict at index %d has empty severity", i)
		}
	}

	return nil
}

// AddProcessingMetadata adds processing metadata to the result
func (f *WriteResponseFormatter) AddProcessingMetadata(result *WriteResult, metadata map[string]interface{}) {
	if !f.config.IncludeMetadata || metadata == nil {
		return
	}

	// This could be extended to add metadata to the result structure
	// For now, we'll log it if debug info is enabled
	if f.config.IncludeDebugInfo {
		fmt.Printf("Processing metadata: %+v\n", metadata)
	}
}

// CreateConflictInfo creates a ConflictInfo structure
func (f *WriteResponseFormatter) CreateConflictInfo(id, conflictType, description, severity string, conflictingIDs []string) ConflictInfo {
	return ConflictInfo{
		ID:             id,
		Type:           conflictType,
		Description:    description,
		ConflictingIDs: conflictingIDs,
		Severity:       f.normalizeSeverity(severity),
	}
}

// MergeConflicts merges multiple conflict lists
func (f *WriteResponseFormatter) MergeConflicts(conflictLists ...[]ConflictInfo) []ConflictInfo {
	var merged []ConflictInfo
	seen := make(map[string]bool)

	for _, conflicts := range conflictLists {
		for _, conflict := range conflicts {
			if !seen[conflict.ID] {
				seen[conflict.ID] = true
				merged = append(merged, conflict)
			}
		}
	}

	// Sort the merged conflicts
	if f.config.SortConflictsBySeverity {
		f.sortConflictsBySeverity(merged)
	}

	return merged
}
