package main

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// EntityResolver handles entity deduplication and linking
type EntityResolver struct {
	storage           *MultiViewStorage
	similarityThreshold float64
	config            *EntityResolverConfig
}

// EntityResolverConfig contains configuration for entity resolution
type EntityResolverConfig struct {
	SimilarityThreshold    float64
	EnableFuzzyMatching   bool
	MaxCandidates         int
	MinConfidenceBoost    float64
}

// NewEntityResolver creates a new EntityResolver instance
func NewEntityResolver(storage *MultiViewStorage) *EntityResolver {
	config := &EntityResolverConfig{
		SimilarityThreshold:   0.8,
		EnableFuzzyMatching:  true,
		MaxCandidates:        10,
		MinConfidenceBoost:   0.1,
	}

	return &EntityResolver{
		storage:             storage,
		similarityThreshold: config.SimilarityThreshold,
		config:             config,
	}
}

// Resolve resolves a list of entities by deduplicating and linking to existing entities
func (er *EntityResolver) Resolve(ctx context.Context, entities []Entity) ([]Entity, error) {
	var resolvedEntities []Entity

	for _, entity := range entities {
		// First deduplicate within the current batch
		deduplicated := er.Deduplicate(entities, entity)
		if deduplicated == nil {
			continue // Entity was a duplicate
		}

		// Then link to existing entities in storage
		linked, err := er.Link(ctx, *deduplicated)
		if err != nil {
			return nil, fmt.Errorf("failed to link entity %s: %w", entity.ID, err)
		}

		resolvedEntities = append(resolvedEntities, linked)
	}

	return resolvedEntities, nil
}

// Deduplicate removes duplicate entities from a batch
func (er *EntityResolver) Deduplicate(entities []Entity, target Entity) *Entity {
	// Check if this entity is a duplicate of any previous entity in the batch
	for _, existing := range entities {
		if existing.ID == target.ID {
			continue // Skip self-comparison
		}

		similarity := er.calculateSimilarity(existing, target)
		if similarity > er.similarityThreshold {
			// Merge entities by taking the one with higher confidence
			if target.Confidence > existing.Confidence {
				merged := er.mergeEntities(existing, target)
				return &merged
			}
			return nil // Target is duplicate of higher-confidence existing entity
		}
	}

	return &target
}

// Link links an entity to existing entities in the storage system
func (er *EntityResolver) Link(ctx context.Context, entity Entity) (Entity, error) {
	// Search for similar entities in the vector store
	candidates, err := er.findSimilarEntities(ctx, entity)
	if err != nil {
		return entity, fmt.Errorf("failed to find similar entities: %w", err)
	}

	// If no similar entities found, return the original entity
	if len(candidates) == 0 {
		return entity, nil
	}

	// Find the best match
	bestMatch := er.findBestMatch(entity, candidates)
	if bestMatch == nil {
		return entity, nil
	}

	// Link entities by creating a relationship edge
	if err := er.createEntityLink(ctx, entity, *bestMatch); err != nil {
		return entity, fmt.Errorf("failed to create entity link: %w", err)
	}

	// Merge entity properties and boost confidence
	merged := er.mergeEntities(entity, *bestMatch)
	merged.Confidence = math.Min(1.0, merged.Confidence+er.config.MinConfidenceBoost)

	return merged, nil
}

// findSimilarEntities searches for entities similar to the target entity
func (er *EntityResolver) findSimilarEntities(ctx context.Context, entity Entity) ([]Entity, error) {
	// For now, fall back to name-based search since entities don't have embeddings in the current structure
	return er.findEntitiesByName(ctx, entity.Name)
}

// findEntitiesByName searches for entities by name using fuzzy matching
func (er *EntityResolver) findEntitiesByName(ctx context.Context, name string) ([]Entity, error) {
	if !er.config.EnableFuzzyMatching {
		return []Entity{}, nil
	}

	// Use search index for name-based lookup
	options := SearchIndexOptions{
		Limit: er.config.MaxCandidates,
		Filters: map[string]interface{}{
			"type": "entity",
		},
	}
	
	results, err := er.storage.searchIndex.Search(ctx, name, options)
	if err != nil {
		return nil, err
	}

	var candidates []Entity
	for _, result := range results {
		// Filter for entity results only
		if entityType, ok := result.Metadata["type"]; ok && entityType == "entity" {
			candidate := Entity{
				ID:         result.ID,
				Name:       fmt.Sprintf("%v", result.Metadata["name"]),
				Type:       fmt.Sprintf("%v", result.Metadata["entity_type"]),
				Confidence: result.Score,
			}
			candidates = append(candidates, candidate)
		}
	}

	return candidates, nil
}

