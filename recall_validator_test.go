package main

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRecallArgsValidator(t *testing.T) {
	Convey("Given a RecallArgsValidator", t, func() {
		validator := NewRecallArgsValidator()

		Convey("When creating a new validator", func() {
			So(validator, ShouldNotBeNil)
			So(validator.config, ShouldNotBeNil)
			So(validator.config.MaxQueryLength, ShouldEqual, 1000)
			So(validator.config.MinQueryLength, ShouldEqual, 1)
			So(validator.config.MaxResults, ShouldEqual, 100)
		})

		Convey("When creating with custom config", func() {
			config := &RecallValidatorConfig{
				MaxQueryLength: 500,
				MinQueryLength: 5,
				MaxResults:     50,
				SanitizeHTML:   false,
			}
			customValidator := NewRecallArgsValidatorWithConfig(config)

			So(customValidator.config.MaxQueryLength, ShouldEqual, 500)
			So(customValidator.config.MinQueryLength, ShouldEqual, 5)
			So(customValidator.config.MaxResults, ShouldEqual, 50)
			So(customValidator.config.SanitizeHTML, ShouldBeFalse)
		})
	})
}

func TestRecallArgsValidatorValidate(t *testing.T) {
	Convey("Given a RecallArgsValidator", t, func() {
		validator := NewRecallArgsValidator()

		Convey("When validating valid arguments", func() {
			args := RecallArgs{
				Query:        "test query",
				MaxResults:   10,
				TimeBudget:   5000,
				IncludeGraph: true,
				Filters:      map[string]interface{}{"source": "test"},
			}

			err := validator.Validate(args)

			Convey("Then validation should succeed", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When validating empty query", func() {
			args := RecallArgs{
				Query:      "",
				MaxResults: 10,
			}

			err := validator.Validate(args)

			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "query cannot be empty")
			})
		})

		Convey("When validating query that's too long", func() {
			longQuery := make([]byte, 1001)
			for i := range longQuery {
				longQuery[i] = 'a'
			}

			args := RecallArgs{
				Query:      string(longQuery),
				MaxResults: 10,
			}

			err := validator.Validate(args)

			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "exceeds maximum length")
			})
		})

		Convey("When validating negative max results", func() {
			args := RecallArgs{
				Query:      "test query",
				MaxResults: -1,
			}

			err := validator.Validate(args)

			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "maxResults cannot be negative")
			})
		})

		Convey("When validating excessive max results", func() {
			args := RecallArgs{
				Query:      "test query",
				MaxResults: 1000,
			}

			err := validator.Validate(args)

			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "maxResults exceeds maximum")
			})
		})

		Convey("When validating negative time budget", func() {
			args := RecallArgs{
				Query:      "test query",
				TimeBudget: -1000,
			}

			err := validator.Validate(args)

			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "timeBudget cannot be negative")
			})
		})

		Convey("When validating excessive time budget", func() {
			args := RecallArgs{
				Query:      "test query",
				TimeBudget: 60000, // 60 seconds, exceeds 30s default max
			}

			err := validator.Validate(args)

			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "timeBudget exceeds maximum")
			})
		})
	})
}

func TestRecallArgsValidatorValidateDetailed(t *testing.T) {
	Convey("Given a RecallArgsValidator", t, func() {
		validator := NewRecallArgsValidator()

		Convey("When performing detailed validation on valid args", func() {
			args := RecallArgs{
				Query:      "test query",
				MaxResults: 10,
				TimeBudget: 5000,
				Filters:    map[string]interface{}{"source": "test", "confidence": 0.8},
			}

			result := validator.ValidateDetailed(args)

			Convey("Then result should be valid", func() {
				So(result.Valid, ShouldBeTrue)
				So(result.Errors, ShouldHaveLength, 0)
				So(result.Sanitized.Query, ShouldEqual, "test query")
			})
		})

		Convey("When performing detailed validation on invalid args", func() {
			args := RecallArgs{
				Query:      "",
				MaxResults: -1,
				TimeBudget: -1000,
			}

			result := validator.ValidateDetailed(args)

			Convey("Then result should be invalid with multiple errors", func() {
				So(result.Valid, ShouldBeFalse)
				So(result.Errors, ShouldHaveLength, 3)
				
				errorMessages := make([]string, len(result.Errors))
				for i, err := range result.Errors {
					errorMessages[i] = err.Message
				}
				
				So(errorMessages, ShouldContain, "query cannot be empty")
				So(errorMessages, ShouldContain, "maxResults cannot be negative")
				So(errorMessages, ShouldContain, "timeBudget cannot be negative")
			})
		})

		Convey("When validating args with blocked patterns", func() {
			args := RecallArgs{
				Query:      "test <script>alert('xss')</script> query",
				MaxResults: 10,
			}

			result := validator.ValidateDetailed(args)

			Convey("Then it should detect blocked patterns", func() {
				So(result.Valid, ShouldBeFalse)
				So(len(result.Errors), ShouldBeGreaterThan, 0)
				
				hasBlockedPatternError := false
				for _, err := range result.Errors {
					if err.Message == "query contains blocked pattern: <script" {
						hasBlockedPatternError = true
						break
					}
				}
				So(hasBlockedPatternError, ShouldBeTrue)
			})
		})

		Convey("When validating args with disallowed filters", func() {
			args := RecallArgs{
				Query:      "test query",
				MaxResults: 10,
				Filters: map[string]interface{}{
					"source":        "test",      // allowed
					"malicious_key": "bad_value", // not allowed
				},
			}

			result := validator.ValidateDetailed(args)

			Convey("Then it should filter out disallowed keys", func() {
				So(result.Valid, ShouldBeTrue)
				So(len(result.Warnings), ShouldBeGreaterThan, 0)
				So(result.Sanitized.Filters, ShouldContainKey, "source")
				So(result.Sanitized.Filters, ShouldNotContainKey, "malicious_key")
			})
		})
	})
}

