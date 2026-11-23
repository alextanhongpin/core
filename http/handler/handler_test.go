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
	"github.com/alextanhongpin/errors/validator"
)

type HelloRequest struct {
	Name string `json:"name"`
}

func (r HelloRequest) Validate() error {
	return validator.Map(map[string]error{
		"name": validator.Required(r.Name),
	})
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
		c.Next(w, r, cause.New(codes.NotFound, "user/not_found", "User not found").WithCause(sql.ErrNoRows))
		return
	}

	c.Body(w, &HelloResponse{
		Message: "Hello, " + req.Name,
	}, http.StatusOK)
}

func TestHandler(t *testing.T) {
	c := new(Controller)
	c.BaseHandler = c.WithLogger(slog.Default())

	tests := map[string]struct {
		body     string
		wantCode int
		wantBody string
	}{
		"success":   {`{"name":"john"}`, http.StatusOK, `{"data":{"message":"Hello, john"}}`},
		"failed":    {`{}`, http.StatusBadRequest, `{"error":{"code":"VALIDATION_ERROR","message":"Validation failed","errors":{"name":"required"}}}`},
		"not found": {`{"name":"bob"}`, http.StatusNotFound, `{"error":{"code":"user/not_found","message":"User not found"}}`},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			http.HandlerFunc(c.Hello).ServeHTTP(w, r)

			if w.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, tc.wantCode)
			}
			if got := strings.TrimSpace(w.Body.String()); got != tc.wantBody {
				t.Fatalf("body = %s, want %s", got, tc.wantBody)
			}
		})
	}
}
