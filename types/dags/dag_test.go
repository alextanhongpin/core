package dags_test

import (
	"encoding/json"
	"testing"

	"github.com/alextanhongpin/core/types/dags"
	"github.com/go-openapi/testify/assert"
)

func TestDAG(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		graph := make(dags.Graph[string])

		// Add edges for a sample graph: A -> B, A -> C, B -> D, C -> D
		graph.AddEdge("A", "B")
		graph.AddEdge("A", "C")
		graph.AddEdge("B", "D")
		graph.AddEdge("C", "D")

		order, err := graph.TopologicalSort()
		is := assert.New(t)
		is.NoError(err)
		is.Equal([]string{"A", "B", "C", "D"}, order)
	})

	t.Run("one node", func(t *testing.T) {
		graph := make(dags.Graph[string])
		graph.AddEdge("A", "")

		order, err := graph.TopologicalSort()
		is := assert.New(t)
		is.NoError(err)
		is.Equal([]string{"A", ""}, order)
	})

	t.Run("no deps", func(t *testing.T) {
		graph := make(dags.Graph[string])
		graph.AddEdge("", "A")

		order, err := graph.TopologicalSort()
		is := assert.New(t)
		is.NoError(err)
		is.Equal([]string{"", "A"}, order)
	})

	t.Run("map", func(t *testing.T) {
		g := dags.Graph[int]{
			0: []int{1, 2},
			2: []int{1},
			1: []int{3, 4},
			3: []int{},
			4: []int{},
		}
		order, err := g.TopologicalSort()

		is := assert.New(t)
		is.NoError(err)
		is.Equal([]int{0, 2, 1, 3, 4}, order)
	})

	t.Run("invert", func(t *testing.T) {
		g := dags.Graph[int]{
			0: []int{1, 2},
			2: []int{1},
			1: []int{3, 4},
			3: []int{},
			4: []int{},
		}
		inv := g.Invert()
		is := assert.New(t)
		is.Equal(dags.Graph[int]{
			0: []int{},
			1: []int{0, 2},
			2: []int{0},
			3: []int{1},
			4: []int{1},
		}, inv)
	})

	t.Run("disconnected", func(t *testing.T) {
		var g dags.Graph[string]
		b := []byte(`{"a": [], "b": [], "c": []}`)
		is := assert.New(t)
		is.NoError(json.Unmarshal(b, &g))

		order, err := g.TopologicalSort()
		is.NoError(err)
		is.Equal([]string{"a", "b", "c"}, order)
	})
}

func TestCycle(t *testing.T) {
	is := assert.New(t)

	g := dags.Graph[string]{"a": {"b"}, "b": {"c"}, "c": {"a"}, "d": {}}
	_, err := g.TopologicalSort()
	is.ErrorIs(err, dags.ErrCycle)

	self := dags.Graph[string]{"a": {"a"}}
	_, err = self.TopologicalSort()
	is.ErrorIs(err, dags.ErrCycle)
}

func TestInvertIsDeterministic(t *testing.T) {
	is := assert.New(t)

	g := dags.Graph[int]{}
	for i := range 20 {
		g.AddEdge(i, 100)
	}
	for range 10 {
		inv := g.Invert()
		is.IsIncreasing(inv[100])
	}
}

func TestVertices(t *testing.T) {
	g := dags.Graph[string]{"b": {"c"}, "a": nil}
	assert.Equal(t, []string{"a", "b", "c"}, g.Vertices())
}
