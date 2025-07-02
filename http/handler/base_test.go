package handler_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/http/handler"
	"github.com/alextanhongpin/errors/cause"
	"github.com/alextanhongpin/errors/codes"
)

// Test structs
type ValidatableRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (r ValidatableRequest) Validate() error {
	if r.Name == "" {
		return cause.Map{
			"name": errors.New("name is required"),
		}.Err()
	}
	if r.Email == "" {
		return cause.Map{
			"email": errors.New("email is required"),
		}.Err()
	}
	return nil
}

type SimpleRequest struct {
	Value string `json:"value"`
}

type TestResponse struct {
	Message string `json:"message"`
	ID      int    `json:"id"`
}

func TestBaseHandler_WithLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	base := handler.BaseHandler{}
	baseWithLogger := base.WithLogger(logger)

	// Test that the original handler is not modified
	if base == baseWithLogger {
		t.Error("WithLogger should return a new instance")
	}

	// Test logging functionality
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.SetPathValue("pattern", "/test")

	baseWithLogger.Next(w, r, errors.New("test error"))

	if buf.Len() == 0 {
		t.Error("Expected log output, got none")
	}
}

func TestBaseHandler_ReadJSON(t *testing.T) {
	base := handler.BaseHandler{}

	tests := []struct {
		name        string
		body        string
		contentType string
		target      any
		wantErr     bool
	}{
		{
			name:        "valid json",
			body:        `{"name":"john","email":"john@example.com"}`,
			contentType: "application/json",
			target:      &ValidatableRequest{},
			wantErr:     false,
		},
		{
			name:        "invalid json",
			body:        `{"name":}`,
			contentType: "application/json",
			target:      &ValidatableRequest{},
			wantErr:     true,
		},
		{
			name:        "empty body",
			body:        "",
			contentType: "application/json",
			target:      &ValidatableRequest{},
			wantErr:     false, // Empty body is allowed by default unless WithRequired() option is used
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", tt.contentType)

			err := base.ReadJSON(r, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBaseHandler_OK(t *testing.T) {
	base := handler.BaseHandler{}
	data := TestResponse{Message: "success", ID: 1}

	tests := []struct {
		name           string
		data           any
		expectedStatus int
	}{
		{
			name:           "default status code",
			data:           data,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "custom status code",
			data:           data,
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			base.JSON(w, tt.data, tt.expectedStatus)

			if w.Code != tt.expectedStatus {
				t.Errorf("OK() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if !strings.Contains(w.Body.String(), "success") {
				t.Error("OK() should contain response data")
			}
		})
	}
}

func TestBaseHandler_JSON(t *testing.T) {
	base := handler.BaseHandler{}
	data := TestResponse{Message: "test", ID: 42}

	w := httptest.NewRecorder()
	base.JSON(w, data, http.StatusAccepted)

	if w.Code != http.StatusAccepted {
		t.Errorf("JSON() status = %v, want %v", w.Code, http.StatusAccepted)
	}

	if !strings.Contains(w.Body.String(), "test") {
		t.Error("JSON() should contain response data")
	}
}

func TestBaseHandler_ErrorJSON(t *testing.T) {
	base := handler.BaseHandler{}

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "simple error",
			err:  errors.New("simple error"),
		},
		{
			name: "cause error",
			err:  cause.New(codes.NotFound, "user/not_found", "User not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			base.ErrorJSON(w, tt.err)

			if w.Code == http.StatusOK {
				t.Error("ErrorJSON() should not return 200 status")
			}
		})
	}
}

func TestBaseHandler_NoContent(t *testing.T) {
	base := handler.BaseHandler{}

	w := httptest.NewRecorder()
	base.NoContent(w)

	if w.Code != http.StatusNoContent {
		t.Errorf("NoContent() status = %v, want %v", w.Code, http.StatusNoContent)
	}

	if w.Body.Len() != 0 {
		t.Error("NoContent() should have empty body")
	}
}

func TestBaseHandler_Next(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	base := handler.BaseHandler{}.WithLogger(logger)

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "validation error",
			err: cause.Map{
				"field": errors.New("validation error"),
			}.Err(),
		},
		{
			name: "cause error",
			err:  cause.New(codes.NotFound, "resource/not_found", "Resource not found"),
		},
		{
			name: "generic error",
			err:  errors.New("generic error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)
			r.SetPathValue("pattern", "/test")

			base.Next(w, r, tt.err)

			if w.Code == http.StatusOK {
				t.Error("Next() should not return 200 status for errors")
			}

			if buf.Len() == 0 {
				t.Error("Next() should log the error")
			}
		})
	}
}

func TestBaseHandler_Next_NoLogger(t *testing.T) {
	base := handler.BaseHandler{} // No logger

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	base.Next(w, r, errors.New("test error"))

	if w.Code == http.StatusOK {
		t.Error("Next() should not return 200 status for errors")
	}
}

func TestBaseHandler_ReadAndValidateJSON(t *testing.T) {
	base := handler.BaseHandler{}

	tests := []struct {
		name        string
		body        string
		contentType string
		target      any
		wantErr     bool
	}{
		{
			name:        "valid json and validation",
			body:        `{"name":"john","email":"john@example.com"}`,
			contentType: "application/json",
			target:      &ValidatableRequest{},
			wantErr:     false,
		},
		{
			name:        "valid json but invalid validation",
			body:        `{"name":"","email":"john@example.com"}`,
			contentType: "application/json",
			target:      &ValidatableRequest{},
			wantErr:     true,
		},
		{
			name:        "invalid json",
			body:        `{"name":}`,
			contentType: "application/json",
			target:      &ValidatableRequest{},
			wantErr:     true,
		},
		{
			name:        "no validation interface",
			body:        `{"value":"test"}`,
			contentType: "application/json",
			target:      &SimpleRequest{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", tt.contentType)

			err := base.ReadJSON(r, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadAndValidateJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBaseHandler_GetRequestID(t *testing.T) {
	base := handler.BaseHandler{}

	tests := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{
			name: "X-Request-ID header",
			headers: map[string]string{
				"X-Request-ID": "test-request-id",
			},
			expected: "test-request-id",
		},
		{
			name: "X-Correlation-ID header",
			headers: map[string]string{
				"X-Correlation-ID": "test-correlation-id",
			},
			expected: "test-correlation-id",
		},
		{
			name: "both headers - X-Request-ID takes precedence",
			headers: map[string]string{
				"X-Request-ID":     "request-id",
				"X-Correlation-ID": "correlation-id",
			},
			expected: "request-id",
		},
		{
			name:     "no headers",
			headers:  map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			for key, value := range tt.headers {
				r.Header.Set(key, value)
			}

			result := base.GetRequestID(r)
			if result != tt.expected {
				t.Errorf("GetRequestID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBaseHandler_SetRequestID(t *testing.T) {
	base := handler.BaseHandler{}

	tests := []struct {
		name      string
		requestID string
		wantSet   bool
	}{
		{
			name:      "valid request ID",
			requestID: "test-request-id",
			wantSet:   true,
		},
		{
			name:      "empty request ID",
			requestID: "",
			wantSet:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			base.SetRequestID(w, tt.requestID)

			headerValue := w.Header().Get("X-Request-ID")
			if tt.wantSet {
				if headerValue != tt.requestID {
					t.Errorf("SetRequestID() header = %v, want %v", headerValue, tt.requestID)
				}
			} else {
				if headerValue != "" {
					t.Errorf("SetRequestID() header should be empty, got %v", headerValue)
				}
			}
		})
	}
}