// findBestMatch finds the best matching entity from candidates
func (er *EntityResolver) findBestMatch(target Entity, candidates []Entity) *Entity {
	var bestMatch *Entity
	bestSimilarity := 0.0

	for _, candidate := range candidates {
		similarity := er.calculateSimilarity(target, candidate)
		if similarity > er.similarityThreshold && similarity > bestSimilarity {
			bestSimilarity = similarity
			bestMatch = &candidate
		}
	}

	return bestMatch
}

// calculateSimilarity calculates similarity between two entities
func (er *EntityResolver) calculateSimilarity(e1, e2 Entity) float64 {
	// Name similarity (using simple string matching for now)
	nameSimilarity := er.calculateStringSimilarity(e1.Name, e2.Name)
	
	// Type similarity
	typeSimilarity := 0.0
	if e1.Type == e2.Type {
		typeSimilarity = 1.0
	}

	// Weighted combination (no embeddings in current Entity structure)
	weights := map[string]float64{
		"name": 0.7,
		"type": 0.3,
	}

	similarity := weights["name"]*nameSimilarity + weights["type"]*typeSimilarity

	return similarity
}

// calculateStringSimilarity calculates similarity between two strings
func (er *EntityResolver) calculateStringSimilarity(s1, s2 string) float64 {
	s1 = strings.ToLower(strings.TrimSpace(s1))
	s2 = strings.ToLower(strings.TrimSpace(s2))

	if s1 == s2 {
		return 1.0
	}

	// Simple Jaccard similarity using character n-grams
	return er.jaccardSimilarity(s1, s2)
}

// jaccardSimilarity calculates Jaccard similarity between two strings
func (er *EntityResolver) jaccardSimilarity(s1, s2 string) float64 {
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	// Create character bigrams
	for i := 0; i < len(s1)-1; i++ {
		set1[s1[i:i+2]] = true
	}
	for i := 0; i < len(s2)-1; i++ {
		set2[s2[i:i+2]] = true
	}

	// Calculate intersection and union
	intersection := 0
	union := len(set1)

	for bigram := range set2 {
		if set1[bigram] {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// calculateCosineSimilarity calculates cosine similarity between two vectors
func (er *EntityResolver) calculateCosineSimilarity(v1, v2 []float32) float64 {
	if len(v1) != len(v2) {
		return 0.0
	}

	var dotProduct, norm1, norm2 float64
	for i := 0; i < len(v1); i++ {
		dotProduct += float64(v1[i] * v2[i])
		norm1 += float64(v1[i] * v1[i])
		norm2 += float64(v2[i] * v2[i])
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// mergeEntities merges two entities, combining their properties
func (er *EntityResolver) mergeEntities(e1, e2 Entity) Entity {
	merged := Entity{
		ID:         e1.ID, // Keep the first entity's ID
		Name:       e1.Name,
		Type:       e1.Type,
		Properties: make(map[string]interface{}),
		Confidence: math.Max(e1.Confidence, e2.Confidence),
		Source:     e1.Source,
		CreatedAt:  e1.CreatedAt,
	}

	// Copy properties from both entities
	for k, v := range e1.Properties {
		merged.Properties[k] = v
	}
	for k, v := range e2.Properties {
		merged.Properties[k] = v
	}

	// If first entity has lower confidence, use second entity's properties
	if e2.Confidence > e1.Confidence {
		merged.Name = e2.Name
		merged.Type = e2.Type
		merged.Source = e2.Source
	}

	return merged
}

// createEntityLink creates a relationship edge between two entities
func (er *EntityResolver) createEntityLink(ctx context.Context, entity1, entity2 Entity) error {
	edge := &Edge{
		ID:   fmt.Sprintf("link_%s_%s", entity1.ID, entity2.ID),
		From: entity1.ID,
		To:   entity2.ID,
		Type: RelatedTo,
		Weight: er.calculateSimilarity(entity1, entity2),
		Properties: map[string]interface{}{
			"link_type": "entity_resolution",
			"created_by": "entity_resolver",
		},
		CreatedAt: time.Now(),
	}

	return er.storage.graphStore.CreateEdge(ctx, edge)
}