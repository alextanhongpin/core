package response

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/alextanhongpin/errors/cause"
	"github.com/alextanhongpin/errors/codes"
)

// ContentType constants
const (
	ContentTypeJSON = "application/json; charset=utf-8"
	ContentTypeText = "text/plain; charset=utf-8"
	ContentTypeHTML = "text/html; charset=utf-8"
	ContentTypeXML  = "application/xml; charset=utf-8"
)

// NoContent sends a 204 No Content response
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// RawJSON sends raw JSON data with proper validation
func RawJSON(w http.ResponseWriter, data []byte, code int) {
	if !json.Valid(data) {
		ErrorJSON(w, errors.New("invalid JSON data"))
		return
	}

	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(cmp.Or(code, http.StatusOK))

	if _, err := w.Write(data); err != nil {
		// Log the error but don't send another response as headers are already written
		slog.Default().Error("failed to write response", "error", err)
	}
}

// JSON sends a JSON response
func JSON(w http.ResponseWriter, data any, code int) {
	b, err := json.Marshal(data)
	if err != nil {
		ErrorJSON(w, err)

		return
	}

	RawJSON(w, b, code)
}

// Text sends a plain text response
func Text(w http.ResponseWriter, text string, code int) {
	w.Header().Set("Content-Type", ContentTypeText)
	w.WriteHeader(code)

	if _, err := w.Write([]byte(text)); err != nil {
		slog.Default().Error("failed to write text response", "error", err)
	}
}

// HTML sends an HTML response
func HTML(w http.ResponseWriter, html string, code int) {
	w.Header().Set("Content-Type", ContentTypeHTML)
	w.WriteHeader(code)

	if _, err := w.Write([]byte(html)); err != nil {
		slog.Default().Error("failed to write HTML response", "error", err)
	}
}

// ErrorJSON sends a JSON error response with proper error handling
func ErrorJSON(w http.ResponseWriter, err error) {
	var (
		c  *cause.Error
		e  *Error
		ve interface {
			Map() map[string]any
		}
		code = http.StatusInternalServerError
	)

	switch {
	case errors.As(err, &ve):
		e = &Error{
			Code:    "VALIDATION_ERROR",
			Message: "Validation failed",
			Errors:  ve.Map(),
		}
		code = http.StatusBadRequest

	case errors.As(err, &c):
		// Handle cause errors
		code = codes.HTTP(c.Code)
		e = &Error{
			Code:    c.Name,
			Message: c.Message,
		}

	default:
		e = &Error{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "An unexpected error occurred. Please try again later.",
		}
	}

	JSON(w, Body{Error: e}, code)
}

// SetCacheHeaders sets appropriate cache headers
func SetCacheHeaders(w http.ResponseWriter, maxAge int) {
	if maxAge > 0 {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
		w.Header().Set("Expires", time.Now().Add(time.Duration(maxAge)*time.Second).Format(http.TimeFormat))
	} else {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}
}

// SetSecurityHeaders sets common security headers
func SetSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

// CORSOptions defines configuration options for CORS middleware
type CORSOptions struct {
	// AllowedOrigins is a list of allowed origins. Use ["*"] to allow any origin
	AllowedOrigins []string
	// AllowedMethods is a list of allowed HTTP methods
	AllowedMethods []string
	// AllowedHeaders is a list of allowed HTTP headers
	AllowedHeaders []string
	// ExposedHeaders is a list of headers that are exposed to the client
	ExposedHeaders []string
	// AllowCredentials indicates whether the request can include user credentials
	AllowCredentials bool
	// MaxAge indicates how long the results of a preflight request can be cached
	MaxAge int
}

// CORS sets CORS headers for cross-origin requests
func CORS(w http.ResponseWriter, allowedOrigins []string, allowedMethods []string, allowedHeaders []string) {
	options := CORSOptions{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: allowedMethods,
		AllowedHeaders: allowedHeaders,
	}
	ApplyCORS(w, options)
}

// ApplyCORS applies CORS headers based on the provided options
func ApplyCORS(w http.ResponseWriter, options CORSOptions) {
	if len(options.AllowedOrigins) > 0 {
		if len(options.AllowedOrigins) == 1 && options.AllowedOrigins[0] == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if len(options.AllowedOrigins) > 0 {
			// In production, you'd typically check the Origin header against the allowed origins
			// and set the specific matching origin, but for simplicity we use the first one here
			w.Header().Set("Access-Control-Allow-Origin", options.AllowedOrigins[0])
		}
	}

	if len(options.AllowedMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(options.AllowedMethods, ", "))
	}

	if len(options.AllowedHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(options.AllowedHeaders, ", "))
	}

	if len(options.ExposedHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(options.ExposedHeaders, ", "))
	}

	if options.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if options.MaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", options.MaxAge))
	}
}

// CORSHandler creates a middleware handler that applies CORS headers
func CORSHandler(h http.Handler, options CORSOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Apply CORS headers to all responses
		ApplyCORS(w, options)

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Continue with the request
		h.ServeHTTP(w, r)
	})
}
