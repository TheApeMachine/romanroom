package main

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWriteArgsValidator(t *testing.T) {
	Convey("WriteArgsValidator", t, func() {
		validator := NewWriteArgsValidator()

		Convey("NewWriteArgsValidator", func() {
			Convey("Should create validator with default config", func() {
				So(validator, ShouldNotBeNil)
				So(validator.config, ShouldNotBeNil)
				So(validator.config.MaxContentLength, ShouldEqual, 10000)
				So(validator.config.MinContentLength, ShouldEqual, 1)
				So(validator.config.RequireSource, ShouldBeTrue)
				So(validator.config.SanitizeHTML, ShouldBeTrue)
				So(validator.config.ValidateUTF8, ShouldBeTrue)
			})

			Convey("Should create validator with custom config", func() {
				config := &WriteValidatorConfig{
					MaxContentLength: 5000,
					MinContentLength: 10,
					RequireSource:    false,
					SanitizeHTML:     false,
					ValidateUTF8:     false,
				}
				customValidator := NewWriteArgsValidatorWithConfig(config)

				So(customValidator.config.MaxContentLength, ShouldEqual, 5000)
				So(customValidator.config.MinContentLength, ShouldEqual, 10)
				So(customValidator.config.RequireSource, ShouldBeFalse)
				So(customValidator.config.SanitizeHTML, ShouldBeFalse)
				So(customValidator.config.ValidateUTF8, ShouldBeFalse)
			})
		})

		Convey("Validate", func() {
			Convey("Should validate correct arguments", func() {
				args := WriteArgs{
					Content: "This is valid content for testing.",
					Source:  "test_source",
					Tags:    []string{"test", "valid"},
					Metadata: map[string]interface{}{
						"key1": "value1",
						"key2": 42,
					},
				}

				err := validator.Validate(args)
				So(err, ShouldBeNil)
			})

			Convey("Should reject empty content", func() {
				args := WriteArgs{
					Content: "",
					Source:  "test_source",
				}

				err := validator.Validate(args)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "content cannot be empty")
			})

			Convey("Should reject whitespace-only content", func() {
				args := WriteArgs{
					Content: "   \n\t   ",
					Source:  "test_source",
				}

				err := validator.Validate(args)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "content cannot be empty")
			})

			Convey("Should reject missing source when required", func() {
				args := WriteArgs{
					Content: "Valid content",
					Source:  "",
				}

				err := validator.Validate(args)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "source is required")
			})

			Convey("Should reject content that is too long", func() {
				longContent := strings.Repeat("a", 10001)
				args := WriteArgs{
					Content: longContent,
					Source:  "test_source",
				}

				err := validator.Validate(args)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "exceeds maximum length")
			})

			Convey("Should reject source that is too long", func() {
				longSource := strings.Repeat("a", 201)
				args := WriteArgs{
					Content: "Valid content",
					Source:  longSource,
				}

				err := validator.Validate(args)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "source exceeds maximum length")
			})

			Convey("Should reject too many tags", func() {
				manyTags := make([]string, 11)
				for i := range manyTags {
					manyTags[i] = "tag" + string(rune('0'+i))
				}

				args := WriteArgs{
					Content: "Valid content",
					Source:  "test_source",
					Tags:    manyTags,
				}

				err := validator.Validate(args)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "tags exceed maximum count")
			})

			Convey("Should reject tags that are too long", func() {
				longTag := strings.Repeat("a", 51)
				args := WriteArgs{
					Content: "Valid content",
					Source:  "test_source",
					Tags:    []string{longTag},
				}

				err := validator.Validate(args)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "exceeds maximum length")
			})

			Convey("Should reject blocked patterns", func() {
				args := WriteArgs{
					Content: "This content has <script>alert('xss')</script> in it",
					Source:  "test_source",
				}

				err := validator.Validate(args)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "blocked pattern")
			})
		})

		Convey("ValidateDetailed", func() {
			Convey("Should return detailed validation results", func() {
				args := WriteArgs{
					Content: "Valid content",
					Source:  "test_source",
					Tags:    []string{"test"},
				}

				result := validator.ValidateDetailed(args)

				So(result, ShouldNotBeNil)
				So(result.Valid, ShouldBeTrue)
				So(len(result.Errors), ShouldEqual, 0)
				So(result.Sanitized.Content, ShouldEqual, "Valid content")
			})

			Convey("Should return errors for invalid content", func() {
				args := WriteArgs{
					Content: "",
					Source:  "",
				}

				result := validator.ValidateDetailed(args)

				So(result, ShouldNotBeNil)
				So(result.Valid, ShouldBeFalse)
				So(len(result.Errors), ShouldBeGreaterThan, 0)
				So(result.Errors[0].Field, ShouldEqual, "content")
			})

			Convey("Should return warnings for non-critical issues", func() {
				args := WriteArgs{
					Content: "Valid content",
					Source:  "test_source",
					Tags:    []string{"", "valid_tag", ""}, // Empty tags
				}

				result := validator.ValidateDetailed(args)

				So(result, ShouldNotBeNil)
				So(result.Valid, ShouldBeTrue)
				So(len(result.Warnings), ShouldBeGreaterThan, 0)
				So(len(result.Sanitized.Tags), ShouldEqual, 1) // Empty tags removed
			})
		})

		Convey("CheckContent", func() {
			Convey("Should return no issues for valid content", func() {
				content := "This is valid content for testing."
				issues := validator.CheckContent(content)

				So(len(issues), ShouldEqual, 0)
			})

			Convey("Should return issues for empty content", func() {
				content := ""
				issues := validator.CheckContent(content)

				So(len(issues), ShouldBeGreaterThan, 0)
				So(issues[0], ShouldContainSubstring, "content cannot be empty")
			})

			Convey("Should return issues for content that is too long", func() {
				content := strings.Repeat("a", 10001)
				issues := validator.CheckContent(content)

				So(len(issues), ShouldBeGreaterThan, 0)
				So(issues[0], ShouldContainSubstring, "exceeds maximum length")
			})

			Convey("Should return issues for invalid UTF-8", func() {
				// Create invalid UTF-8 content
				content := string([]byte{0xff, 0xfe, 0xfd})
				issues := validator.CheckContent(content)

				So(len(issues), ShouldBeGreaterThan, 0)
				So(issues[0], ShouldContainSubstring, "invalid UTF-8")
			})
		})

		Convey("ValidateMetadata", func() {
			Convey("Should return no issues for valid metadata", func() {
				metadata := map[string]interface{}{
					"key1": "value1",
					"key2": 42,
					"key3": true,
				}

				issues := validator.ValidateMetadata(metadata)
				So(len(issues), ShouldEqual, 0)
			})

			Convey("Should return issues for too many keys", func() {
				metadata := make(map[string]interface{})
				for i := 0; i < 21; i++ {
					metadata[string(rune('a'+i))] = "value"
				}

				issues := validator.ValidateMetadata(metadata)
				So(len(issues), ShouldBeGreaterThan, 0)
				So(issues[0], ShouldContainSubstring, "exceeds maximum")
			})

			Convey("Should return issues for empty keys", func() {
				metadata := map[string]interface{}{
					"": "value",
				}

				issues := validator.ValidateMetadata(metadata)
				So(len(issues), ShouldBeGreaterThan, 0)
				So(issues[0], ShouldContainSubstring, "key cannot be empty")
			})

			Convey("Should return issues for long values", func() {
				longValue := strings.Repeat("a", 501)
				metadata := map[string]interface{}{
					"key": longValue,
				}

				issues := validator.ValidateMetadata(metadata)
				So(len(issues), ShouldBeGreaterThan, 0)
				So(issues[0], ShouldContainSubstring, "exceeds maximum length")
			})
		})

		Convey("SanitizeInput", func() {
			Convey("Should sanitize valid input", func() {
				args := WriteArgs{
					Content: "  Valid content with extra spaces  ",
					Source:  "  test_source  ",
					Tags:    []string{"  TAG1  ", "tag2", ""},
					Metadata: map[string]interface{}{
						"key1": "  value1  ",
						"key2": 42,
					},
				}

				sanitized, err := validator.SanitizeInput(args)

				So(err, ShouldBeNil)
				So(sanitized.Content, ShouldEqual, "Valid content with extra spaces")
				So(sanitized.Source, ShouldEqual, "test_source")
				So(len(sanitized.Tags), ShouldEqual, 2) // Empty tag removed
				So(sanitized.Tags[0], ShouldEqual, "tag1") // Lowercase and trimmed
				So(sanitized.Tags[1], ShouldEqual, "tag2")
			})

			Convey("Should return error for invalid input", func() {
				args := WriteArgs{
					Content: "", // Invalid
					Source:  "test_source",
				}

				_, err := validator.SanitizeInput(args)
				So(err, ShouldNotBeNil)
			})

			Convey("Should sanitize HTML if enabled", func() {
				args := WriteArgs{
					Content: "Content with <b>HTML</b> tags",
					Source:  "test_source",
				}

				sanitized, err := validator.SanitizeInput(args)

				So(err, ShouldBeNil)
				So(sanitized.Content, ShouldContainSubstring, "&amp;lt;b&amp;gt;")
				So(sanitized.Content, ShouldContainSubstring, "&amp;lt;/b&amp;gt;")
			})
		})

		Convey("Configuration", func() {
			Convey("GetConfig should return current config", func() {
				config := validator.GetConfig()
				So(config, ShouldNotBeNil)
				So(config.MaxContentLength, ShouldEqual, 10000)
			})

			Convey("UpdateConfig should update configuration", func() {
				newConfig := &WriteValidatorConfig{
					MaxContentLength: 5000,
					RequireSource:    false,
				}

				validator.UpdateConfig(newConfig)
				config := validator.GetConfig()

				So(config.MaxContentLength, ShouldEqual, 5000)
				So(config.RequireSource, ShouldBeFalse)
			})
		})

		Convey("ValidateAndSanitize", func() {
			Convey("Should validate and sanitize in one call", func() {
				args := WriteArgs{
					Content: "  Valid content  ",
					Source:  "test_source",
					Tags:    []string{"TAG1", "tag2"},
				}

				sanitized, warnings, err := validator.ValidateAndSanitize(args)

				So(err, ShouldBeNil)
				So(sanitized.Content, ShouldEqual, "Valid content")
				So(len(sanitized.Tags), ShouldEqual, 2)
				So(warnings, ShouldNotBeNil)
			})

			Convey("Should return error for invalid input", func() {
				args := WriteArgs{
					Content: "",
					Source:  "test_source",
				}

				_, warnings, err := validator.ValidateAndSanitize(args)

				So(err, ShouldNotBeNil)
				So(warnings, ShouldNotBeNil)
			})
		})
	})
}

