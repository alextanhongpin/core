// Package middleware provides common HTTP middleware for production applications.
package middleware

import (
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
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
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			logger.InfoContext(r.Context(), "HTTP request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
				slog.Int("status", rw.statusCode),
				slog.Int("size", rw.size),
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

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// Recovery creates a panic recovery middleware that catches panics and returns
// a 500 Internal Server Error response while logging the panic. If the handler
// already started the response, the status cannot be changed and the error text
// is appended to whatever was written. http.ErrAbortHandler is re-panicked so
// net/http can abort the connection as intended.
//
// Example:
//
//	handler = middleware.Recovery(handler, slog.Default())
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					if err == http.ErrAbortHandler {
						// Let net/http abort the response silently.
						panic(err)
					}
					logger.LogAttrs(r.Context(), slog.LevelError, "HTTP handler panic",
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

// CORS creates a CORS middleware.
//
// CORS headers are only added for requests whose Origin is allowed. An allowed
// origin is echoed back with "Vary: Origin"; a "*" entry allows every origin
// and is answered with "Access-Control-Allow-Origin: *", which browsers refuse
// for credentialed requests, so list origins explicitly when AllowCredentials
// is true. Only real preflight requests (OPTIONS with Origin and
// Access-Control-Request-Method) are answered with 204; other OPTIONS
// requests reach next.
//
// Example:
//
//	handler = middleware.CORS(middleware.CORSConfig{
//		AllowedOrigins:   []string{"https://example.com"},
//		AllowCredentials: true,
//	})(handler)
func CORS(config CORSConfig) func(http.Handler) http.Handler {
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

	wildcard := slices.Contains(config.AllowedOrigins, "*")
	methods := strings.Join(config.AllowedMethods, ", ")
	headers := strings.Join(config.AllowedHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			h := w.Header()
			if !wildcard {
				h.Add("Vary", "Origin")
			}

			if origin == "" || !(wildcard || slices.Contains(config.AllowedOrigins, origin)) {
				next.ServeHTTP(w, r)
				return
			}

			if wildcard {
				h.Set("Access-Control-Allow-Origin", "*")
			} else {
				h.Set("Access-Control-Allow-Origin", origin)
				if config.AllowCredentials {
					h.Set("Access-Control-Allow-Credentials", "true")
				}
			}

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				h.Set("Access-Control-Allow-Methods", methods)
				h.Set("Access-Control-Allow-Headers", headers)
				if config.MaxAge > 0 {
					h.Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORSConfig holds the configuration for CORS middleware.
type CORSConfig struct {
	AllowedOrigins   []string // Allowed origins; "*" allows all (without credentials). Default ["*"].
	AllowedMethods   []string // Allowed methods for preflight responses.
	AllowedHeaders   []string // Allowed request headers for preflight responses.
	AllowCredentials bool     // Send Access-Control-Allow-Credentials for listed origins.
	MaxAge           int      // Preflight cache duration in seconds; negative omits the header. Default 86400.
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
