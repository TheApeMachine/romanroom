package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// AgenticMemoryServer wraps the MCP server with memory engine capabilities
type AgenticMemoryServer struct {
	server        *mcp.Server
	config        *ServerConfig
	recallHandler *RecallHandler
	writeHandler  *WriteHandler
	mu            sync.RWMutex
	isRunning     bool
	shutdownChan  chan struct{}
}

// NewAgenticMemoryServer creates a new MCP server with memory capabilities
func NewAgenticMemoryServer(config *ServerConfig) (*AgenticMemoryServer, error) {
	if config == nil {
		return nil, fmt.Errorf("server config cannot be nil")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid server config: %w", err)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    config.Server.Name,
		Version: config.Server.Version,
	}, nil)

	ams := &AgenticMemoryServer{
		server:       server,
		config:       config,
		shutdownChan: make(chan struct{}),
	}

	// Initialize handlers
	queryProcessor := NewQueryProcessor(nil)
	resultFuser := NewResultFuser()
	ams.recallHandler = NewRecallHandler(queryProcessor, resultFuser)

	// Initialize write handler
	contentProcessor := NewContentProcessor()
	// Initialize storage components
	vectorStore := NewMockVectorStore()
	graphStore := NewMockGraphStore()
	searchIndex := NewMockSearchIndex()
	storageConfig := &MultiViewStorageConfig{
		VectorStore: VectorStoreConfig{
			Provider:   "mock",
			Dimensions: 1536,
		},
		GraphStore: GraphStoreConfig{
			Provider: "mock",
		},
		SearchIndex: SearchIndexConfig{
			Provider: "mock",
		},
		Timeout: 10 * time.Second,
	}
	storage := NewMultiViewStorage(vectorStore, graphStore, searchIndex, storageConfig)
	memoryWriter := NewMemoryWriter(storage, contentProcessor, nil)
	ams.writeHandler = NewWriteHandler(memoryWriter, contentProcessor)

	// Register MCP tools
	if err := ams.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	return ams, nil
}

// Start starts the MCP server
func (ams *AgenticMemoryServer) Start(ctx context.Context) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if ams.isRunning {
		return fmt.Errorf("server is already running")
	}

	log.Printf("Starting Agentic Memory Server v1.0.0")
	log.Printf("Server config: %+v", ams.config)

	// Create a new shutdown channel for this start cycle
	ams.shutdownChan = make(chan struct{})
	ams.isRunning = true
	return nil
}

// Stop stops the MCP server
func (ams *AgenticMemoryServer) Stop(ctx context.Context) error {
	ams.mu.Lock()
	defer ams.mu.Unlock()

	if !ams.isRunning {
		return fmt.Errorf("server is not running")
	}

	log.Printf("Stopping Agentic Memory Server")

	// Signal shutdown
	close(ams.shutdownChan)
	ams.isRunning = false

	return nil
}

// IsRunning returns whether the server is currently running
func (ams *AgenticMemoryServer) IsRunning() bool {
	ams.mu.RLock()
	defer ams.mu.RUnlock()
	return ams.isRunning
}

// Run starts the MCP server using the specified transport
func (ams *AgenticMemoryServer) Run(ctx context.Context, transport mcp.Transport) error {
	if err := ams.Start(ctx); err != nil {
		return err
	}

	defer func() {
		if err := ams.Stop(ctx); err != nil {
			log.Printf("Error stopping server: %v", err)
		}
	}()

	return ams.server.Run(ctx, transport)
}

// RunHTTP starts the server as an HTTP handler
func (ams *AgenticMemoryServer) RunHTTP(addr string) error {
	ctx := context.Background()
	if err := ams.Start(ctx); err != nil {
		return err
	}

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return ams.server
	}, nil)

	log.Printf("Starting HTTP server on %s", addr)
	return http.ListenAndServe(addr, handler)
}

// GetServer returns the underlying MCP server for testing
func (ams *AgenticMemoryServer) GetServer() *mcp.Server {
	return ams.server
}

// GetConfig returns the server configuration
func (ams *AgenticMemoryServer) GetConfig() *ServerConfig {
	return ams.config
}