func TestWriteValidatorEdgeCases(t *testing.T) {
	Convey("WriteValidator Edge Cases", t, func() {
		validator := NewWriteArgsValidator()

		Convey("Should handle Unicode content", func() {
			args := WriteArgs{
				Content: "Content with émojis 🚀 and ünïcödé characters",
				Source:  "unicode_test",
				Tags:    []string{"unicode", "émoji"},
			}

			err := validator.Validate(args)
			So(err, ShouldBeNil)
		})

		Convey("Should handle content at exact limits", func() {
			// Content at exactly max length
			exactContent := strings.Repeat("a", 10000)
			args := WriteArgs{
				Content: exactContent,
				Source:  "limit_test",
			}

			err := validator.Validate(args)
			So(err, ShouldBeNil)
		})

		Convey("Should handle complex metadata structures", func() {
			args := WriteArgs{
				Content: "Valid content",
				Source:  "test_source",
				Metadata: map[string]interface{}{
					"nested": map[string]interface{}{
						"level1": map[string]interface{}{
							"level2": "deep_value",
						},
					},
					"array": []interface{}{"item1", "item2", 42, true},
					"mixed": []interface{}{
						map[string]interface{}{"key": "value"},
						"string_item",
						123,
					},
				},
			}

			err := validator.Validate(args)
			So(err, ShouldBeNil)
		})

		Convey("Should handle special characters in tags", func() {
			args := WriteArgs{
				Content: "Valid content",
				Source:  "test_source",
				Tags:    []string{"tag-with-hyphens", "tag_with_underscores", "tag with spaces"},
			}

			sanitized, err := validator.SanitizeInput(args)
			So(err, ShouldBeNil)
			
			// Spaces should be removed, hyphens and underscores preserved
			So(sanitized.Tags[0], ShouldEqual, "tag-with-hyphens")
			So(sanitized.Tags[1], ShouldEqual, "tag_with_underscores")
			So(sanitized.Tags[2], ShouldEqual, "tagwithspaces")
		})

		Convey("Should handle control characters", func() {
			// Content with control characters
			contentWithControl := "Content with\x00null\x01and\x02control\x03characters"
			args := WriteArgs{
				Content: contentWithControl,
				Source:  "control_test",
			}

			sanitized, err := validator.SanitizeInput(args)
			So(err, ShouldBeNil)
			
			// Control characters should be removed
			So(sanitized.Content, ShouldNotContainSubstring, "\x00")
			So(sanitized.Content, ShouldNotContainSubstring, "\x01")
			So(sanitized.Content, ShouldContainSubstring, "Content with")
			So(sanitized.Content, ShouldContainSubstring, "characters")
		})

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

		Convey("Should handle nil metadata gracefully", func() {
			args := WriteArgs{
				Content:  "Valid content",
				Source:   "test_source",
				Metadata: nil,
			}

			err := validator.Validate(args)
			So(err, ShouldBeNil)
		})

		Convey("Should handle empty tags slice gracefully", func() {
			args := WriteArgs{
				Content: "Valid content",
				Source:  "test_source",
				Tags:    []string{},
			}

			err := validator.Validate(args)
			So(err, ShouldBeNil)
		})
	})
}

func BenchmarkWriteValidator(b *testing.B) {
	validator := NewWriteArgsValidator()
	args := WriteArgs{
		Content: "This is benchmark content for testing validator performance with various elements.",
		Source:  "benchmark_test",
		Tags:    []string{"benchmark", "performance", "test"},
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.Validate(args)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

func BenchmarkWriteValidatorSanitize(b *testing.B) {
	validator := NewWriteArgsValidator()
	args := WriteArgs{
		Content: "  Content with <b>HTML</b> and   extra   spaces  ",
		Source:  "  benchmark_test  ",
		Tags:    []string{"  TAG1  ", "tag2", ""},
		Metadata: map[string]interface{}{
			"key1": "  value1  ",
			"key2": 42,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.SanitizeInput(args)
		if err != nil {
			b.Fatalf("Sanitization failed: %v", err)
		}
	}
}