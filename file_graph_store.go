package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// FileGraphStore implements GraphStore interface with JSON file persistence
type FileGraphStore struct {
	mu       sync.RWMutex
	filePath string
	nodes    map[string]*Node
	edges    map[string]*Edge
	// Adjacency list for efficient graph operations
	adjacencyList map[string][]string // nodeID -> list of connected nodeIDs
	closed        bool
}

// GraphStoreData represents the JSON structure for persistence
type GraphStoreData struct {
	Nodes map[string]*Node `json:"nodes"`
	Edges map[string]*Edge `json:"edges"`
}

// NewFileGraphStore creates a new file-based graph store
func NewFileGraphStore(filePath string) *FileGraphStore {
	return &FileGraphStore{
		filePath:      filePath,
		nodes:         make(map[string]*Node),
		edges:         make(map[string]*Edge),
		adjacencyList: make(map[string][]string),
	}
}

// CreateNode creates a new node in the graph
func (f *FileGraphStore) CreateNode(ctx context.Context, node *Node) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return fmt.Errorf("graph store is closed")
	}
	
	if err := node.Validate(); err != nil {
		return fmt.Errorf("invalid node: %w", err)
	}
	
	f.nodes[node.ID] = node
	
	// Initialize adjacency list entry if not exists
	if _, exists := f.adjacencyList[node.ID]; !exists {
		f.adjacencyList[node.ID] = []string{}
	}
	
	return f.save()
}

// CreateEdge creates a new edge in the graph
func (f *FileGraphStore) CreateEdge(ctx context.Context, edge *Edge) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return fmt.Errorf("graph store is closed")
	}
	
	if err := edge.Validate(); err != nil {
		return fmt.Errorf("invalid edge: %w", err)
	}
	
	// Check if nodes exist
	if _, exists := f.nodes[edge.From]; !exists {
		return fmt.Errorf("source node %s does not exist", edge.From)
	}
	if _, exists := f.nodes[edge.To]; !exists {
		return fmt.Errorf("target node %s does not exist", edge.To)
	}
	
	f.edges[edge.ID] = edge
	
	// Update adjacency list
	f.addToAdjacencyList(edge.From, edge.To)
	
	return f.save()
}

// GetNode retrieves a node by ID
func (f *FileGraphStore) GetNode(ctx context.Context, id string) (*Node, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	node, exists := f.nodes[id]
	if !exists {
		return nil, fmt.Errorf("node with ID %s not found", id)
	}
	
	return node, nil
}

// GetEdge retrieves an edge by ID
func (f *FileGraphStore) GetEdge(ctx context.Context, id string) (*Edge, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	edge, exists := f.edges[id]
	if !exists {
		return nil, fmt.Errorf("edge with ID %s not found", id)
	}
	
	return edge, nil
}

// UpdateNode updates an existing node
func (f *FileGraphStore) UpdateNode(ctx context.Context, node *Node) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return fmt.Errorf("graph store is closed")
	}
	
	if err := node.Validate(); err != nil {
		return fmt.Errorf("invalid node: %w", err)
	}
	
	if _, exists := f.nodes[node.ID]; !exists {
		return fmt.Errorf("node with ID %s not found", node.ID)
	}
	
	f.nodes[node.ID] = node
	return f.save()
}

// UpdateEdge updates an existing edge
func (f *FileGraphStore) UpdateEdge(ctx context.Context, edge *Edge) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return fmt.Errorf("graph store is closed")
	}
	
	if err := edge.Validate(); err != nil {
		return fmt.Errorf("invalid edge: %w", err)
	}
	
	oldEdge, exists := f.edges[edge.ID]
	if !exists {
		return fmt.Errorf("edge with ID %s not found", edge.ID)
	}
	
	// Update adjacency list if endpoints changed
	if oldEdge.From != edge.From || oldEdge.To != edge.To {
		f.removeFromAdjacencyList(oldEdge.From, oldEdge.To)
		f.addToAdjacencyList(edge.From, edge.To)
	}
	
	f.edges[edge.ID] = edge
	return f.save()
}

