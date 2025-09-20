package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ServerConfig holds the complete server configuration
type ServerConfig struct {
	Server      ServerSettings    `json:"server"`
	Storage     MultiViewStorageConfig     `json:"storage"`
	Processing  ProcessingConfig  `json:"processing"`
	Performance PerformanceConfig `json:"performance"`
	Logging     LoggingConfig     `json:"logging"`
}

// ServerSettings holds MCP server specific settings
type ServerSettings struct {
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Description string        `json:"description"`
	Port        int           `json:"port"`
	Host        string        `json:"host"`
	Timeout     time.Duration `json:"timeout"`
}

// ProcessingConfig holds content processing configuration
type ProcessingConfig struct {
	EmbeddingModel    string  `json:"embedding_model"`
	EmbeddingProvider string  `json:"embedding_provider"`
	MaxChunkSize      int     `json:"max_chunk_size"`
	ChunkOverlap      int     `json:"chunk_overlap"`
	MinConfidence     float64 `json:"min_confidence"`
	EntityExtraction  bool    `json:"entity_extraction"`
	ClaimExtraction   bool    `json:"claim_extraction"`
}

// PerformanceConfig holds performance-related settings
type PerformanceConfig struct {
	MaxConcurrentRequests int           `json:"max_concurrent_requests"`
	DefaultTimeout        time.Duration `json:"default_timeout"`
	CacheSize             int           `json:"cache_size"`
	BatchSize             int           `json:"batch_size"`
	WorkerPoolSize        int           `json:"worker_pool_size"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	Output     string `json:"output"`
	MaxSize    int    `json:"max_size"`
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age"`
}

// DefaultServerConfig returns a default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Server: ServerSettings{
			Name:        "agentic-memory-system",
			Version:     "1.0.0",
			Description: "Agentic Memory System MCP Server",
			Port:        8080,
			Host:        "localhost",
			Timeout:     30 * time.Second,
		},
		Storage: MultiViewStorageConfig{
			VectorStore: VectorStoreConfig{
				Provider:   "memory",

				Dimensions: 1536,
				IndexType:  "flat",
			},
			GraphStore: GraphStoreConfig{
				Provider: "memory",
				URI:      "",
				Database: "memory",
			},
			SearchIndex: SearchIndexConfig{
				Provider: "memory",
				URI:      "",
				IndexName: "memory",
			},
			Timeout: 10 * time.Second,
		},
		Processing: ProcessingConfig{
			EmbeddingModel:    "text-embedding-ada-002",
			EmbeddingProvider: "openai",
			MaxChunkSize:      1000,
			ChunkOverlap:      200,
			MinConfidence:     0.5,
			EntityExtraction:  true,
			ClaimExtraction:   true,
		},
		Performance: PerformanceConfig{
			MaxConcurrentRequests: 100,
			DefaultTimeout:        5 * time.Second,
			CacheSize:             1000,
			BatchSize:             100,
			WorkerPoolSize:        10,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
		},
	}
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(filename string) (*ServerConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}
	
	var config ServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}
	
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}
	
	return &config, nil
}

// SaveConfig saves configuration to a JSON file
func (c *ServerConfig) SaveConfig(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}
	
	return nil
}

// Validate validates the server configuration
func (c *ServerConfig) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config validation failed: %v", err)
	}
	
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage config validation failed: %v", err)
	}
	
	if err := c.Processing.Validate(); err != nil {
		return fmt.Errorf("processing config validation failed: %v", err)
	}
	
	if err := c.Performance.Validate(); err != nil {
		return fmt.Errorf("performance config validation failed: %v", err)
	}
	
	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging config validation failed: %v", err)
	}
	
	return nil
}

// Validate validates server settings
func (s *ServerSettings) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("server name cannot be empty")
	}
	if s.Version == "" {
		return fmt.Errorf("server version cannot be empty")
	}
	if s.Port <= 0 || s.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535, got %d", s.Port)
	}
	if s.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}
	if s.Timeout <= 0 {
		return fmt.Errorf("server timeout must be positive, got %v", s.Timeout)
	}
	return nil
}

