#  cores


Useful collection of dependencies required to build microservices.


WIP:
- add suggested folder structure for APIs, domain layers etc.


```go
package main

import (
	"errors"
	"net/http"
	"os"

	"github.com/alextanhongpin/core/http/middleware"
	"github.com/alextanhongpin/core/http/server"
	"github.com/alextanhongpin/core/http/types"
	chi "github.com/go-chi/chi/v5"
	"golang.org/x/exp/slog"
)

func main() {
	textHandler := slog.NewTextHandler(os.Stdout)
	logger := slog.New(textHandler)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.LogRequest(logger))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			types.EncodeError(w, r, errors.New("bad request"))
			return
		}

		type result struct {
			Name string `json:"name"`
		}

		types.EncodeResult(w, http.StatusOK, result{Name: name})
	})

	server.New(logger, r, 8080)
}
```