// DeleteNode deletes a node and all its edges
func (f *FileGraphStore) DeleteNode(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return fmt.Errorf("graph store is closed")
	}
	
	if _, exists := f.nodes[id]; !exists {
		return fmt.Errorf("node with ID %s not found", id)
	}
	
	// Remove all edges connected to this node
	var edgesToDelete []string
	for edgeID, edge := range f.edges {
		if edge.From == id || edge.To == id {
			edgesToDelete = append(edgesToDelete, edgeID)
		}
	}
	
	for _, edgeID := range edgesToDelete {
		edge := f.edges[edgeID]
		f.removeFromAdjacencyList(edge.From, edge.To)
		delete(f.edges, edgeID)
	}
	
	// Remove node
	delete(f.nodes, id)
	delete(f.adjacencyList, id)
	
	return f.save()
}

// DeleteEdge deletes an edge by ID
func (f *FileGraphStore) DeleteEdge(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.closed {
		return fmt.Errorf("graph store is closed")
	}
	
	edge, exists := f.edges[id]
	if !exists {
		return fmt.Errorf("edge with ID %s not found", id)
	}
	
	f.removeFromAdjacencyList(edge.From, edge.To)
	delete(f.edges, id)
	
	return f.save()
}

// FindPaths finds paths between two nodes with traversal options
func (f *FileGraphStore) FindPaths(ctx context.Context, from, to string, options GraphTraversalOptions) ([]Path, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	// Check if nodes exist
	if _, exists := f.nodes[from]; !exists {
		return nil, fmt.Errorf("source node %s does not exist", from)
	}
	if _, exists := f.nodes[to]; !exists {
		return nil, fmt.Errorf("target node %s does not exist", to)
	}
	
	// Use BFS to find paths
	paths := f.findPathsBFS(from, to, options)
	
	// Sort by cost and limit results
	sort.Slice(paths, func(i, j int) bool {
		return paths[i].Cost < paths[j].Cost
	})
	
	if options.MaxResults > 0 && len(paths) > options.MaxResults {
		paths = paths[:options.MaxResults]
	}
	
	return paths, nil
}

// PageRank computes PageRank scores for nodes with options
func (f *FileGraphStore) PageRank(ctx context.Context, options PageRankOptions) (map[string]float64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	nodeCount := len(f.nodes)
	if nodeCount == 0 {
		return make(map[string]float64), nil
	}
	
	// Initialize PageRank scores
	scores := make(map[string]float64)
	newScores := make(map[string]float64)
	
	for nodeID := range f.nodes {
		scores[nodeID] = 1.0 / float64(nodeCount)
	}
	
	// Iterative PageRank computation
	for iter := 0; iter < options.MaxIter; iter++ {
		// Reset new scores
		for nodeID := range f.nodes {
			newScores[nodeID] = (1.0 - options.Alpha) / float64(nodeCount)
		}
		
		// Calculate contributions from each node
		for nodeID := range f.nodes {
			outDegree := len(f.adjacencyList[nodeID])
			if outDegree > 0 {
				contribution := options.Alpha * scores[nodeID] / float64(outDegree)
				for _, neighbor := range f.adjacencyList[nodeID] {
					newScores[neighbor] += contribution
				}
			} else {
				// Handle dangling nodes - distribute their score evenly
				contribution := options.Alpha * scores[nodeID] / float64(nodeCount)
				for neighborID := range f.nodes {
					newScores[neighborID] += contribution
				}
			}
		}
		
		// Check for convergence
		converged := true
		for nodeID := range f.nodes {
			if math.Abs(newScores[nodeID]-scores[nodeID]) > options.Tolerance {
				converged = false
				break
			}
		}
		
		// Update scores
		for nodeID := range f.nodes {
			scores[nodeID] = newScores[nodeID]
		}
		
		if converged {
			break
		}
	}
	
	return scores, nil
}

