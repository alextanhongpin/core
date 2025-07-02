// Package middleware provides common HTTP middleware for production applications.
package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// Logger creates a structured logging middleware that logs HTTP requests and responses.
// It logs request method, path, remote address, user agent, response status,
// response size, and request duration.
//
// Example:
//
//	handler = middleware.Logger(handler, slog.Default())
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap the ResponseWriter to capture status and size
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			logger.InfoContext(r.Context(), "HTTP request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
				slog.Int("status", wrapped.statusCode),
				slog.Int("size", wrapped.size),
				slog.Duration("duration", duration),
			)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// Recovery creates a panic recovery middleware that catches panics and returns
// a 500 Internal Server Error response while logging the panic.
//
// Example:
//
//	handler = middleware.Recovery(handler, slog.Default())
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.ErrorContext(r.Context(), "HTTP handler panic",
						slog.Any("panic", err),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
					)

					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal Server Error"))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// CORS creates a CORS middleware with configurable options.
//
// Example:
//
//	corsConfig := middleware.CORSConfig{
//		AllowedOrigins: []string{"https://example.com"},
//		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
//		AllowedHeaders: []string{"Content-Type", "Authorization"},
//		AllowCredentials: true,
//		MaxAge: 3600,
//	}
//	handler = middleware.CORS(corsConfig)(handler)
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	// Set defaults
	if len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = []string{"*"}
	}
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	}
	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{"Accept", "Content-Type", "Content-Length", "Authorization"}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 86400 // 24 hours
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if origin != "" && isOriginAllowed(origin, config.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if contains(config.AllowedOrigins, "*") {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}

			w.Header().Set("Access-Control-Allow-Methods", joinStrings(config.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", joinStrings(config.AllowedHeaders, ", "))

			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if config.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORSConfig holds the configuration for CORS middleware.
type CORSConfig struct {
	AllowedOrigins   []string // Allowed origins, use ["*"] for all
	AllowedMethods   []string // Allowed HTTP methods
	AllowedHeaders   []string // Allowed headers
	AllowCredentials bool     // Whether to allow credentials
	MaxAge           int      // Preflight cache duration in seconds
}

// DefaultCORSConfig returns a permissive CORS configuration suitable for development.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           86400,
	}
}

// Helper functions
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
