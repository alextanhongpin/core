package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/http/handler"
	"github.com/alextanhongpin/errors/cause"
	"github.com/alextanhongpin/testdump/httpdump"
)

type HelloRequest struct {
	Name string `json:"name"`
}

func (r HelloRequest) Validate() error {
	return cause.Map{
		"name": cause.Required(r.Name),
	}.Err()
}

type HelloResponse struct {
	Message string `json:"message"`
}

func TestHandler(t *testing.T) {
	hello := func(ctx context.Context, req HelloRequest) (*HelloResponse, error) {
		return &HelloResponse{
			Message: "Hello, " + req.Name,
		}, nil
	}

	t.Run("success", func(t *testing.T) {
		wr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(`{"name":"john"}`))
		r.Header.Set("Content-Type", "application/json")
		hd := httpdump.Handler(t, handler.NewFunc(hello))
		hd.ServeHTTP(wr, r)
	})

	t.Run("failed", func(t *testing.T) {
		wr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(`{}`))
		r.Header.Set("Content-Type", "application/json")
		hd := httpdump.Handler(t, handler.NewFunc(hello))
		hd.ServeHTTP(wr, r)
	})
}
