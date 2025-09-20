package main

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEntityResolver(t *testing.T) {
	Convey("Given an EntityResolver", t, func() {
		storage := &MultiViewStorage{
			vectorStore: NewMockVectorStore(),
			graphStore:  NewMockGraphStore(),
			searchIndex: NewMockSearchIndex(),
		}
		
		resolver := NewEntityResolver(storage)
		
		Convey("When creating a new EntityResolver", func() {
			So(resolver, ShouldNotBeNil)
			So(resolver.storage, ShouldEqual, storage)
			So(resolver.similarityThreshold, ShouldEqual, 0.8)
			So(resolver.config, ShouldNotBeNil)
		})

		Convey("When resolving entities", func() {
			ctx := context.Background()
			entities := []Entity{
				{
					ID:         "entity_1",
					Name:       "Artificial Intelligence",
					Type:       "Technology",
					Confidence: 0.9,
					Source:     "test",
					CreatedAt:  time.Now(),
					Properties: make(map[string]interface{}),
				},
				{
					ID:         "entity_2",
					Name:       "Machine Learning",
					Type:       "Technology",
					Confidence: 0.8,
					Source:     "test",
					CreatedAt:  time.Now(),
					Properties: make(map[string]interface{}),
				},
			}

			resolved, err := resolver.Resolve(ctx, entities)

			Convey("Then it should resolve successfully", func() {
				So(err, ShouldBeNil)
				So(len(resolved), ShouldEqual, len(entities))
				
				for i, entity := range resolved {
					So(entity.ID, ShouldEqual, entities[i].ID)
					So(entity.Name, ShouldEqual, entities[i].Name)
				}
			})
		})

		Convey("When resolving with existing similar entities", func() {
			ctx := context.Background()
			
			// Setup mock vector store with similar entities
			mockVectorStore := storage.vectorStore.(*MockVectorStore)
			mockVectorStore.SetSearchResults([]VectorResult{
				{
					ID:    "existing_entity_1",
					Score: 0.95,
					Metadata: map[string]interface{}{
						"name": "Artificial Intelligence",
						"type": "entity",
					},
				},
			})

			entities := []Entity{
				{
					ID:         "entity_1",
					Name:       "AI", // Similar to existing "Artificial Intelligence"
					Type:       "Technology",
					Confidence: 0.9,
					Source:     "test",
					CreatedAt:  time.Now(),
					Properties: make(map[string]interface{}),
				},
			}

			resolved, err := resolver.Resolve(ctx, entities)

			Convey("Then it should link to existing entities", func() {
				So(err, ShouldBeNil)
				So(len(resolved), ShouldEqual, 1)
				
				// Confidence should be boosted
				So(resolved[0].Confidence, ShouldBeGreaterThan, entities[0].Confidence)
			})
		})
	})
}

func TestEntityResolverDeduplicate(t *testing.T) {
	Convey("Given an EntityResolver", t, func() {
		resolver := NewEntityResolver(&MultiViewStorage{})

		Convey("When deduplicating entities", func() {
			entities := []Entity{
				{
					ID:         "entity_1",
					Name:       "Artificial Intelligence",
					Type:       "Technology",
					Confidence: 0.9,
					Source:     "test",
					CreatedAt:  time.Now(),
					Properties: make(map[string]interface{}),
				},
				{
					ID:         "entity_2",
					Name:       "Artificial Intelligence", // Duplicate
					Type:       "Technology",
					Confidence: 0.7,
					Source:     "test",
					CreatedAt:  time.Now(),
					Properties: make(map[string]interface{}),
				},
				{
					ID:         "entity_3",
					Name:       "Machine Learning",
					Type:       "Technology",
					Confidence: 0.8,
					Source:     "test",
					CreatedAt:  time.Now(),
					Properties: make(map[string]interface{}),
				},
			}

			target := entities[1] // Lower confidence duplicate
			result := resolver.Deduplicate(entities, target)

			Convey("Then it should return nil for duplicates", func() {
				So(result, ShouldBeNil)
			})
		})

		Convey("When deduplicating unique entities", func() {
			entities := []Entity{
				{
					ID:         "entity_1",
					Name:       "Artificial Intelligence",
					Type:       "Technology",
					Confidence: 0.9,
					Source:     "test",
					CreatedAt:  time.Now(),
					Properties: make(map[string]interface{}),
				},
			}

			target := Entity{
				ID:         "entity_2",
				Name:       "Machine Learning",
				Type:       "Technology",
				Confidence: 0.8,
				Source:     "test",
				CreatedAt:  time.Now(),
				Properties: make(map[string]interface{}),
			}

			result := resolver.Deduplicate(entities, target)

			Convey("Then it should return the entity", func() {
				So(result, ShouldNotBeNil)
				So(result.ID, ShouldEqual, target.ID)
				So(result.Name, ShouldEqual, target.Name)
			})
		})
	})
}

