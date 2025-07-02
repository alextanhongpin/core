package response_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/response"
	"github.com/stretchr/testify/assert"
)

func TestContentTypes(t *testing.T) {
	t.Run("text response", func(t *testing.T) {
		w := httptest.NewRecorder()

		response.Text(w, "Hello, World!", http.StatusOK)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
		assert.Equal(t, "Hello, World!", w.Body.String())
	})

	t.Run("HTML response", func(t *testing.T) {
		w := httptest.NewRecorder()
		html := "<html><body><h1>Hello</h1></body></html>"

		response.HTML(w, html, http.StatusOK)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
		assert.Equal(t, html, w.Body.String())
	})
}

func TestSecurityHeaders(t *testing.T) {
	t.Run("set security headers", func(t *testing.T) {
		w := httptest.NewRecorder()

		response.SetSecurityHeaders(w)

		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
		assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	})

	t.Run("set cache headers with max age", func(t *testing.T) {
		w := httptest.NewRecorder()

		response.SetCacheHeaders(w, 3600) // 1 hour

		assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))
		assert.NotEmpty(t, w.Header().Get("Expires"))
	})

	t.Run("set no-cache headers", func(t *testing.T) {
		w := httptest.NewRecorder()

		response.SetCacheHeaders(w, 0)

		assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
		assert.Equal(t, "no-cache", w.Header().Get("Pragma"))
		assert.Equal(t, "0", w.Header().Get("Expires"))
	})
}

func TestCORS(t *testing.T) {
	t.Run("set CORS headers", func(t *testing.T) {
		w := httptest.NewRecorder()

		origins := []string{"https://example.com"}
		methods := []string{"GET", "POST", "PUT", "DELETE"}
		headers := []string{"Content-Type", "Authorization"}

		response.CORS(w, origins, methods, headers)

		assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, PUT, DELETE", w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
	})
}