func TestRecallArgsValidatorCheckArgs(t *testing.T) {
	Convey("Given a RecallArgsValidator", t, func() {
		validator := NewRecallArgsValidator()

		Convey("When checking valid arguments", func() {
			args := RecallArgs{
				Query:      "test query",
				MaxResults: 10,
				TimeBudget: 5000,
			}

			issues := validator.CheckArgs(args)

			Convey("Then no issues should be found", func() {
				So(issues, ShouldHaveLength, 0)
			})
		})

		Convey("When checking arguments with multiple issues", func() {
			args := RecallArgs{
				Query:      "",
				MaxResults: 200,
				TimeBudget: 60000,
			}

			issues := validator.CheckArgs(args)

			Convey("Then multiple issues should be found", func() {
				So(len(issues), ShouldBeGreaterThan, 0)
				So(issues, ShouldContain, "query cannot be empty")
				So(issues, ShouldContain, "maxResults exceeds maximum of 100")
				So(issues, ShouldContain, "timeBudget exceeds maximum of 30000 milliseconds")
			})
		})
	})
}

func TestRecallArgsValidatorSanitizeInput(t *testing.T) {
	Convey("Given a RecallArgsValidator", t, func() {
		validator := NewRecallArgsValidator()

		Convey("When sanitizing valid input", func() {
			args := RecallArgs{
				Query:      "  test query  ",
				MaxResults: 10,
				TimeBudget: 5000,
				Filters:    map[string]interface{}{"source": "test"},
			}

			sanitized, err := validator.SanitizeInput(args)

			Convey("Then input should be sanitized successfully", func() {
				So(err, ShouldBeNil)
				So(sanitized.Query, ShouldEqual, "test query") // trimmed
				So(sanitized.MaxResults, ShouldEqual, 10)
				So(sanitized.Filters, ShouldContainKey, "source")
			})
		})

		Convey("When sanitizing input with HTML", func() {
			args := RecallArgs{
				Query:      "<b>test</b> query",
				MaxResults: 10,
			}

			sanitized, err := validator.SanitizeInput(args)

			Convey("Then HTML should be escaped", func() {
				So(err, ShouldBeNil)
				So(sanitized.Query, ShouldEqual, "&lt;b&gt;test&lt;/b&gt; query")
			})
		})

		Convey("When sanitizing invalid input", func() {
			args := RecallArgs{
				Query:      "",
				MaxResults: -1,
			}

			_, err := validator.SanitizeInput(args)

			Convey("Then sanitization should fail", func() {
				So(err, ShouldNotBeNil)
			})
		})

		Convey("When sanitizing input with zero values", func() {
			args := RecallArgs{
				Query:      "test query",
				MaxResults: 0,
				TimeBudget: 0,
			}

			sanitized, err := validator.SanitizeInput(args)

			Convey("Then defaults should be applied", func() {
				So(err, ShouldBeNil)
				So(sanitized.MaxResults, ShouldEqual, 10)    // default
				So(sanitized.TimeBudget, ShouldEqual, 5000)  // default
			})
		})
	})
}

func TestRecallArgsValidatorSanitizeQuery(t *testing.T) {
	Convey("Given a RecallArgsValidator", t, func() {
		validator := NewRecallArgsValidator()

		Convey("When sanitizing query with whitespace", func() {
			query := "  test   query  "
			sanitized := validator.sanitizeQuery(query)

			Convey("Then whitespace should be normalized", func() {
				So(sanitized, ShouldEqual, "test query")
			})
		})

		Convey("When sanitizing query with HTML", func() {
			query := "<script>alert('xss')</script>"
			sanitized := validator.sanitizeQuery(query)

			Convey("Then HTML should be escaped", func() {
				So(sanitized, ShouldEqual, "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;")
			})
		})

		Convey("When sanitizing query with control characters", func() {
			query := "test\x00\x01query"
			sanitized := validator.sanitizeQuery(query)

			Convey("Then control characters should be removed", func() {
				So(sanitized, ShouldEqual, "testquery")
			})
		})

		Convey("When sanitizing query with multiple spaces", func() {
			query := "test    multiple     spaces"
			sanitized := validator.sanitizeQuery(query)

			Convey("Then multiple spaces should be normalized", func() {
				So(sanitized, ShouldEqual, "test multiple spaces")
			})
		})
	})
}