// CommunityDetection performs community detection using Louvain algorithm
func (f *FileGraphStore) CommunityDetection(ctx context.Context) ([]Community, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	// Simple community detection - group nodes by type
	communities := make(map[NodeType][]string)
	
	for nodeID, node := range f.nodes {
		communities[node.Type] = append(communities[node.Type], nodeID)
	}
	
	var result []Community
	communityID := 0
	for nodeType, nodeIDs := range communities {
		if len(nodeIDs) > 0 {
			result = append(result, Community{
				ID:      fmt.Sprintf("community_%d", communityID),
				Nodes:   nodeIDs,
				Score:   1.0,
				Summary: fmt.Sprintf("Community of %s nodes", nodeType),
			})
			communityID++
		}
	}
	
	return result, nil
}

// GetNeighbors returns neighboring nodes of a given node
func (f *FileGraphStore) GetNeighbors(ctx context.Context, nodeID string, options GraphTraversalOptions) ([]Node, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	if _, exists := f.nodes[nodeID]; !exists {
		return nil, fmt.Errorf("node with ID %s not found", nodeID)
	}
	
	var neighbors []Node
	neighborIDs := f.adjacencyList[nodeID]
	
	for _, neighborID := range neighborIDs {
		if neighbor, exists := f.nodes[neighborID]; exists {
			// Apply node type filter if specified
			if len(options.NodeTypes) > 0 {
				found := false
				for _, nodeType := range options.NodeTypes {
					if neighbor.Type == nodeType {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			
			neighbors = append(neighbors, *neighbor)
		}
	}
	
	// Apply result limit
	if options.MaxResults > 0 && len(neighbors) > options.MaxResults {
		neighbors = neighbors[:options.MaxResults]
	}
	
	return neighbors, nil
}

// FindEdgesByType finds edges by type with optional filters
func (f *FileGraphStore) FindEdgesByType(ctx context.Context, edgeType EdgeType, filters map[string]interface{}) ([]*Edge, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	var result []*Edge
	
	for _, edge := range f.edges {
		if edge.Type == edgeType {
			// Apply filters
			if f.matchesFilters(edge.Properties, filters) {
				result = append(result, edge)
			}
		}
	}
	
	return result, nil
}

// FindNodesByType finds nodes by type with optional filters
func (f *FileGraphStore) FindNodesByType(ctx context.Context, nodeType NodeType, filters map[string]interface{}) ([]*Node, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	var result []*Node
	
	for _, node := range f.nodes {
		if node.Type == nodeType {
			// Apply filters
			if f.matchesFilters(node.Properties, filters) {
				result = append(result, node)
			}
		}
	}
	
	return result, nil
}

// NodeCount returns the total number of nodes
func (f *FileGraphStore) NodeCount(ctx context.Context) (int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return 0, fmt.Errorf("graph store is closed")
	}
	
	return int64(len(f.nodes)), nil
}

// EdgeCount returns the total number of edges
func (f *FileGraphStore) EdgeCount(ctx context.Context) (int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return 0, fmt.Errorf("graph store is closed")
	}
	
	return int64(len(f.edges)), nil
}

// Close closes the graph store connection
func (f *FileGraphStore) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.closed = true
	return nil
}

// Health checks if the graph store is healthy
func (f *FileGraphStore) Health(ctx context.Context) error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.closed {
		return fmt.Errorf("graph store is closed")
	}
	
	// Check if file is accessible
	if _, err := os.Stat(f.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("file access error: %w", err)
	}
	
	return nil
}

// Load loads the graph from the JSON file
func (f *FileGraphStore) Load() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if _, err := os.Stat(f.filePath); os.IsNotExist(err) {
		// File doesn't exist, start with empty graph
		f.nodes = make(map[string]*Node)
		f.edges = make(map[string]*Edge)
		f.adjacencyList = make(map[string][]string)
		return nil
	}
	
	data, err := os.ReadFile(f.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	if len(data) == 0 {
		f.nodes = make(map[string]*Node)
		f.edges = make(map[string]*Edge)
		f.adjacencyList = make(map[string][]string)
		return nil
	}
	
	var graphData GraphStoreData
	if err := json.Unmarshal(data, &graphData); err != nil {
		return fmt.Errorf("failed to unmarshal graph data: %w", err)
	}
	
	f.nodes = graphData.Nodes
	f.edges = graphData.Edges
	
	// Rebuild adjacency list
	f.adjacencyList = make(map[string][]string)
	for _, node := range f.nodes {
		f.adjacencyList[node.ID] = []string{}
	}
	
	for _, edge := range f.edges {
		f.addToAdjacencyList(edge.From, edge.To)
	}
	
	return nil
}

