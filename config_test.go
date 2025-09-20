package main

import (
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDefaultServerConfig(t *testing.T) {
	Convey("Given a default server config", t, func() {
		config := DefaultServerConfig()
		
		Convey("Then it should have correct default values", func() {
			So(config.Server.Name, ShouldEqual, "agentic-memory-system")
			So(config.Server.Version, ShouldEqual, "1.0.0")
			So(config.Server.Port, ShouldEqual, 8080)
			So(config.Server.Host, ShouldEqual, "localhost")
			So(config.Server.Timeout, ShouldEqual, 30*time.Second)
			
			So(config.Storage.VectorStore.Provider, ShouldEqual, "memory")
			So(config.Storage.VectorStore.Dimensions, ShouldEqual, 1536)
			So(config.Storage.GraphStore.Provider, ShouldEqual, "memory")
			So(config.Storage.SearchIndex.Provider, ShouldEqual, "memory")
			
			So(config.Processing.EmbeddingModel, ShouldEqual, "text-embedding-ada-002")
			So(config.Processing.MaxChunkSize, ShouldEqual, 1000)
			So(config.Processing.MinConfidence, ShouldEqual, 0.5)
			
			So(config.Performance.MaxConcurrentRequests, ShouldEqual, 100)
			So(config.Performance.DefaultTimeout, ShouldEqual, 5*time.Second)
			
			So(config.Logging.Level, ShouldEqual, "info")
			So(config.Logging.Format, ShouldEqual, "json")
		})
		
		Convey("And it should pass validation", func() {
			err := config.Validate()
			So(err, ShouldBeNil)
		})
	})
}

func TestServerConfigValidation(t *testing.T) {
	Convey("Given server config validation", t, func() {
		config := DefaultServerConfig()
		
		Convey("When validating valid config", func() {
			err := config.Validate()
			
			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})
		
		Convey("When server name is empty", func() {
			config.Server.Name = ""
			err := config.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "server name cannot be empty")
			})
		})
		
		Convey("When server port is invalid", func() {
			config.Server.Port = 0
			err := config.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "server port must be between 1 and 65535")
			})
		})
		
		Convey("When vector store dimensions are invalid", func() {
			config.Storage.VectorStore.Dimensions = 0
			err := config.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "vector dimensions must be positive")
			})
		})
		
		Convey("When chunk overlap is invalid", func() {
			config.Processing.ChunkOverlap = config.Processing.MaxChunkSize
			err := config.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "chunk overlap")
				So(err.Error(), ShouldContainSubstring, "must be less than max chunk size")
			})
		})
		
		Convey("When log level is invalid", func() {
			config.Logging.Level = "invalid"
			err := config.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "invalid log level")
			})
		})
	})
}

func TestServerSettingsValidation(t *testing.T) {
	Convey("Given server settings validation", t, func() {
		settings := ServerSettings{
			Name:    "test-server",
			Version: "1.0.0",
			Port:    8080,
			Host:    "localhost",
			Timeout: 30 * time.Second,
		}
		
		Convey("When validating valid settings", func() {
			err := settings.Validate()
			
			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})
		
		Convey("When name is empty", func() {
			settings.Name = ""
			err := settings.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "server name cannot be empty")
			})
		})
		
		Convey("When version is empty", func() {
			settings.Version = ""
			err := settings.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "server version cannot be empty")
			})
		})
		
		Convey("When port is out of range", func() {
			settings.Port = 70000
			err := settings.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "server port must be between 1 and 65535")
			})
		})
		
		Convey("When host is empty", func() {
			settings.Host = ""
			err := settings.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "server host cannot be empty")
			})
		})
		
		Convey("When timeout is negative", func() {
			settings.Timeout = -1 * time.Second
			err := settings.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "server timeout must be positive")
			})
		})
	})
}

func TestVectorStoreConfigValidation(t *testing.T) {
	Convey("Given vector store config validation", t, func() {
		config := VectorStoreConfig{
			Provider:   "pgvector",
			Dimensions: 1536,
			IndexType:  "ivfflat",
		}
		
		Convey("When validating valid config", func() {
			err := config.Validate()
			
			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})
		
		Convey("When provider is empty", func() {
			config.Provider = ""
			err := config.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "vector store provider cannot be empty")
			})
		})
		
		Convey("When dimensions are invalid", func() {
			config.Dimensions = -1
			err := config.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "vector dimensions must be positive")
			})
		})
		
		Convey("When index type is empty", func() {
			config.IndexType = ""
			err := config.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "vector index type cannot be empty")
			})
		})
	})
}

