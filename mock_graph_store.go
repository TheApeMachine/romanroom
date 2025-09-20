package main

import (
	"context"
	"fmt"
	"math"
	"sync"
)

// MockGraphStore provides an in-memory implementation of GraphStore for testing
type MockGraphStore struct {
	mu           sync.RWMutex
	nodes        map[string]*Node
	edges        map[string]*Edge
	adjacencyOut map[string][]string // outgoing edges from node
	adjacencyIn  map[string][]string // incoming edges to node
	closed       bool
	healthy      bool
}

// NewMockGraphStore creates a new mock graph store
func NewMockGraphStore() *MockGraphStore {
	return &MockGraphStore{
		nodes:        make(map[string]*Node),
		edges:        make(map[string]*Edge),
		adjacencyOut: make(map[string][]string),
		adjacencyIn:  make(map[string][]string),
		healthy:      true,
	}
}

// GetNodes returns all nodes for testing
func (m *MockGraphStore) GetNodes() map[string]*Node {
	return m.nodes
}

// GetEdges returns all edges for testing
func (m *MockGraphStore) GetEdges() []*Edge {
	var edges []*Edge
	for _, edge := range m.edges {
		edges = append(edges, edge)
	}
	return edges
}

// CreateNode creates a new node in the graph
func (m *MockGraphStore) CreateNode(ctx context.Context, node *Node) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("graph store is closed")
	}
	
	if _, exists := m.nodes[node.ID]; exists {
		return fmt.Errorf("node with ID %s already exists", node.ID)
	}
	
	m.nodes[node.ID] = node
	m.adjacencyOut[node.ID] = []string{}
	m.adjacencyIn[node.ID] = []string{}
	
	return nil
}

// GetNode retrieves a node by ID
func (m *MockGraphStore) GetNode(ctx context.Context, id string) (*Node, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	if node, exists := m.nodes[id]; exists {
		return node, nil
	}
	
	return nil, fmt.Errorf("node with ID %s not found", id)
}

// UpdateNode updates an existing node
func (m *MockGraphStore) UpdateNode(ctx context.Context, node *Node) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("graph store is closed")
	}
	
	if _, exists := m.nodes[node.ID]; !exists {
		return fmt.Errorf("node with ID %s not found", node.ID)
	}
	
	m.nodes[node.ID] = node
	return nil
}

// DeleteNode removes a node and all its edges
func (m *MockGraphStore) DeleteNode(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("graph store is closed")
	}
	
	if _, exists := m.nodes[id]; !exists {
		return fmt.Errorf("node with ID %s not found", id)
	}
	
	// Remove all edges connected to this node
	for _, edgeID := range m.adjacencyOut[id] {
		delete(m.edges, edgeID)
	}
	for _, edgeID := range m.adjacencyIn[id] {
		delete(m.edges, edgeID)
	}
	
	// Clean up adjacency lists
	delete(m.nodes, id)
	delete(m.adjacencyOut, id)
	delete(m.adjacencyIn, id)
	
	return nil
}

// CreateEdge creates a new edge in the graph
func (m *MockGraphStore) CreateEdge(ctx context.Context, edge *Edge) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("graph store is closed")
	}
	
	if _, exists := m.edges[edge.ID]; exists {
		return fmt.Errorf("edge with ID %s already exists", edge.ID)
	}
	
	// Verify nodes exist
	if _, exists := m.nodes[edge.From]; !exists {
		return fmt.Errorf("source node %s not found", edge.From)
	}
	if _, exists := m.nodes[edge.To]; !exists {
		return fmt.Errorf("target node %s not found", edge.To)
	}
	
	m.edges[edge.ID] = edge
	m.adjacencyOut[edge.From] = append(m.adjacencyOut[edge.From], edge.ID)
	m.adjacencyIn[edge.To] = append(m.adjacencyIn[edge.To], edge.ID)
	
	return nil
}

// GetEdge retrieves an edge by ID
func (m *MockGraphStore) GetEdge(ctx context.Context, id string) (*Edge, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	if edge, exists := m.edges[id]; exists {
		return edge, nil
	}
	
	return nil, fmt.Errorf("edge with ID %s not found", id)
}

// UpdateEdge updates an existing edge
func (m *MockGraphStore) UpdateEdge(ctx context.Context, edge *Edge) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("graph store is closed")
	}
	
	if _, exists := m.edges[edge.ID]; !exists {
		return fmt.Errorf("edge with ID %s not found", edge.ID)
	}
	
	m.edges[edge.ID] = edge
	return nil
}

