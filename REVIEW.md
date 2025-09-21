write_validator_test.go
			Convey("Should reject too many tags", func() {
				manyTags := make([]string, 11)
				for i := range manyTags {
					manyTags[i] = "tag" + string(rune('0'+i))


Copilot AI
13 minutes ago
The string conversion string(rune('0'+i)) will produce incorrect characters for i >= 10. For i=10, this becomes rune(58) which is ':'. Use strconv.Itoa(i) instead for proper numeric string conversion.
Copilot uses AI. Check for mistakes.
—
write_validator_test.go
			Convey("Should return issues for too many keys", func() {
				metadata := make(map[string]interface{})
				for i := 0; i < 21; i++ {
					metadata[string(rune('a'+i))] = "value"


Copilot AI
14 minutes ago
The string conversion string(rune('a'+i)) will produce incorrect characters for i >= 26. For i=26, this becomes rune(123) which is '{'. Use a proper character generation method or limit the loop to i < 26.
Copilot uses AI. Check for mistakes.
—
recall_formatter.go

		// Clean up key names
		cleanKey := strings.ReplaceAll(key, "_", " ")
		cleanKey = strings.Title(cleanKey)


Copilot AI
14 minutes ago
The strings.Title function is deprecated as of Go 1.18 and has incorrect behavior for Unicode. Use cases.Title(language.Und, cases.NoLower).String(cleanKey) from the golang.org/x/text/cases package instead.
Copilot uses AI. Check for mistakes.
—

recall_benchmark_test.go
Comment on lines +296 to +298
	for i := range evidence {
		evidence[i] = Evidence{
			Content:     "Benchmark test evidence content number " + string(rune('0'+i%10)),


Copilot AI
14 minutes ago
The string conversion string(rune('0'+i%10)) is used for numeric conversion. Use strconv.Itoa(i%10) or fmt.Sprintf(\"%d\", i%10) for proper numeric string conversion.
Copilot uses AI. Check for mistakes.
—

file_vector_store.go
			Metadata:  item.Metadata,
		}

		f.vectors[item.ID] = VectorStoreRecord(item)


Copilot AI
15 minutes ago
The type conversion assumes that VectorStoreItem and VectorStoreRecord have the same field structure. This is brittle and could break if either type changes. Consider using explicit field assignment for clarity and safety.
Suggested change
		f.vectors[item.ID] = VectorStoreRecord(item)
		f.vectors[item.ID] = VectorStoreRecord{
			ID:        item.ID,
			Embedding: item.Embedding,
			Metadata:  item.Metadata,
		}

Copilot uses AI. Check for mistakes.
—
entity_resolver.go
Comment on lines +151 to +157
		candidate := Entity{
			ID:         result.ID,
			Name:       fmt.Sprintf("%v", result.Metadata["name"]),
			Type:       fmt.Sprintf("%v", result.Metadata["entity_type"]),
			Confidence: similarity, // Use similarity as confidence
			Properties: result.Metadata,
		}

The use of fmt.Sprintf("%v", ...) to extract values from the metadata map is not type-safe. If result.Metadata["name"] or result.Metadata["entity_type"] is not a string, or is nil, this could lead to unexpected string values like "<nil>" in your Entity struct. It's safer to use type assertions with an ok check to handle these cases gracefully.
		var name, entityType string
		if n, ok := result.Metadata["name"].(string); ok {
			name = n
		}
		if t, ok := result.Metadata["entity_type"].(string); ok {
			entityType = t
		}

		candidate := Entity{
			ID:         result.ID,
			Name:       name,
			Type:       entityType,
			Confidence: similarity, // Use similarity as confidence
			Properties: result.Metadata,
		}

—

entity_extractor.go
Comment on lines +449 to +465
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
@gemini-code-assist gemini-code-assist bot 14 minutes ago
medium

The custom hash implementation in generateEntityID is not standard and could be replaced with a more robust solution from the standard library. Using hash/fnv would be a better choice for non-cryptographic hashing, as it's designed for this purpose and is less prone to collisions or unexpected behavior with different inputs. You will need to import the hash/fnv package.

func (ee *EntityExtractor) generateEntityID(entityType, entityName string) string {
	h := fnv.New64a()
	h.Write([]byte(entityType))
	h.Write([]byte(entityName))
	h.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return fmt.Sprintf("%s_%x", entityType, h.Sum64())
}

---

multi_view_storage.go
Comment on lines 92 to 95
			if updateErr := mvs.graphStore.UpdateNode(timeoutCtx, node); updateErr != nil {
				errors = append(errors, fmt.Errorf("graph store node error: %w", err))
				// If both create and update fail, it's likely a duplicate - just log and continue
				// This is acceptable for testing scenarios
			}
@gemini-code-assist gemini-code-assist bot 14 minutes ago
medium

In StoreChunk, when mvs.graphStore.UpdateNode fails after a CreateNode failure, the error is silently ignored. While the comment suggests this is acceptable for testing, in a production environment, silently ignoring errors can hide underlying issues with the graph store or data consistency. It would be more robust to at least log these failures as warnings, so they can be monitored. You will need to import the log package. This feedback also applies to lines 116-119 for claim nodes.

Suggested change
			if updateErr := mvs.graphStore.UpdateNode(timeoutCtx, node); updateErr != nil {
				errors = append(errors, fmt.Errorf("graph store node error: %w", err))
				// If both create and update fail, it's likely a duplicate - just log and continue
				// This is acceptable for testing scenarios
			}
			if updateErr := mvs.graphStore.UpdateNode(timeoutCtx, node); updateErr != nil {
				// If both create and update fail, it's likely a duplicate. Log as a warning.
				log.Printf("warning: failed to create or update node %s: %v", node.ID, updateErr)
			}

---

recall_benchmark_test.go
	resultCounts := []int{1, 5, 10, 25, 50, 100}

	for _, count := range resultCounts {
		b.Run("Results"+string(rune('0'+count/100))+string(rune('0'+(count%100)/10))+string(rune('0'+count%10)), func(b *testing.B) {
@gemini-code-assist gemini-code-assist bot 14 minutes ago
medium

The construction of the benchmark name is overly complex and hard to read. Using fmt.Sprintf would make the code much clearer and more idiomatic for formatting strings in Go.

Suggested change
		b.Run("Results"+string(rune('0'+count/100))+string(rune('0'+(count%100)/10))+string(rune('0'+count%10)), func(b *testing.B) {
		b.Run(fmt.Sprintf("Results%d", count), func(b *testing.B) {

---

entity_resolver.go
Comment on lines +120 to +126
	// Check if entity has embedding for vector-based similarity
	if embedding, hasEmbedding := entity.Properties["embedding"]; hasEmbedding {
		if embeddingSlice, ok := embedding.([]float32); ok && len(embeddingSlice) > 0 {
			return er.findEntitiesByEmbedding(ctx, embeddingSlice)
		}
	}

@coderabbitai coderabbitai bot 9 minutes ago
⚠️ Potential issue

Embedding type handling is too strict; accept common JSON forms.

entity.Properties["embedding"] will often be []float64 or []any after JSON unmarshaling, so the embedding path will be silently skipped, degrading recall. Convert robustly to []float32 before falling back to name search.

Apply this diff and helper:

- // Check if entity has embedding for vector-based similarity
- if embedding, hasEmbedding := entity.Properties["embedding"]; hasEmbedding {
-   if embeddingSlice, ok := embedding.([]float32); ok && len(embeddingSlice) > 0 {
-     return er.findEntitiesByEmbedding(ctx, embeddingSlice)
-   }
- }
+ // Check if entity has embedding for vector-based similarity
+ if raw, ok := entity.Properties["embedding"]; ok {
+   if embeddingSlice, ok := extractEmbedding(raw); ok && len(embeddingSlice) > 0 {
+     return er.findEntitiesByEmbedding(ctx, embeddingSlice)
+   }
+ }
Add this helper somewhere in this file:

// extractEmbedding converts various decoded JSON forms to []float32.
func extractEmbedding(v interface{}) ([]float32, bool) {
	switch t := v.(type) {
	case []float32:
		return t, len(t) > 0
	case []float64:
		out := make([]float32, len(t))
		for i, f := range t { out[i] = float32(f) }
		return out, len(out) > 0
	case []interface{}:
		out := make([]float32, 0, len(t))
		for _, x := range t {
			switch xv := x.(type) {
			case float64: out = append(out, float32(xv))
			case float32: out = append(out, xv)
			}
		}
		if len(out) == len(t) && len(out) > 0 { return out, true }
	}
	return nil, false
}
🤖 Prompt for AI Agents
In entity_resolver.go around lines 120 to 126, the current embedding check only
accepts []float32 and silently skips other JSON-decoded forms; add the provided
extractEmbedding helper function somewhere in this file and replace the existing
embedding type check to call extractEmbedding(entity.Properties["embedding"])
and, if it returns a non-nil []float32, pass that slice to
er.findEntitiesByEmbedding(ctx, embeddingSlice); ensure the helper handles
[]float32, []float64 and []interface{} (converting to []float32) and only
returns true when a non-empty converted slice is produced.

---

file_vector_store.go
Comment on lines 68 to 87
	var results []VectorResult


	for _, record := range f.vectors {
		// Apply filters
		if !f.matchesFilters(record.Metadata, filters) {
			continue
		}


		// Calculate cosine similarity
		score := f.cosineSimilarity(query, record.Embedding)


		result := VectorResult{
			ID:        record.ID,
			Score:     score,
			Embedding: record.Embedding,
			Metadata:  record.Metadata,
		}


		results = append(results, result)
	}
@coderabbitai coderabbitai bot 9 minutes ago
🧹 Nitpick

Minor: avoid extra allocations and enable early limit in Search

Preallocate results capacity and stop once k items are collected post-sort.

-  var results []VectorResult
+  results := make([]VectorResult, 0, len(f.vectors))
   for _, record := range f.vectors {
     // filters...
     // similarity...
     results = append(results, result)
   }
   // sort desc
   sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
-  // Return top k results
-  if k > 0 && len(results) > k {
-    results = results[:k]
-  }
+  if k > 0 && len(results) > k {
+    results = results[:k]
+  }
   return results, nil
Also applies to: 89-99

🤖 Prompt for AI Agents
In file_vector_store.go around lines 68-87 (and similarly 89-99), the loop over
f.vectors currently appends to results without preallocating capacity and the
search does not stop early; preallocate results with make([]VectorResult, 0,
len(f.vectors)) or with capacity limited by k to avoid extra allocations, and
after computing all scores sort results and then truncate to k (or maintain a
bounded heap during scoring) so you return only the top-k; ensure you break or
slice results to k after sorting to enable the early limit and avoid returning
more items than requested.

---

file_vector_store.go
Comment on lines 274 to 276
	if err := os.WriteFile(f.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
@coderabbitai coderabbitai bot 9 minutes ago
⚠️ Potential issue

Tighten file permissions when persisting vectors

Vectors/metadata may contain sensitive data. Use 0600.

-  if err := os.WriteFile(f.filePath, data, 0644); err != nil {
+  if err := os.WriteFile(f.filePath, data, 0600); err != nil {
📝 Committable suggestion
🤖 Prompt for AI Agents
In file_vector_store.go around lines 274 to 276, the file is being written with
mode 0644 which leaves it world-readable; change the write permission to 0600 so
only the owner can read/write the persisted vectors/metadata (i.e., replace 0644
with 0600 in the os.WriteFile call); if any temp files or directory creation
related to persistence exist, ensure they also use restrictive permissions
consistently.

---

file_vector_store.go
Comment on lines +133 to 134
		f.vectors[item.ID] = VectorStoreRecord(item)
	}
@coderabbitai coderabbitai bot 10 minutes ago
⚠️ Potential issue

Compile error: invalid struct conversion VectorStoreRecord(item)

Go cannot convert between distinct named struct types even if fields are identical. Use a struct literal.

-    f.vectors[item.ID] = VectorStoreRecord(item)
+    f.vectors[item.ID] = VectorStoreRecord{
+      ID:        item.ID,
+      Embedding: item.Embedding,
+      Metadata:  item.Metadata,
+    }
📝 Committable suggestion
🤖 Prompt for AI Agents
In file_vector_store.go around lines 133-134, the code attempts an invalid
conversion using VectorStoreRecord(item); replace this with a struct literal
that constructs a VectorStoreRecord from the fields of item (e.g.,
VectorStoreRecord{ID: item.ID, Embedding: item.Embedding, Metadata:
item.Metadata} or the actual field names/types present) so the compiler sees a
proper composite literal rather than a type conversion.

---

write_validator_test.go
Comment on lines +465 to +483
		Convey("Should handle various blocked patterns", func() {
			blockedContents := []string{
				"<script>alert('xss')</script>",
				"javascript:void(0)",
				"data:text/html,<script>alert(1)</script>",
				"vbscript:msgbox(1)",
			}

			for _, content := range blockedContents {
				args := WriteArgs{
					Content: content,
					Source:  "blocked_test",
				}

				err := validator.Validate(args)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "blocked pattern")
			}
		})
@coderabbitai coderabbitai bot 10 minutes ago
🧹 Nitpick

Add a non-blocking test to avoid ‘data:’ false positives (e.g., ‘metadata:’).

Current blocked-pattern logic (string contains) can flag innocent substrings. After switching to regex with word-boundaries, add a test to ensure “metadata:” passes.

Proposed test (add near other blocked pattern tests):

Convey("Should not block substrings like 'metadata:'", func() {
  args := WriteArgs{ Content: "Discuss metadata: structure and usage", Source: "blocked_test" }
  err := validator.Validate(args)
  So(err, ShouldBeNil)
})
🤖 Prompt for AI Agents
In write_validator_test.go around lines 465-483, add a non-blocking test case to
ensure substrings like "metadata:" are not incorrectly flagged by the
blocked-pattern logic: create a Convey test named something like "Should not
block substrings like 'metadata:'" that constructs WriteArgs with Content
"Discuss metadata: structure and usage" and Source "blocked_test", calls
validator.Validate(args), and asserts So(err, ShouldBeNil); place it alongside
the other blocked pattern tests so it runs in the same suite.

---

write_validator.go
Comment on lines +51 to +66
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
		BlockedPatterns:     []string{`<script`, `javascript:`, `data:`, `vbscript:`},
		SanitizeHTML:        true,
		ValidateUTF8:        true,
		MaxMetadataKeys:     20,
		MaxMetadataValueLength: 500,
	}
@coderabbitai coderabbitai bot 10 minutes ago
🛠️ Refactor suggestion

Tighten default blocked patterns to regex with boundaries.

Using plain substrings (e.g., "data:") causes false positives ("metadata:"). Switch defaults to anchored, case-insensitive regexes.

- BlockedPatterns:     []string{`<script`, `javascript:`, `data:`, `vbscript:`},
+ BlockedPatterns:     []string{
+   `(?i)<script[^>]*>`,
+   `(?i)\bjavascript:`,
+   `(?i)\bdata:(?:text|image|application)/`,
+   `(?i)\bvbscript:`,
+ },
📝 Committable suggestion
🤖 Prompt for AI Agents
In write_validator.go around lines 51 to 66, the BlockedPatterns defaults use
plain substrings causing false positives (e.g., "data:" matching "metadata:");
replace those substrings with anchored, case-insensitive regular expressions
that include word/boundary or tag boundaries as appropriate (for example use
patterns like `(?i)<script\b`, `(?i)javascript:`, `(?i)\bdata:`,
`(?i)vbscript:`) so matches are precise; update the array entries to these regex
strings and ensure any code that treats these values interprets them as regexes
(compile with regexp.Compile or regexp.MustCompile) and preserves existing
behavior like case-insensitive matching and HTML tag boundaries.

---

write_validator.go
Comment on lines +182 to +200
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
@coderabbitai coderabbitai bot 12 minutes ago
⚠️ Potential issue

Double-sanitization bug in SanitizeInput (causes HTML double-escape).

SanitizeInput calls ValidateDetailed (which already sanitizes into result.Sanitized) and then sanitizes again. This yields “&lt;” outputs and can corrupt stored content.

Apply this diff to return the single-pass sanitized args:

 func (v *WriteArgsValidator) SanitizeInput(args WriteArgs) (WriteArgs, error) {
   result := v.ValidateDetailed(args)
   if !result.Valid {
     if len(result.Errors) > 0 {
       return WriteArgs{}, result.Errors[0]
     }
     return WriteArgs{}, fmt.Errorf("validation failed")
   }
-
-  sanitized := result.Sanitized
-
-  // Additional sanitization
-  sanitized.Content = v.sanitizeContent(sanitized.Content)
-  sanitized.Source = v.sanitizeSource(sanitized.Source)
-  sanitized.Tags = v.sanitizeTags(sanitized.Tags)
-  sanitized.Metadata = v.sanitizeMetadata(sanitized.Metadata)
-
-  return sanitized, nil
+  return result.Sanitized, nil
 }
📝 Committable suggestion
🤖 Prompt for AI Agents
In write_validator.go around lines 182 to 200, SanitizeInput currently takes
result.Sanitized from ValidateDetailed and then re-applies sanitization
(sanitizeContent, sanitizeSource, sanitizeTags, sanitizeMetadata), which
double-sanitizes (causing HTML double-escape); remove the second pass by
returning result.Sanitized directly (no additional sanitizeX calls) and
propagate its nil/error semantics as before so SanitizeInput does a single-pass
sanitize and returns sanitized, nil.
@TheApeMachine	Reply...
write_validator.go
Comment on lines +385 to +398
// checkBlockedPatterns checks for blocked patterns in content
func (v *WriteArgsValidator) checkBlockedPatterns(content string, result *WriteValidationResult) {
	lowerContent := strings.ToLower(content)

	for _, pattern := range v.config.BlockedPatterns {
		if strings.Contains(lowerContent, strings.ToLower(pattern)) {
			result.Errors = append(result.Errors, WriteValidationError{
				Field:   "content",
				Message: fmt.Sprintf("content contains blocked pattern: %s", pattern),
				Value:   pattern,
			})
		}
	}
}
@coderabbitai coderabbitai bot 12 minutes ago
🛠️ Refactor suggestion

Use regex matching for blocked patterns to avoid false positives.

Do not lower-case and substring-match; treat patterns as regex and respect boundaries.

-func (v *WriteArgsValidator) checkBlockedPatterns(content string, result *WriteValidationResult) {
-  lowerContent := strings.ToLower(content)
-  
-  for _, pattern := range v.config.BlockedPatterns {
-    if strings.Contains(lowerContent, strings.ToLower(pattern)) {
-      result.Errors = append(result.Errors, WriteValidationError{
-        Field:   "content",
-        Message: fmt.Sprintf("content contains blocked pattern: %s", pattern),
-        Value:   pattern,
-      })
-    }
-  }
-}
+func (v *WriteArgsValidator) checkBlockedPatterns(content string, result *WriteValidationResult) {
+  for _, pattern := range v.config.BlockedPatterns {
+    re, err := regexp.Compile(pattern)
+    if err != nil {
+      continue // skip invalid pattern
+    }
+    if re.MatchString(content) {
+      result.Errors = append(result.Errors, WriteValidationError{
+        Field:   "content",
+        Message: fmt.Sprintf("content contains blocked pattern: %s", pattern),
+        Value:   pattern,
+      })
+    }
+  }
+}
Committable suggestion skipped: line range outside the PR's diff.

🤖 Prompt for AI Agents
In write_validator.go around lines 385 to 398, the current code lowercases and
substring-matches blocked patterns causing false positives; change it to treat
each configured pattern as a regular expression and perform regex matching
(respecting pattern boundaries). For each pattern, compile it as a regexp (use
case-insensitive matching if desired by prefixing with (?i) or by using
regexp.CompilePOSIX options), handle compilation errors (skip or log/append a
validation error), then use re.MatchString(content) against the original
content; when a match is found append the same WriteValidationError with the
pattern as Value. Ensure you do not mutate the original content and properly
handle and surface regex compilation failures.
@TheApeMachine	Reply...
write_validator.go
Comment on lines +517 to +548
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
	}

	return nil
}
@coderabbitai coderabbitai bot 12 minutes ago
⚠️ Potential issue

Unknown metadata types bypass length/UTF-8 checks.

Default branch converts to string in sanitization but ValidateMetadataValue accepts unknown types without constraints. Validate the stringified value for length/UTF-8 here.

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
     ...
   case map[string]interface{}:
     ...
+  default:
+    // Fallback: validate stringified representation
+    s := fmt.Sprintf("%v", val)
+    if len(s) > v.config.MaxMetadataValueLength {
+      return fmt.Errorf("value exceeds maximum length of %d characters", v.config.MaxMetadataValueLength)
+    }
+    if v.config.ValidateUTF8 && !utf8.ValidString(s) {
+      return fmt.Errorf("value contains invalid UTF-8 characters")
+    }
   }
   return nil
 }
📝 Committable suggestion
🤖 Prompt for AI Agents
In write_validator.go around lines 517 to 548, the validator currently skips
unknown metadata value types which allows oversized or non-UTF8 stringifiable
values; update validateMetadataValue to handle the default case by converting
unknown types to their string representation (e.g., fmt.Sprint(value)) and then
apply the same MaxMetadataValueLength and ValidateUTF8 checks as for string
values, returning the same formatted errors if limits are exceeded or UTF-8 is
invalid; keep existing handling for string, []interface{}, and
map[string]interface{} unchanged.
@TheApeMachine	Reply...
coderabbitai[bot]
coderabbitai bot requested changes 11 minutes ago
coderabbitai bot left a comment
Review continued from previous batch...

entity_resolver.go
Comment on lines +131 to +163
// findEntitiesByEmbedding searches for entities using vector similarity
func (er *EntityResolver) findEntitiesByEmbedding(ctx context.Context, embedding []float32) ([]Entity, error) {
	// Search for similar entity embeddings using vector store
	vectorResults, err := er.storage.vectorStore.Search(ctx, embedding, er.config.MaxCandidates, map[string]interface{}{
		"type": "entity",
	})
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	var candidates []Entity
	for _, result := range vectorResults {
		// Calculate cosine similarity using our method
		similarity := er.calculateCosineSimilarity(embedding, result.Embedding)

		// Skip entities below similarity threshold
		if similarity < er.config.SimilarityThreshold {
			continue
		}

		candidate := Entity{
			ID:         result.ID,
			Name:       fmt.Sprintf("%v", result.Metadata["name"]),
			Type:       fmt.Sprintf("%v", result.Metadata["entity_type"]),
			Confidence: similarity, // Use similarity as confidence
			Properties: result.Metadata,
		}

		candidates = append(candidates, candidate)
	}

	return candidates, nil
}
@coderabbitai coderabbitai bot 11 minutes ago
⚠️ Potential issue

Vector search: strengthen filtering, dimension checks, and carry embeddings forward.

Skip candidates with mismatched dimensions to avoid spurious 0.0 similarities.
Enforce the "type == entity" filter defensively (don’t rely solely on store).
Persist result.Embedding into candidate.Properties to enable later vector-aware weighting/linking.
Apply this diff:

 func (er *EntityResolver) findEntitiesByEmbedding(ctx context.Context, embedding []float32) ([]Entity, error) {
   // Search for similar entity embeddings using vector store
   vectorResults, err := er.storage.vectorStore.Search(ctx, embedding, er.config.MaxCandidates, map[string]interface{}{
     "type": "entity",
   })
   if err != nil {
     return nil, fmt.Errorf("vector search failed: %w", err)
   }

   var candidates []Entity
   for _, result := range vectorResults {
-    // Calculate cosine similarity using our method
-    similarity := er.calculateCosineSimilarity(embedding, result.Embedding)
+    // Enforce index/category filter defensively
+    if fmt.Sprintf("%v", result.Metadata["type"]) != "entity" {
+      continue
+    }
+    // Dimension guard
+    if len(result.Embedding) == 0 || len(result.Embedding) != len(embedding) {
+      continue
+    }
+    // Calculate cosine similarity
+    similarity := er.calculateCosineSimilarity(embedding, result.Embedding)

     // Skip entities below similarity threshold
     if similarity < er.config.SimilarityThreshold {
       continue
     }

-    candidate := Entity{
+    // Carry embedding forward for downstream decisions
+    props := result.Metadata
+    if props == nil {
+      props = map[string]interface{}{}
+    }
+    props["embedding"] = result.Embedding
+    candidate := Entity{
       ID:         result.ID,
       Name:       fmt.Sprintf("%v", result.Metadata["name"]),
       Type:       fmt.Sprintf("%v", result.Metadata["entity_type"]),
       Confidence: similarity, // Use similarity as confidence
-      Properties: result.Metadata,
+      Properties: props,
     }

     candidates = append(candidates, candidate)
   }

   return candidates, nil
 }
📝 Committable suggestion
🤖 Prompt for AI Agents
In entity_resolver.go around lines 131 to 163, strengthen vector-search
post-filtering: defensively enforce the "type" metadata equals "entity" before
accepting a result, skip any result whose embedding is nil or whose
dimensionality doesn't match the input embedding to avoid spurious 0.0
similarities, and persist the result.Embedding into the candidate.Properties
(under a clear key like "embedding") so downstream logic can use the original
vector; compute similarity only after passing these checks and only then append
the candidate. Ensure nil-checks and length comparisons are used for dimension
validation and do not change the external function signature.
@TheApeMachine	Reply...
entity_resolver.go
Comment on lines 345 to 361
func (er *EntityResolver) createEntityLink(ctx context.Context, entity1, entity2 Entity) error {
	edge := &Edge{
		ID:   fmt.Sprintf("link_%s_%s", entity1.ID, entity2.ID),
		From: entity1.ID,
		To:   entity2.ID,
		Type: RelatedTo,
		ID:     fmt.Sprintf("link_%s_%s", entity1.ID, entity2.ID),
		From:   entity1.ID,
		To:     entity2.ID,
		Type:   RelatedTo,
		Weight: er.calculateSimilarity(entity1, entity2),
		Properties: map[string]interface{}{
			"link_type": "entity_resolution",
			"link_type":  "entity_resolution",
			"created_by": "entity_resolver",
		},
		CreatedAt: time.Now(),
	}

	return er.storage.graphStore.CreateEdge(ctx, edge)
} No newline at end of file
}
@coderabbitai coderabbitai bot 11 minutes ago
⚠️ Potential issue

Edge weight should reflect vector similarity when available; also consider idempotent edge IDs.

Compute Weight from cosine similarity if both entities carry embeddings; fallback to name/type similarity.
The current edge ID is order-dependent; consider a stable, order‑insensitive ID to avoid duplicate edges across runs.
Apply this diff for weight:

 func (er *EntityResolver) createEntityLink(ctx context.Context, entity1, entity2 Entity) error {
-  edge := &Edge{
-    ID:     fmt.Sprintf("link_%s_%s", entity1.ID, entity2.ID),
-    From:   entity1.ID,
-    To:     entity2.ID,
-    Type:   RelatedTo,
-    Weight: er.calculateSimilarity(entity1, entity2),
+  // Compute weight with vector similarity when possible
+  weight := er.calculateSimilarity(entity1, entity2)
+  if v1, ok1 := extractEmbedding(entity1.Properties["embedding"]); ok1 {
+    if v2, ok2 := extractEmbedding(entity2.Properties["embedding"]); ok2 && len(v1) == len(v2) {
+      weight = er.calculateCosineSimilarity(v1, v2)
+    }
+  }
+  edge := &Edge{
+    ID:     fmt.Sprintf("link_%s_%s", entity1.ID, entity2.ID),
+    From:   entity1.ID,
+    To:     entity2.ID,
+    Type:   RelatedTo,
+    Weight: weight,
     Properties: map[string]interface{}{
       "link_type":  "entity_resolution",
       "created_by": "entity_resolver",
     },
     CreatedAt: time.Now(),
   }
For ID stability, consider hashing the unordered pair (suggested change, not provided as diff to avoid broader semantics).

Committable suggestion skipped: line range outside the PR's diff.

@TheApeMachine	Reply...
handlers_test.go
Comment on lines 31 to 37
			Convey("Then it should return successfully", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
				So(recallResult.Evidence, ShouldNotBeEmpty)
				So(len(recallResult.Evidence), ShouldEqual, 1)
				So(recallResult.Evidence[0].Content, ShouldContainSubstring, "test query")
				So(recallResult.Stats.TotalCandidates, ShouldEqual, 1)
				So(recallResult.Evidence, ShouldNotBeNil) // Can be empty with mock storage
				So(recallResult.Stats, ShouldNotBeNil)
				So(recallResult.Stats.TotalCandidates, ShouldBeGreaterThanOrEqualTo, 0)
			})
@coderabbitai coderabbitai bot 11 minutes ago
🧹 Nitpick

Strengthen recall assertions to catch formatting regressions.

Also assert MCP content is a single TextContent before accessing fields in other tests.

Apply this diff in this test block:

 So(err, ShouldBeNil)
 So(result, ShouldNotBeNil)
 So(recallResult.Evidence, ShouldNotBeNil) // Can be empty with mock storage
 So(recallResult.Stats, ShouldNotBeNil)
 So(recallResult.Stats.TotalCandidates, ShouldBeGreaterThanOrEqualTo, 0)
+// Optional: verify MCP text content shape here too if available
+// mcpResult, _, _ := server.handleRecall(ctx, req, args)
+// So(mcpResult.Content, ShouldHaveLength, 1)
+// _, ok := mcpResult.Content[0].(*mcp.TextContent)
+// So(ok, ShouldBeTrue)
📝 Committable suggestion
🤖 Prompt for AI Agents
In handlers_test.go around lines 31-37, strengthen the assertions by checking
that recallResult.Recall is not nil and that its Content slice exists and
contains exactly one element to catch formatting regressions (e.g.,
So(recallResult.Recall, ShouldNotBeNil); So(len(recallResult.Recall.Content),
ShouldBeGreaterThan, 0) or equal to 1). Additionally, in tests that access MCP
fields, assert the MCP content is a single TextContent instance before using its
fields (e.g., verify the single content element exists and is of TextContent
type) so downstream field access is safe.
@TheApeMachine	Reply...
memory_writer_integration_test.go
Comment on lines +120 to 137
				// EntitiesLinked may be nil or empty with mock data
				if result.EntitiesLinked != nil {
					So(len(result.EntitiesLinked), ShouldBeGreaterThanOrEqualTo, 0)
				}

				// Verify entity linking created edges (if entities were linked)
				if len(result.EntitiesLinked) > 0 {
					mockGraphStore := storage.graphStore.(*MockGraphStore)
					edges := mockGraphStore.GetEdges()
					hasRelationEdge := false
					for _, edge := range edges {
						if edge.Type == RelatedTo {
							hasRelationEdge = true
							break
						}
					}
					So(hasRelationEdge, ShouldBeTrue)
				}
@coderabbitai coderabbitai bot 11 minutes ago
🧹 Nitpick

Simplify redundant nil/length checks on EntitiesLinked.

len(nil) is valid and returns 0, so the initial nil guard is unnecessary.

- // EntitiesLinked may be nil or empty with mock data
- if result.EntitiesLinked != nil {
-   So(len(result.EntitiesLinked), ShouldBeGreaterThanOrEqualTo, 0)
- }
+ // EntitiesLinked may be nil or empty with mock data
+ So(len(result.EntitiesLinked), ShouldBeGreaterThanOrEqualTo, 0)

- // Verify entity linking created edges (if entities were linked)
- if len(result.EntitiesLinked) > 0 {
+ // Verify entity linking created edges (if entities were linked)
+ if len(result.EntitiesLinked) > 0 {
    ...
   So(hasRelationEdge, ShouldBeTrue)
 }
📝 Committable suggestion
🤖 Prompt for AI Agents
memory_writer_integration_test.go around lines 120 to 137: the initial nil guard
for result.EntitiesLinked is redundant because len(nil) == 0; remove the if
result.EntitiesLinked != nil block and directly assert on length
(So(len(result.EntitiesLinked), ShouldBeGreaterThanOrEqualTo, 0)), and keep the
subsequent conditional that checks if len(result.EntitiesLinked) > 0 before
inspecting mockGraphStore edges.
@TheApeMachine	Reply...
recall_e2e_test.go
Comment on lines +120 to +123
					// Verify query is reflected in the response
					textContent := mcpResult.Content[0].(*mcp.TextContent)
					So(textContent.Text, ShouldContainSubstring, tc.query)
				})
@coderabbitai coderabbitai bot 11 minutes ago
⚠️ Potential issue

Avoid panic: guard type assertion to TextContent.

Directly asserting mcpResult.Content[0].(*mcp.TextContent) can panic if format changes.

Apply this diff:

- textContent := mcpResult.Content[0].(*mcp.TextContent)
- So(textContent.Text, ShouldContainSubstring, tc.query)
+ So(mcpResult.Content, ShouldHaveLength, 1)
+ textContent, ok := mcpResult.Content[0].(*mcp.TextContent)
+ So(ok, ShouldBeTrue)
+ So(textContent.Text, ShouldContainSubstring, tc.query)
📝 Committable suggestion
🤖 Prompt for AI Agents
In recall_e2e_test.go around lines 120 to 123, the test does an unchecked type
assertion on mcpResult.Content[0] to *mcp.TextContent which can panic if the
slice is empty or the element type changes; fix by first ensuring the Content
slice has at least one element, then perform a safe type assertion (value, ok)
or a type switch to confirm it’s a *mcp.TextContent, and if not, fail the test
with a clear error message instead of letting the test panic; once confirmed,
use the asserted value to verify the Text contains tc.query.
@TheApeMachine	Reply...
11 hidden conversations
Load more…
recall_handler.go
Comment on lines +104 to +117
	mcpResult := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Retrieved %d pieces of evidence for query: %s",
					len(result.Evidence), args.Query),
			},
		},
	}

	log.Printf("Recall completed in %v, returned %d evidence items",
		time.Since(startTime), len(result.Evidence))

	return mcpResult, result, nil
}
@coderabbitai coderabbitai bot 11 minutes ago
🧹 Nitpick

MCP response only returns a text summary. Consider including structured payload.

Add an additional JSON content part (if MCP client supports it) with the serialized RecallResult for richer clients, while keeping the text summary for human readability.

🤖 Prompt for AI Agents
In recall_handler.go around lines 104 to 117, the MCP response currently
contains only a human-readable TextContent summary; add a second Content entry
that includes the serialized RecallResult as structured JSON so richer clients
can consume the full payload. Marshal the result (e.g., json.Marshal(result))
and append an appropriate JSON content type (e.g., mcp.JsonContent or
equivalent) to mcpResult.Content, handle any marshal error by logging and
returning an error or falling back to the text-only response, and preserve the
existing TextContent for readability.

recall_handler.go
Comment on lines +303 to +305
		for _, method := range result.SourceMethods {
			relationMap[method] = fmt.Sprintf("score_%.3f", result.MethodScores[method])
		}
@coderabbitai coderabbitai bot 11 minutes ago
🧹 Nitpick

RelationMap values are prefixed with "score_"; prefer plain numeric strings for easier consumption.

- relationMap[method] = fmt.Sprintf("score_%.3f", result.MethodScores[method])
+ relationMap[method] = fmt.Sprintf("%.3f", result.MethodScores[method])
📝 Committable suggestion
🤖 Prompt for AI Agents
In recall_handler.go around lines 303 to 305, the relationMap entries are being
set with a "score_" prefix (fmt.Sprintf("score_%.3f",
result.MethodScores[method])); remove the "score_" prefix so values are plain
numeric strings (e.g., fmt.Sprintf("%.3f", result.MethodScores[method])) and
keep the same precision/formatting for each method in result.SourceMethods.

recall_handler.go
Comment on lines +387 to +406
	for i := 0; i < len(evidence); i++ {
		for j := i + 1; j < len(evidence); j++ {
			if evidence[i].Source != evidence[j].Source {
				confidenceDiff := evidence[i].Confidence - evidence[j].Confidence
				if confidenceDiff > 0.5 || confidenceDiff < -0.5 {
					conflicts = append(conflicts, ConflictInfo{
						ID:   fmt.Sprintf("conflict_%d_%d", i, j),
						Type: "confidence_mismatch",
						Description: fmt.Sprintf("Significant confidence difference between sources %s and %s",
							evidence[i].Source, evidence[j].Source),
						ConflictingIDs: []string{fmt.Sprintf("evidence_%d", i), fmt.Sprintf("evidence_%d", j)},
						Severity:       "medium",
					})
				}
			}
		}
	}

	return conflicts
}
@coderabbitai coderabbitai bot 11 minutes ago
⚠️ Potential issue

Conflict detection is O(n²). Cap the pairwise comparisons for large result sets.

Limit to top-K evidence (by confidence) or early‑exit once a threshold is met.
Example:

const maxPairs = 5000 // ~100 items => 4950 pairs
count := 0
for i := 0; i < len(evidence) && count < maxPairs; i++ {
  for j := i + 1; j < len(evidence) && count < maxPairs; j++ {
    // existing logic
    count++
  }
}
🤖 Prompt for AI Agents
In recall_handler.go around lines 387 to 406 the pairwise conflict detection
loops are O(n²) and must be capped for large evidence sets; limit work by (a)
sorting evidence by Confidence descending and truncating to a top-K (e.g. 100)
before the nested loops OR (b) adding a maxPairs counter and short-circuiting
both loops when reached (e.g. maxPairs = 5000). Implement one of these
approaches: if using top-K, sort and slice evidence to evidence =
evidence[:min(len(evidence), K)] before comparing; if using maxPairs, increment
a counter inside the inner loop and break both loops when count >= maxPairs.
Ensure behavior and return type unchanged and update any related tests.

recall_validator_test.go
Comment on lines +333 to +339
			sanitized, err := validator.SanitizeInput(args)

			Convey("Then defaults should be applied", func() {
				So(err, ShouldBeNil)
				So(sanitized.MaxResults, ShouldEqual, 10)    // default
				So(sanitized.TimeBudget, ShouldEqual, 5000)  // default
			})
@coderabbitai coderabbitai bot 11 minutes ago
🧹 Nitpick

Defaults coupling: keep validator defaults consistent with handler.

Test asserts default MaxResults=10 and TimeBudget=5000. Ensure these come from a single source of truth (shared constants/config) to avoid drift between validator and handler.

🤖 Prompt for AI Agents
In recall_validator_test.go around lines 333 to 339, the test hardcodes defaults
(MaxResults=10, TimeBudget=5000) causing potential drift; update the test to
reference the shared source of truth instead of literals—import the constants or
config values used by the handler (or the package where defaults are defined)
and assert sanitized.MaxResults equals that constant and sanitized.TimeBudget
equals that constant; if such shared constants do not yet exist, create them in
a common package (e.g., config or defaults) and update both the validator,
handler, and this test to use them.

recall_validator_test.go
Comment on lines +356 to +363

		Convey("When sanitizing query with HTML", func() {
			query := "<script>alert('xss')</script>"
			sanitized := validator.sanitizeQuery(query)

			Convey("Then HTML should be escaped", func() {
				So(sanitized, ShouldEqual, "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;")
			})
@coderabbitai coderabbitai bot 11 minutes ago
🧹 Nitpick

Add UTF‑8 validation coverage.

Given ValidateUTF8=true by default, add a case with invalid UTF‑8 input to ensure normalization/rejection is exercised.

I can add a Convey block generating invalid UTF‑8 (e.g., []byte{0xff}) and asserting Validate/ValidateDetailed behavior if you want me to push a patch.

🤖 Prompt for AI Agents
In recall_validator_test.go around lines 356 to 363, add a new Convey block that
injects an invalid UTF-8 input (for example from []byte{0xff} converted to
string) and calls validator.Validate and validator.ValidateDetailed; because
ValidateUTF8=true by default, assert that Validate returns false (or an
error/failure) and that ValidateDetailed reports an UTF-8 validation failure
(i.e., the detailed result/error contains "UTF-8" or "invalid UTF-8"); keep the
test isolated and follow existing Convey style and assertions.

