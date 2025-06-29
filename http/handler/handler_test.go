package handler_test

import (
	"database/sql"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/http/handler"
	"github.com/alextanhongpin/errors/cause"
	"github.com/alextanhongpin/errors/codes"
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

type Controller struct {
	handler.BaseHandler
}

func (c *Controller) Hello(w http.ResponseWriter, r *http.Request) {
	var req HelloRequest
	if err := c.ReadJSON(r, &req); err != nil {
		c.Next(w, r, err)

		return
	}
	if req.Name == "bob" {
		c.Next(w, r, cause.New(codes.NotFound, "user/not_found", "User not found").Wrap(sql.ErrNoRows))
		return
	}

	c.JSON(w, &HelloResponse{
		Message: "Hello, " + req.Name,
	})
}

func TestHandler(t *testing.T) {
	c := new(Controller)
	c.BaseHandler = c.WithLogger(slog.Default())

	t.Run("success", func(t *testing.T) {
		wr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(`{"name":"john"}`))
		r.Header.Set("Content-Type", "application/json")
		hd := httpdump.HandlerFunc(t, c.Hello)
		hd.ServeHTTP(wr, r)
	})

	t.Run("failed", func(t *testing.T) {
		wr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(`{}`))
		r.Header.Set("Content-Type", "application/json")
		hd := httpdump.HandlerFunc(t, c.Hello)
		hd.ServeHTTP(wr, r)
	})

	t.Run("not found", func(t *testing.T) {
		wr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(`{"name":"bob"}`))
		r.Header.Set("Content-Type", "application/json")
		hd := httpdump.HandlerFunc(t, c.Hello)
		hd.ServeHTTP(wr, r)
	})
}