// DeleteEdge removes an edge from the graph
func (m *MockGraphStore) DeleteEdge(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return fmt.Errorf("graph store is closed")
	}
	
	edge, exists := m.edges[id]
	if !exists {
		return fmt.Errorf("edge with ID %s not found", id)
	}
	
	// Remove from adjacency lists
	m.removeFromSlice(m.adjacencyOut[edge.From], id)
	m.removeFromSlice(m.adjacencyIn[edge.To], id)
	
	delete(m.edges, id)
	return nil
}

// FindPaths finds paths between two nodes
func (m *MockGraphStore) FindPaths(ctx context.Context, from, to string, options GraphTraversalOptions) ([]Path, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	// Simple BFS implementation for finding paths
	var paths []Path
	visited := make(map[string]bool)
	
	var dfs func(current string, target string, currentPath []string, currentEdges []string, depth int)
	dfs = func(current string, target string, currentPath []string, currentEdges []string, depth int) {
		if depth > options.MaxDepth {
			return
		}
		
		if current == target {
			paths = append(paths, Path{
				Nodes: append([]string{}, currentPath...),
				Edges: append([]string{}, currentEdges...),
				Cost:  float64(len(currentPath) - 1),
			})
			return
		}
		
		if visited[current] {
			return
		}
		visited[current] = true
		
		for _, edgeID := range m.adjacencyOut[current] {
			edge := m.edges[edgeID]
			if m.matchesEdgeFilters(edge, options.EdgeTypes) {
				dfs(edge.To, target, append(currentPath, edge.To), append(currentEdges, edgeID), depth+1)
			}
		}
		
		visited[current] = false
	}
	
	dfs(from, to, []string{from}, []string{}, 0)
	
	// Limit results
	if options.MaxResults > 0 && len(paths) > options.MaxResults {
		paths = paths[:options.MaxResults]
	}
	
	return paths, nil
}

// GetNeighbors returns neighboring nodes
func (m *MockGraphStore) GetNeighbors(ctx context.Context, nodeID string, options GraphTraversalOptions) ([]Node, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	var neighbors []Node
	visited := make(map[string]bool)
	
	for _, edgeID := range m.adjacencyOut[nodeID] {
		edge := m.edges[edgeID]
		if !visited[edge.To] && m.matchesEdgeFilters(edge, options.EdgeTypes) {
			if node, exists := m.nodes[edge.To]; exists {
				neighbors = append(neighbors, *node)
				visited[edge.To] = true
			}
		}
	}
	
	return neighbors, nil
}

// PageRank implements a simple PageRank algorithm
func (m *MockGraphStore) PageRank(ctx context.Context, options PageRankOptions) (map[string]float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	scores := make(map[string]float64)
	newScores := make(map[string]float64)
	
	// Initialize scores
	for nodeID := range m.nodes {
		scores[nodeID] = 1.0
	}
	
	// Iterate PageRank algorithm
	for iter := 0; iter < options.MaxIter; iter++ {
		for nodeID := range m.nodes {
			newScores[nodeID] = (1.0 - options.Alpha)
		}
		
		for nodeID := range m.nodes {
			outDegree := len(m.adjacencyOut[nodeID])
			if outDegree > 0 {
				contribution := options.Alpha * scores[nodeID] / float64(outDegree)
				for _, edgeID := range m.adjacencyOut[nodeID] {
					edge := m.edges[edgeID]
					newScores[edge.To] += contribution
				}
			}
		}
		
		// Check convergence
		converged := true
		for nodeID := range m.nodes {
			if math.Abs(newScores[nodeID]-scores[nodeID]) > options.Tolerance {
				converged = false
				break
			}
		}
		
		scores, newScores = newScores, scores
		
		if converged {
			break
		}
	}
	
	return scores, nil
}

// CommunityDetection implements a simple community detection algorithm
func (m *MockGraphStore) CommunityDetection(ctx context.Context) ([]Community, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	// Simple connected components as communities
	visited := make(map[string]bool)
	var communities []Community
	communityID := 0
	
	for nodeID := range m.nodes {
		if !visited[nodeID] {
			var component []string
			m.dfsComponent(nodeID, visited, &component)
			
			if len(component) > 0 {
				communities = append(communities, Community{
					ID:    fmt.Sprintf("community_%d", communityID),
					Nodes: component,
					Score: float64(len(component)),
				})
				communityID++
			}
		}
	}
	
	return communities, nil
}

