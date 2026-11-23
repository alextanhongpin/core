package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/response"
)

func TestJSONEncodingFailure(t *testing.T) {
	// Regression: the status was written before encoding, so a failure
	// produced 200 with a partial or appended body (and a superfluous
	// WriteHeader call).
	w := httptest.NewRecorder()
	data := map[string]any{"bad_number": json.Number("1.5x")}
	response.JSON(w, response.Body[map[string]any]{Data: data}, http.StatusOK)

	assertJSON(t, w, http.StatusInternalServerError,
		`{"error":{"code":"INTERNAL_SERVER_ERROR","message":"An unexpected error occurred. Please try again later."}}`)
}

type anyMapError map[string]any

func (e anyMapError) Error() string       { return "validation failed" }
func (e anyMapError) Map() map[string]any { return e }
func (e anyMapError) Unwrap() []error     { return nil }

var _ interface{ Map() map[string]any } = anyMapError{}

func TestErrorJSONAnyMapValidationError(t *testing.T) {
	// Regression: validation errors with Map() map[string]any (as returned by
	// github.com/alextanhongpin/errors/validator) were reported as 500.
	w := httptest.NewRecorder()
	response.ErrorJSON(w, errors.Join(anyMapError{"name": "required"}))

	assertJSON(t, w, http.StatusBadRequest,
		`{"error":{"code":"VALIDATION_ERROR","message":"Validation failed","errors":{"name":"required"}}}`)
}

func TestResponseWriterRecorderCapturesWholeBody(t *testing.T) {
	// Regression: only the first Write was captured.
	w := httptest.NewRecorder()
	rec := response.NewResponseWriterRecorder(w)
	rec.SetWriteBody(true)

	rec.Write([]byte("hello, "))
	rec.Write([]byte("world"))

	if got := string(rec.Body()); got != "hello, world" {
		t.Fatalf("Body() = %q, want %q", got, "hello, world")
	}
	if rec.StatusCode() != http.StatusOK || w.Body.String() != "hello, world" {
		t.Fatalf("got %d %q", rec.StatusCode(), w.Body.String())
	}
}
