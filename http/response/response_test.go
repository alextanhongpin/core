package response_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/http/response"
	"github.com/alextanhongpin/errors/cause"
	"github.com/alextanhongpin/errors/codes"
	"github.com/alextanhongpin/errors/validation"
	"github.com/google/go-cmp/cmp"
)

// assertJSON checks the status code, content type and JSON body of w.
func assertJSON(t *testing.T, w *httptest.ResponseRecorder, code int, body string) {
	t.Helper()

	if w.Code != code {
		t.Errorf("status = %d, want %d", w.Code, code)
	}
	if got, want := w.Header().Get("Content-Type"), response.ContentTypeJSON; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}

	var got, want any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON body %q: %v", w.Body.String(), err)
	}
	if err := json.Unmarshal([]byte(body), &want); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("body mismatch (-want +got):\n%s", diff)
	}
}

func TestErrorJSON(t *testing.T) {
	const badRequest = `{"error":{"code":"BAD_REQUEST","message":"The request provided is invalid"}}`
	known := cause.New(codes.BadRequest, "BAD_REQUEST", "The request provided is invalid")

	ve := make(validation.Errors)
	ve.If("email", !strings.Contains("xyz", "@"), "The email is invalid")

	tests := map[string]struct {
		err  error
		code int
		body string
	}{
		"known error":   {known, http.StatusBadRequest, badRequest},
		"error wrapped": {fmt.Errorf("wrapped error: %w", known), http.StatusBadRequest, badRequest},
		"error joined":  {errors.Join(known, errors.New("another error")), http.StatusBadRequest, badRequest},
		"validation errors": {ve.Error(), http.StatusBadRequest,
			`{"error":{"code":"VALIDATION_ERROR","message":"Validation failed","errors":{"email":["The email is invalid"]}}}`},
		"unknown error": {sql.ErrNoRows, http.StatusInternalServerError,
			`{"error":{"code":"INTERNAL_SERVER_ERROR","message":"An unexpected error occurred. Please try again later."}}`},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			response.ErrorJSON(w, tc.err)
			assertJSON(t, w, tc.code, tc.body)
		})
	}
}

func TestJSON(t *testing.T) {
	type user struct {
		ID   string `json:"id"`
		Name string
	}

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := []user{
			{ID: "user-1", Name: "Alice"},
			{ID: "user-2", Name: "Bob"},
		}
		response.JSON(w, response.Body[[]user]{Data: data}, http.StatusCreated)
		assertJSON(t, w, http.StatusCreated, `{"data":[{"id":"user-1","Name":"Alice"},{"id":"user-2","Name":"Bob"}]}`)
	})

	t.Run("no content", func(t *testing.T) {
		w := httptest.NewRecorder()
		response.NoContent(w)
		if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
			t.Fatalf("got %d %q, want 204 with no body", w.Code, w.Body.String())
		}
	})

	t.Run("custom body", func(t *testing.T) {
		w := httptest.NewRecorder()
		response.JSON(w, &response.Body[any]{
			PageInfo: &response.PageInfo{
				HasNextPage: true,
			},
		}, http.StatusOK)
		assertJSON(t, w, http.StatusOK, `{"pageInfo":{"hasPrevPage":false,"hasNextPage":true}}`)
	})

	t.Run("encode", func(t *testing.T) {
		w := httptest.NewRecorder()
		response.JSON(w, map[string]string{"hello": "world"}, http.StatusAccepted)
		assertJSON(t, w, http.StatusAccepted, `{"hello":"world"}`)
	})
}
