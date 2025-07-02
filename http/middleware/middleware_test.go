package middleware_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/middleware"
)

func TestLogger(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with logger middleware
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	loggedHandler := middleware.Logger(logger)(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.RemoteAddr = "127.0.0.1:12345"

	w := httptest.NewRecorder()
	loggedHandler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "test response" {
		t.Errorf("Expected 'test response', got '%s'", w.Body.String())
	}
}

func TestRecovery(t *testing.T) {
	// Create a panicking handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with recovery middleware
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	recoveryHandler := middleware.Recovery(logger)(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// This should not panic
	recoveryHandler.ServeHTTP(w, req)

	// Check that we got a 500 response
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	if w.Body.String() != "Internal Server Error" {
		t.Errorf("Expected 'Internal Server Error', got '%s'", w.Body.String())
	}
}

func TestCORS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	})

	config := middleware.CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	corsHandler := middleware.CORS(config)(handler)

	t.Run("allowed origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		corsHandler.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Error("Expected Access-Control-Allow-Origin header to be set")
		}

		if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Error("Expected Access-Control-Allow-Credentials header to be set")
		}
	})

	t.Run("preflight request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		corsHandler.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}

		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Expected Access-Control-Allow-Methods header to be set")
		}
	})

	t.Run("disallowed origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://evil.com")
		w := httptest.NewRecorder()

		corsHandler.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("Expected Access-Control-Allow-Origin header to not be set for disallowed origin")
		}
	})
}

func TestDefaultCORSConfig(t *testing.T) {
	config := middleware.DefaultCORSConfig()

	if len(config.AllowedOrigins) == 0 || config.AllowedOrigins[0] != "*" {
		t.Error("Expected default config to allow all origins")
	}

	if len(config.AllowedMethods) == 0 {
		t.Error("Expected default config to have allowed methods")
	}

	if config.MaxAge == 0 {
		t.Error("Expected default config to have max age set")
	}
}

func TestRateLimit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("success"))
	})

	// Allow 2 requests per second
	rateLimitHandler := middleware.RateLimit(2, time.Second, middleware.ByIP)(handler)

	t.Run("within limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"

		// First request should succeed
		w1 := httptest.NewRecorder()
		rateLimitHandler.ServeHTTP(w1, req)
		if w1.Code != http.StatusOK {
			t.Errorf("Expected status 200 for first request, got %d", w1.Code)
		}

		// Second request should succeed
		w2 := httptest.NewRecorder()
		rateLimitHandler.ServeHTTP(w2, req)
		if w2.Code != http.StatusOK {
			t.Errorf("Expected status 200 for second request, got %d", w2.Code)
		}
	})

	t.Run("exceeds limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "127.0.0.2:12345"

		// Use up the quota
		for i := 0; i < 2; i++ {
			w := httptest.NewRecorder()
			rateLimitHandler.ServeHTTP(w, req)
		}

		// This request should be rate limited
		w := httptest.NewRecorder()
		rateLimitHandler.ServeHTTP(w, req)
		if w.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status 429, got %d", w.Code)
		}

		if w.Header().Get("X-RateLimit-Limit") == "" {
			t.Error("Expected X-RateLimit-Limit header to be set")
		}
	})
}

func TestByIP(t *testing.T) {
	t.Run("X-Forwarded-For header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.1")
		req.RemoteAddr = "127.0.0.1:12345"

		key := middleware.ByIP(req)
		if key != "192.168.1.1" {
			t.Errorf("Expected '192.168.1.1', got '%s'", key)
		}
	})

	t.Run("X-Real-IP header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Real-IP", "192.168.1.2")
		req.RemoteAddr = "127.0.0.1:12345"

		key := middleware.ByIP(req)
		if key != "192.168.1.2" {
			t.Errorf("Expected '192.168.1.2', got '%s'", key)
		}
	})

	t.Run("RemoteAddr fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"

		key := middleware.ByIP(req)
		if key != "127.0.0.1:12345" {
			t.Errorf("Expected '127.0.0.1:12345', got '%s'", key)
		}
	})
}

func TestByAPIKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer abc123")

	key := middleware.ByAPIKey(req)
	if key != "Bearer abc123" {
		t.Errorf("Expected 'Bearer abc123', got '%s'", key)
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := middleware.NewRateLimiter(2, time.Second)
	defer limiter.Close()

	// First two requests should be allowed
	if !limiter.Allow("test-key") {
		t.Error("First request should be allowed")
	}
	if !limiter.Allow("test-key") {
		t.Error("Second request should be allowed")
	}

	// Third request should be denied
	if limiter.Allow("test-key") {
		t.Error("Third request should be denied")
	}

	// Different key should be allowed
	if !limiter.Allow("other-key") {
		t.Error("Request with different key should be allowed")
	}
}