func TestEntityResolverLink(t *testing.T) {
	Convey("Given an EntityResolver with mock storage", t, func() {
		mockVectorStore := NewMockVectorStore()
		mockGraphStore := NewMockGraphStore()
		
		storage := &MultiViewStorage{
			vectorStore: mockVectorStore,
			graphStore:  mockGraphStore,
			searchIndex: NewMockSearchIndex(),
		}
		
		resolver := NewEntityResolver(storage)

		Convey("When linking to existing entities", func() {
			ctx := context.Background()
			
			// Setup mock to return similar entity
			mockVectorStore.SetSearchResults([]VectorResult{
				{
					ID:    "existing_entity_1",
					Score: 0.9,
					Metadata: map[string]interface{}{
						"name": "Artificial Intelligence",
						"type": "entity",
					},
				},
			})

			entity := Entity{
				ID:         "entity_1",
				Name:       "AI",
				Type:       "Technology",
				Confidence: 0.8,
				Source:     "test",
				CreatedAt:  time.Now(),
				Properties: make(map[string]interface{}),
			}

			linked, err := resolver.Link(ctx, entity)

			Convey("Then it should create links and boost confidence", func() {
				So(err, ShouldBeNil)
				So(linked.Confidence, ShouldBeGreaterThan, entity.Confidence)
				
				// Verify edge was created
				edges := mockGraphStore.GetEdges()
				So(len(edges), ShouldEqual, 1)
				edge := edges[0]
				So(edge.From, ShouldEqual, entity.ID)
				So(edge.Type, ShouldEqual, RelatedTo)
			})
		})

		Convey("When no similar entities exist", func() {
			ctx := context.Background()
			
			// Setup mock to return no results
			mockVectorStore.SetSearchResults([]VectorResult{})

			entity := Entity{
				ID:         "entity_1",
				Name:       "Unique Entity",
				Type:       "Technology",
				Confidence: 0.8,
				Source:     "test",
				CreatedAt:  time.Now(),
				Properties: make(map[string]interface{}),
			}

			linked, err := resolver.Link(ctx, entity)

			Convey("Then it should return the original entity", func() {
				So(err, ShouldBeNil)
				So(linked, ShouldResemble, entity)
				
				// No edges should be created
				edges := mockGraphStore.GetEdges()
				So(len(edges), ShouldEqual, 0)
			})
		})
	})
}

func TestEntityResolverSimilarity(t *testing.T) {
	Convey("Given an EntityResolver", t, func() {
		resolver := NewEntityResolver(&MultiViewStorage{})

		Convey("When calculating similarity between identical entities", func() {
			entity1 := Entity{
				Name:       "Artificial Intelligence",
				Type:       "Technology",
				Source:     "test",
				CreatedAt:  time.Now(),
				Properties: make(map[string]interface{}),
			}
			entity2 := Entity{
				Name:       "Artificial Intelligence",
				Type:       "Technology",
				Source:     "test",
				CreatedAt:  time.Now(),
				Properties: make(map[string]interface{}),
			}

			similarity := resolver.calculateSimilarity(entity1, entity2)

			Convey("Then similarity should be high", func() {
				So(similarity, ShouldBeGreaterThan, 0.9)
			})
		})

		Convey("When calculating similarity between different entities", func() {
			entity1 := Entity{
				Name:       "Artificial Intelligence",
				Type:       "Technology",
				Source:     "test",
				CreatedAt:  time.Now(),
				Properties: make(map[string]interface{}),
			}
			entity2 := Entity{
				Name:       "Quantum Computing",
				Type:       "Physics",
				Source:     "test",
				CreatedAt:  time.Now(),
				Properties: make(map[string]interface{}),
			}

			similarity := resolver.calculateSimilarity(entity1, entity2)

			Convey("Then similarity should be low", func() {
				So(similarity, ShouldBeLessThan, 0.5)
			})
		})

		Convey("When calculating string similarity", func() {
			similarity1 := resolver.calculateStringSimilarity("AI", "Artificial Intelligence")
			similarity2 := resolver.calculateStringSimilarity("Machine Learning", "ML")
			similarity3 := resolver.calculateStringSimilarity("identical", "identical")

			Convey("Then it should handle various cases", func() {
				So(similarity1, ShouldBeGreaterThan, 0)
				So(similarity2, ShouldBeGreaterThan, 0)
				So(similarity3, ShouldEqual, 1.0)
			})
		})

		// Cosine similarity test removed since embeddings are not part of Entity struct
	})
}

func TestEntityResolverMerge(t *testing.T) {
	Convey("Given an EntityResolver", t, func() {
		resolver := NewEntityResolver(&MultiViewStorage{})

		Convey("When merging entities", func() {
			entity1 := Entity{
				ID:         "entity_1",
				Name:       "AI",
				Type:       "Technology",
				Confidence: 0.7,
				Source:     "test",
				CreatedAt:  time.Now(),
				Properties: make(map[string]interface{}),
			}
			entity2 := Entity{
				ID:         "entity_2",
				Name:       "Artificial Intelligence",
				Type:       "Technology",
				Confidence: 0.9,
				Source:     "test",
				CreatedAt:  time.Now(),
				Properties: make(map[string]interface{}),
			}

			merged := resolver.mergeEntities(entity1, entity2)

			Convey("Then it should use higher confidence properties", func() {
				So(merged.ID, ShouldEqual, entity1.ID) // Keep first ID
				So(merged.Name, ShouldEqual, entity2.Name) // Use higher confidence name
				So(merged.Confidence, ShouldEqual, entity2.Confidence) // Use max confidence
				So(merged.Source, ShouldEqual, entity2.Source) // Use higher confidence source
			})
		})
	})
}