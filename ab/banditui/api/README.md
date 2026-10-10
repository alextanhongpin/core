# api

Register an HTTP endpoint for experiment results.

## Installation

```sh
go get github.com/alextanhongpin/core/ab/banditui/api
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"github.com/alextanhongpin/core/ab"
	"github.com/alextanhongpin/core/ab/banditui/api"
	"net/http"
)

func register(mux *http.ServeMux, engine *ab.ExperimentEngine) {
	api.RegisterExperimentResultsAPI(mux, engine)
}

func main() {}
```

## Behavior and limits

GET /api/experiments/{id}/results encodes results as JSON. Other methods return 405, malformed paths return 404, and engine errors return 404 with a JSON error body. Registration stores the engine in package-global state; multiple registrations share the most recently assigned engine. Pass a non-nil engine. Authentication must be provided by the surrounding HTTP service.
