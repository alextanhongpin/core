# dags

A directed graph as an adjacency map, with deterministic topological sorting
and cycle detection.

**Use it for:** ordering tasks, migrations, or build steps by their
dependencies.

**Don't use it for:** weighted graphs, shortest paths, or very large graphs
where the map-of-slices representation is too costly.

## Installation

```sh
go get github.com/alextanhongpin/core/types/dags
```

This package is part of the `types` module; check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/types/dags"
)

func main() {
	g := make(dags.Graph[string])
	g.AddEdge("migrate", "serve")
	g.AddEdge("config", "migrate")
	g.AddEdge("config", "serve")

	order, err := g.TopologicalSort()
	fmt.Println(order, err)

	g.AddEdge("serve", "config")
	_, err = g.TopologicalSort()
	fmt.Println(errors.Is(err, dags.ErrCycle))
}
```

It prints `[config migrate serve] <nil>`, then `true`.

## API

```go
var ErrCycle error

type Graph[T cmp.Ordered] map[T][]T // g[from] lists the vertices from points to

func (g Graph[T]) AddEdge(from, to T)
func (g Graph[T]) TopologicalSort() ([]T, error)
func (g Graph[T]) Vertices() []T     // ascending
func (g Graph[T]) Invert() Graph[T]  // new graph, edges reversed
```

## Behavior and guarantees

- `Graph` is a map: create it with `make` or a literal before `AddEdge`. It can
  also be decoded from JSON such as `{"a": ["b"], "b": []}`.
- A vertex that only appears as an edge target is still a vertex.
- `TopologicalSort` uses Kahn's algorithm. Vertices with no incoming edges
  start in ascending order; the rest follow in the order they become ready, by
  edge-list order. The same graph always gives the same order.
- It returns `ErrCycle` for any cycle, including a self-loop. Duplicate edges
  are allowed.
- `Invert` lists every vertex of `g` as a key and sorts each edge list.
- Not safe for concurrent writes.

## Testing

```sh
go vet ./dags/
go test -race -count=1 ./dags/
```

Run from the `types` directory.