func TestRecallArgsValidatorSanitizeFilters(t *testing.T) {
	Convey("Given a RecallArgsValidator", t, func() {
		validator := NewRecallArgsValidator()

		Convey("When sanitizing filters with allowed keys", func() {
			filters := map[string]interface{}{
				"source":     "test source",
				"confidence": 0.8,
				"date":       "2024-01-01",
			}

			sanitized := validator.sanitizeFilters(filters)

			Convey("Then allowed filters should be preserved", func() {
				So(sanitized, ShouldContainKey, "source")
				So(sanitized, ShouldContainKey, "confidence")
				So(sanitized, ShouldContainKey, "date")
				So(sanitized["source"], ShouldEqual, "test source")
				So(sanitized["confidence"], ShouldEqual, 0.8)
			})
		})

		Convey("When sanitizing filters with disallowed keys", func() {
			filters := map[string]interface{}{
				"source":        "test",
				"malicious_key": "bad_value",
				"another_bad":   "value",
			}

			sanitized := validator.sanitizeFilters(filters)

			Convey("Then only allowed filters should be preserved", func() {
				So(sanitized, ShouldContainKey, "source")
				So(sanitized, ShouldNotContainKey, "malicious_key")
				So(sanitized, ShouldNotContainKey, "another_bad")
			})
		})

		Convey("When sanitizing nil filters", func() {
			sanitized := validator.sanitizeFilters(nil)

			Convey("Then result should be nil", func() {
				So(sanitized, ShouldBeNil)
			})
		})

		Convey("When sanitizing filters with HTML in values", func() {
			filters := map[string]interface{}{
				"source": "<script>alert('xss')</script>",
			}

			sanitized := validator.sanitizeFilters(filters)

			Convey("Then HTML should be escaped in values", func() {
				So(sanitized["source"], ShouldEqual, "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;")
			})
		})
	})
}

func TestRecallArgsValidatorConfiguration(t *testing.T) {
	Convey("Given a RecallArgsValidator", t, func() {
		validator := NewRecallArgsValidator()

		Convey("When getting configuration", func() {
			config := validator.GetConfig()

			Convey("Then it should return the current config", func() {
				So(config, ShouldNotBeNil)
				So(config.MaxQueryLength, ShouldEqual, 1000)
				So(config.MinQueryLength, ShouldEqual, 1)
				So(config.MaxResults, ShouldEqual, 100)
				So(config.MaxTimeBudget, ShouldEqual, 30*time.Second)
			})
		})

		Convey("When updating configuration", func() {
			newConfig := &RecallValidatorConfig{
				MaxQueryLength: 500,
				MinQueryLength: 5,
				MaxResults:     50,
				SanitizeHTML:   false,
			}

			validator.UpdateConfig(newConfig)
			config := validator.GetConfig()

			Convey("Then the config should be updated", func() {
				So(config.MaxQueryLength, ShouldEqual, 500)
				So(config.MinQueryLength, ShouldEqual, 5)
				So(config.MaxResults, ShouldEqual, 50)
				So(config.SanitizeHTML, ShouldBeFalse)
			})
		})

		Convey("When updating with nil config", func() {
			originalConfig := validator.GetConfig()
			validator.UpdateConfig(nil)
			currentConfig := validator.GetConfig()

			Convey("Then the config should remain unchanged", func() {
				So(currentConfig, ShouldEqual, originalConfig)
			})
		})
	})
}

func TestRecallArgsValidatorValidateAndSanitize(t *testing.T) {
	Convey("Given a RecallArgsValidator", t, func() {
		validator := NewRecallArgsValidator()

		Convey("When validating and sanitizing valid args", func() {
			args := RecallArgs{
				Query:      "  test query  ",
				MaxResults: 10,
				TimeBudget: 5000,
			}

			sanitized, warnings, err := validator.ValidateAndSanitize(args)

			Convey("Then it should succeed with sanitized output", func() {
				So(err, ShouldBeNil)
				So(sanitized.Query, ShouldEqual, "test query")
				So(len(warnings), ShouldBeGreaterThanOrEqualTo, 0)
			})
		})

		Convey("When validating and sanitizing invalid args", func() {
			args := RecallArgs{
				Query:      "",
				MaxResults: -1,
			}

			_, warnings, err := validator.ValidateAndSanitize(args)

			Convey("Then it should fail with error", func() {
				So(err, ShouldNotBeNil)
				So(len(warnings), ShouldBeGreaterThanOrEqualTo, 0)
			})
		})

		Convey("When validating args that generate warnings", func() {
			args := RecallArgs{
				Query:      "test query",
				MaxResults: 0, // Will generate warning and be set to default
				Filters: map[string]interface{}{
					"source":     "test",
					"bad_filter": "value", // Will generate warning
				},
			}

			sanitized, warnings, err := validator.ValidateAndSanitize(args)

			Convey("Then it should succeed with warnings", func() {
				So(err, ShouldBeNil)
				So(len(warnings), ShouldBeGreaterThan, 0)
				So(sanitized.MaxResults, ShouldEqual, 10) // default applied
			})
		})
	})
}