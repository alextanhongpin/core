# banditstore

Define experiment and result storage interfaces for bandit experiments.

## Installation

```sh
go get github.com/alextanhongpin/core/ab/banditstore
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"context"
	"fmt"
	"github.com/alextanhongpin/core/ab/banditstore"
)

func main() {
	store := banditstore.NewMemoryStore()
	experiments, err := store.ListExperiments(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println(len(experiments))
}
```

## Behavior and limits

Store combines ExperimentStore and ResultStore. NewMemoryStore provides process-local storage with mutex-protected maps. GetExperiment returns ErrNotFound for unknown IDs. CreateExperiment and UpdateExperiment both overwrite by ID. Experiment pointers are stored and returned directly, so mutations need caller coordination. ListResults copies the result slice. The memory implementation does not persist data or enforce context cancellation.
