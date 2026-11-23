package dags_test

import (
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/types/dags"
)

func ExampleGraph_TopologicalSort() {
	g := make(dags.Graph[string])
	g.AddEdge("migrate", "serve")
	g.AddEdge("config", "migrate")
	g.AddEdge("config", "serve")

	order, err := g.TopologicalSort()
	fmt.Println(order, err)

	g.AddEdge("serve", "config")
	_, err = g.TopologicalSort()
	fmt.Println(errors.Is(err, dags.ErrCycle))
	// Output:
	// [config migrate serve] <nil>
	// true
}