func TestProcessingConfigValidation(t *testing.T) {
	Convey("Given processing config validation", t, func() {
		config := ProcessingConfig{
			EmbeddingModel:    "text-embedding-ada-002",
			EmbeddingProvider: "openai",
			MaxChunkSize:      1000,
			ChunkOverlap:      200,
			MinConfidence:     0.5,
		}
		
		Convey("When validating valid config", func() {
			err := config.Validate()
			
			Convey("Then validation should pass", func() {
				So(err, ShouldBeNil)
			})
		})
		
		Convey("When embedding model is empty", func() {
			config.EmbeddingModel = ""
			err := config.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "embedding model cannot be empty")
			})
		})
		
		Convey("When max chunk size is invalid", func() {
			config.MaxChunkSize = 0
			err := config.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "max chunk size must be positive")
			})
		})
		
		Convey("When chunk overlap is negative", func() {
			config.ChunkOverlap = -1
			err := config.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "chunk overlap cannot be negative")
			})
		})
		
		Convey("When chunk overlap is too large", func() {
			config.ChunkOverlap = config.MaxChunkSize
			err := config.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "chunk overlap")
				So(err.Error(), ShouldContainSubstring, "must be less than max chunk size")
			})
		})
		
		Convey("When min confidence is out of range", func() {
			config.MinConfidence = 1.5
			err := config.Validate()
			
			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "min confidence must be between 0 and 1")
			})
		})
	})
}

func TestConfigFileOperations(t *testing.T) {
	Convey("Given config file operations", t, func() {
		config := DefaultServerConfig()
		tempFile := "test_config.json"
		
		Convey("When saving config to file", func() {
			err := config.SaveConfig(tempFile)
			
			Convey("Then it should save successfully", func() {
				So(err, ShouldBeNil)
				
				// Check file exists
				_, err := os.Stat(tempFile)
				So(err, ShouldBeNil)
			})
			
			Convey("And when loading config from file", func() {
				loadedConfig, err := LoadConfig(tempFile)
				
				Convey("Then it should load successfully", func() {
					So(err, ShouldBeNil)
					So(loadedConfig.Server.Name, ShouldEqual, config.Server.Name)
					So(loadedConfig.Server.Version, ShouldEqual, config.Server.Version)
					So(loadedConfig.Storage.VectorStore.Dimensions, ShouldEqual, config.Storage.VectorStore.Dimensions)
				})
			})
		})
		
		Reset(func() {
			// Clean up test file
			os.Remove(tempFile)
		})
	})
}

func TestLoadConfigErrors(t *testing.T) {
	Convey("Given config loading errors", t, func() {
		Convey("When loading non-existent file", func() {
			_, err := LoadConfig("non_existent.json")
			
			Convey("Then it should return error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "failed to read config file")
			})
		})
		
		Convey("When loading invalid JSON", func() {
			invalidFile := "invalid_config.json"
			os.WriteFile(invalidFile, []byte("invalid json"), 0644)
			
			_, err := LoadConfig(invalidFile)
			
			Convey("Then it should return error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "failed to parse config file")
			})
			
			Reset(func() {
				os.Remove(invalidFile)
			})
		})
		
		Convey("When loading config with validation errors", func() {
			invalidConfigFile := "invalid_validation_config.json"
			invalidConfig := DefaultServerConfig()
			invalidConfig.Server.Name = "" // This will cause validation to fail
			invalidConfig.SaveConfig(invalidConfigFile)
			
			_, err := LoadConfig(invalidConfigFile)
			
			Convey("Then it should return validation error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "invalid configuration")
			})
			
			Reset(func() {
				os.Remove(invalidConfigFile)
			})
		})
	})
}

func BenchmarkConfigValidation(b *testing.B) {
	config := DefaultServerConfig()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		config.Validate()
	}
}

func BenchmarkConfigSerialization(b *testing.B) {
	config := DefaultServerConfig()
	tempFile := "bench_config.json"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.SaveConfig(tempFile)
		LoadConfig(tempFile)
	}
	
	b.StopTimer()
	os.Remove(tempFile)
}