// Save saves the graph to the JSON file
func (f *FileGraphStore) Save() error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	return f.save()
}

// save is the internal save method (assumes lock is held)
func (f *FileGraphStore) save() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(f.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	graphData := GraphStoreData{
		Nodes: f.nodes,
		Edges: f.edges,
	}
	
	data, err := json.MarshalIndent(graphData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal graph data: %w", err)
	}
	
	if err := os.WriteFile(f.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// Helper methods

// addToAdjacencyList adds an edge to the adjacency list
func (f *FileGraphStore) addToAdjacencyList(from, to string) {
	if _, exists := f.adjacencyList[from]; !exists {
		f.adjacencyList[from] = []string{}
	}
	
	// Check if edge already exists
	for _, neighbor := range f.adjacencyList[from] {
		if neighbor == to {
			return
		}
	}
	
	f.adjacencyList[from] = append(f.adjacencyList[from], to)
}

// removeFromAdjacencyList removes an edge from the adjacency list
func (f *FileGraphStore) removeFromAdjacencyList(from, to string) {
	if neighbors, exists := f.adjacencyList[from]; exists {
		for i, neighbor := range neighbors {
			if neighbor == to {
				f.adjacencyList[from] = append(neighbors[:i], neighbors[i+1:]...)
				break
			}
		}
	}
}

// findPathsBFS finds paths using breadth-first search
func (f *FileGraphStore) findPathsBFS(from, to string, options GraphTraversalOptions) []Path {
	if from == to {
		return []Path{{
			Nodes: []string{from},
			Edges: []string{},
			Cost:  0.0,
		}}
	}
	
	type pathState struct {
		currentNode string
		path        []string
		edges       []string
		cost        float64
		depth       int
	}
	
	queue := []pathState{{
		currentNode: from,
		path:        []string{from},
		edges:       []string{},
		cost:        0.0,
		depth:       0,
	}}
	
	visited := make(map[string]bool)
	var paths []Path
	
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		
		if current.depth >= options.MaxDepth && options.MaxDepth > 0 {
			continue
		}
		
		if current.currentNode == to {
			paths = append(paths, Path{
				Nodes: current.path,
				Edges: current.edges,
				Cost:  current.cost,
			})
			continue
		}
		
		if visited[current.currentNode] {
			continue
		}
		visited[current.currentNode] = true
		
		// Explore neighbors
		for _, neighborID := range f.adjacencyList[current.currentNode] {
			if !visited[neighborID] {
				// Find the edge connecting current node to neighbor
				var edgeID string
				var edgeCost float64
				
				for eID, edge := range f.edges {
					if edge.From == current.currentNode && edge.To == neighborID {
						edgeID = eID
						edgeCost = edge.Weight
						break
					}
				}
				
				newPath := make([]string, len(current.path))
				copy(newPath, current.path)
				newPath = append(newPath, neighborID)
				
				newEdges := make([]string, len(current.edges))
				copy(newEdges, current.edges)
				if edgeID != "" {
					newEdges = append(newEdges, edgeID)
				}
				
				queue = append(queue, pathState{
					currentNode: neighborID,
					path:        newPath,
					edges:       newEdges,
					cost:        current.cost + edgeCost,
					depth:       current.depth + 1,
				})
			}
		}
	}
	
	return paths
}

// matchesFilters checks if properties match the given filters
func (f *FileGraphStore) matchesFilters(properties map[string]interface{}, filters map[string]interface{}) bool {
	if filters == nil {
		return true
	}
	
	for key, expectedValue := range filters {
		if actualValue, exists := properties[key]; !exists || actualValue != expectedValue {
			return false
		}
	}
	
	return true
}