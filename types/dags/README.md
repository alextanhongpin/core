# dags

Represent directed graphs and order dependencies with a topological sort.

## Installation

```sh
go get github.com/alextanhongpin/core/types/dags
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/dags"
)

func main() {
	g := make(dags.Graph[string])
	g.AddEdge("build", "deploy")
	order, err := g.TopologicalSort()
	if err != nil {
		panic(err)
	}
	fmt.Println(order)
}
```

## Behavior and limits

Initialize the map before AddEdge. AddEdge also creates the destination vertex. TopologicalSort returns an error for cycles. Vertices returns sorted vertex names; Invert reverses edges. Map exposes the underlying map rather than a copy.