// Validate validates storage configuration
func (s *MultiViewStorageConfig) Validate() error {
	if err := s.VectorStore.Validate(); err != nil {
		return fmt.Errorf("vector store config validation failed: %v", err)
	}
	if err := s.GraphStore.Validate(); err != nil {
		return fmt.Errorf("graph store config validation failed: %v", err)
	}
	if err := s.SearchIndex.Validate(); err != nil {
		return fmt.Errorf("search index config validation failed: %v", err)
	}
	if s.Timeout <= 0 {
		return fmt.Errorf("storage timeout must be positive, got %v", s.Timeout)
	}
	return nil
}

// Validate validates vector store configuration
func (v *VectorStoreConfig) Validate() error {
	if v.Provider == "" {
		return fmt.Errorf("vector store provider cannot be empty")
	}
	if v.Dimensions <= 0 {
		return fmt.Errorf("vector dimensions must be positive, got %d", v.Dimensions)
	}
	if v.IndexType == "" {
		return fmt.Errorf("vector index type cannot be empty")
	}
	return nil
}

// Validate validates graph store configuration
func (g *GraphStoreConfig) Validate() error {
	if g.Provider == "" {
		return fmt.Errorf("graph store provider cannot be empty")
	}
	if g.Database == "" {
		return fmt.Errorf("graph database name cannot be empty")
	}
	return nil
}

// Validate validates search index configuration
func (s *SearchIndexConfig) Validate() error {
	if s.Provider == "" {
		return fmt.Errorf("search index provider cannot be empty")
	}
	if s.IndexName == "" {
		return fmt.Errorf("search index name cannot be empty")
	}
	return nil
}

// Validate validates processing configuration
func (p *ProcessingConfig) Validate() error {
	if p.EmbeddingModel == "" {
		return fmt.Errorf("embedding model cannot be empty")
	}
	if p.EmbeddingProvider == "" {
		return fmt.Errorf("embedding provider cannot be empty")
	}
	if p.MaxChunkSize <= 0 {
		return fmt.Errorf("max chunk size must be positive, got %d", p.MaxChunkSize)
	}
	if p.ChunkOverlap < 0 {
		return fmt.Errorf("chunk overlap cannot be negative, got %d", p.ChunkOverlap)
	}
	if p.ChunkOverlap >= p.MaxChunkSize {
		return fmt.Errorf("chunk overlap (%d) must be less than max chunk size (%d)", p.ChunkOverlap, p.MaxChunkSize)
	}
	if p.MinConfidence < 0 || p.MinConfidence > 1 {
		return fmt.Errorf("min confidence must be between 0 and 1, got %f", p.MinConfidence)
	}
	return nil
}

// Validate validates performance configuration
func (p *PerformanceConfig) Validate() error {
	if p.MaxConcurrentRequests <= 0 {
		return fmt.Errorf("max concurrent requests must be positive, got %d", p.MaxConcurrentRequests)
	}
	if p.DefaultTimeout <= 0 {
		return fmt.Errorf("default timeout must be positive, got %v", p.DefaultTimeout)
	}
	if p.CacheSize < 0 {
		return fmt.Errorf("cache size cannot be negative, got %d", p.CacheSize)
	}
	if p.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive, got %d", p.BatchSize)
	}
	if p.WorkerPoolSize <= 0 {
		return fmt.Errorf("worker pool size must be positive, got %d", p.WorkerPoolSize)
	}
	return nil
}

// Validate validates logging configuration
func (l *LoggingConfig) Validate() error {
	validLevels := []string{"debug", "info", "warn", "error", "fatal"}
	validLevel := false
	for _, level := range validLevels {
		if l.Level == level {
			validLevel = true
			break
		}
	}
	if !validLevel {
		return fmt.Errorf("invalid log level: %s, must be one of %v", l.Level, validLevels)
	}
	
	validFormats := []string{"json", "text"}
	validFormat := false
	for _, format := range validFormats {
		if l.Format == format {
			validFormat = true
			break
		}
	}
	if !validFormat {
		return fmt.Errorf("invalid log format: %s, must be one of %v", l.Format, validFormats)
	}
	
	if l.Output == "" {
		return fmt.Errorf("log output cannot be empty")
	}
	if l.MaxSize <= 0 {
		return fmt.Errorf("log max size must be positive, got %d", l.MaxSize)
	}
	if l.MaxBackups < 0 {
		return fmt.Errorf("log max backups cannot be negative, got %d", l.MaxBackups)
	}
	if l.MaxAge <= 0 {
		return fmt.Errorf("log max age must be positive, got %d", l.MaxAge)
	}
	return nil
}