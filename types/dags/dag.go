package dags

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
)

// Graph represents a directed acyclic graph.
type Graph[T cmp.Ordered] map[T][]T

func (g Graph[T]) Map() map[T][]T {
	return g
}

func (g Graph[T]) Invert() Graph[T] {
	inv := make(Graph[T])
	for node, edges := range g {
		if _, ok := inv[node]; !ok {
			inv[node] = []T{}
		}
		for _, edge := range edges {
			inv[edge] = append(inv[edge], node)
		}
	}

	return inv
}

// AddEdge adds a directed edge from 'from' to 'to'.
func (g Graph[T]) AddEdge(from, to T) {
	if _, exists := g[from]; !exists {
		g[from] = []T{}
	}
	if _, exists := g[to]; !exists {
		g[to] = []T{}
	}
	g[from] = append(g[from], to)
}

// TopologicalSort performs a topological sort using Kahn's algorithm.
func (g Graph[T]) TopologicalSort() ([]T, error) {
	inDegrees := make(map[T]int)
	for node, edges := range g {
		if _, ok := inDegrees[node]; !ok {
			inDegrees[node] = 0
		}
		for _, edge := range edges {
			inDegrees[edge]++
		}
	}

	var queue []T
	// Find all vertices with an in-degree of 0 and add them to the queue
	for node := range inDegrees {
		if inDegrees[node] == 0 {
			queue = append(queue, node)
		}
	}

	var result []T
	for len(queue) > 0 {
		// Dequeue a vertex and add it to the result
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Decrease the in-degree of all adjacent vertices
		for _, neighbor := range g[current] {
			inDegrees[neighbor]--
			// If a neighbor's in-degree becomes 0, enqueue it
			if inDegrees[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for a cycle
	if len(result) != len(inDegrees) {
		return nil, fmt.Errorf("graph contains a cycle, topological sort not possible")
	}

	return result, nil
}

func (g Graph[T]) Vertices() []T {
	nodes := make(map[T]struct{})
	for node, edges := range g {
		nodes[node] = struct{}{}
		for _, edge := range edges {
			nodes[edge] = struct{}{}
		}
	}

	return slices.Sorted(maps.Keys(nodes))
}
