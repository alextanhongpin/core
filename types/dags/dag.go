package dags

import (
	"cmp"
	"errors"
	"maps"
	"slices"
)

// ErrCycle is returned by TopologicalSort when the graph contains a cycle.
var ErrCycle = errors.New("dags: graph contains a cycle")

// Graph is an adjacency list: g[from] lists the vertices that from points to.
// A vertex that appears only as an edge target is still a vertex. Graph is a
// map, so create it with make or a literal before calling AddEdge.
type Graph[T cmp.Ordered] map[T][]T

// Invert returns a new graph with every edge reversed. Each vertex of g is a
// key of the result, and each edge list is sorted.
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
	for _, edges := range inv {
		slices.Sort(edges)
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

// TopologicalSort orders the vertices so that every edge points from an
// earlier vertex to a later one, using Kahn's algorithm. Vertices with no
// incoming edges start in ascending order; later vertices follow in the order
// they become ready, by edge-list order. The result is deterministic. It
// returns ErrCycle if the graph has a cycle, including a self-loop.
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
	slices.Sort(queue)

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
		return nil, ErrCycle
	}

	return result, nil
}

// Vertices returns every vertex, keys and edge targets, in ascending order.
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