// ShortestPath finds the shortest path between two nodes
func (m *MockGraphStore) ShortestPath(ctx context.Context, from, to string) (*Path, error) {
	paths, err := m.FindPaths(ctx, from, to, GraphTraversalOptions{
		MaxDepth:   10,
		MaxResults: 1,
	})
	
	if err != nil {
		return nil, err
	}
	
	if len(paths) == 0 {
		return nil, fmt.Errorf("no path found from %s to %s", from, to)
	}
	
	return &paths[0], nil
}

// BatchCreateNodes creates multiple nodes
func (m *MockGraphStore) BatchCreateNodes(ctx context.Context, nodes []*Node) error {
	for _, node := range nodes {
		if err := m.CreateNode(ctx, node); err != nil {
			return err
		}
	}
	return nil
}

// BatchCreateEdges creates multiple edges
func (m *MockGraphStore) BatchCreateEdges(ctx context.Context, edges []*Edge) error {
	for _, edge := range edges {
		if err := m.CreateEdge(ctx, edge); err != nil {
			return err
		}
	}
	return nil
}

// FindNodesByType finds nodes by type
func (m *MockGraphStore) FindNodesByType(ctx context.Context, nodeType NodeType, filters map[string]interface{}) ([]*Node, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	var result []*Node
	for _, node := range m.nodes {
		if node.Type == nodeType && m.matchesNodeFilters(node, filters) {
			result = append(result, node)
		}
	}
	
	return result, nil
}

// FindEdgesByType finds edges by type
func (m *MockGraphStore) FindEdgesByType(ctx context.Context, edgeType EdgeType, filters map[string]interface{}) ([]*Edge, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, fmt.Errorf("graph store is closed")
	}
	
	var result []*Edge
	for _, edge := range m.edges {
		if edge.Type == edgeType && m.matchesEdgePropertiesFilters(edge, filters) {
			result = append(result, edge)
		}
	}
	
	return result, nil
}

// NodeCount returns the number of nodes
func (m *MockGraphStore) NodeCount(ctx context.Context) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return 0, fmt.Errorf("graph store is closed")
	}
	
	if !m.healthy {
		return 0, fmt.Errorf("graph store is unhealthy")
	}
	
	return int64(len(m.nodes)), nil
}

// EdgeCount returns the number of edges
func (m *MockGraphStore) EdgeCount(ctx context.Context) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return 0, fmt.Errorf("graph store is closed")
	}
	
	if !m.healthy {
		return 0, fmt.Errorf("graph store is unhealthy")
	}
	
	return int64(len(m.edges)), nil
}

// Close closes the graph store
func (m *MockGraphStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.closed = true
	return nil
}

// Health checks if the graph store is healthy
func (m *MockGraphStore) Health(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return fmt.Errorf("graph store is closed")
	}
	
	if !m.healthy {
		return fmt.Errorf("graph store is unhealthy")
	}
	
	return nil
}

// SetHealthy sets the health status for testing
func (m *MockGraphStore) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
}

// Helper methods

func (m *MockGraphStore) removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func (m *MockGraphStore) matchesEdgeFilters(edge *Edge, edgeTypes []EdgeType) bool {
	if len(edgeTypes) == 0 {
		return true
	}
	
	for _, edgeType := range edgeTypes {
		if edge.Type == edgeType {
			return true
		}
	}
	
	return false
}

func (m *MockGraphStore) matchesNodeFilters(node *Node, filters map[string]interface{}) bool {
	if filters == nil {
		return true
	}
	
	for key, expectedValue := range filters {
		if actualValue, exists := node.Properties[key]; !exists || actualValue != expectedValue {
			return false
		}
	}
	
	return true
}

func (m *MockGraphStore) matchesEdgePropertiesFilters(edge *Edge, filters map[string]interface{}) bool {
	if filters == nil {
		return true
	}
	
	for key, expectedValue := range filters {
		if actualValue, exists := edge.Properties[key]; !exists || actualValue != expectedValue {
			return false
		}
	}
	
	return true
}

func (m *MockGraphStore) dfsComponent(nodeID string, visited map[string]bool, component *[]string) {
	visited[nodeID] = true
	*component = append(*component, nodeID)
	
	for _, edgeID := range m.adjacencyOut[nodeID] {
		edge := m.edges[edgeID]
		if !visited[edge.To] {
			m.dfsComponent(edge.To, visited, component)
		}
	}
	
	for _, edgeID := range m.adjacencyIn[nodeID] {
		edge := m.edges[edgeID]
		if !visited[edge.From] {
			m.dfsComponent(edge.From, visited, component)
		}
	}
}