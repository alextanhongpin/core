// Package requestid provides HTTP middleware for request ID generation and tracking.
//
// Request IDs are essential for tracing HTTP requests through distributed systems,
// enabling correlation of logs, errors, and performance metrics across multiple
// services and components.
//
// This package supports both client-provided request IDs (from headers) and
// server-generated IDs when none are provided, ensuring every request has
// a unique identifier for tracking purposes.
//
// Example usage:
//
//	// Generate UUIDs for missing request IDs
//	handler := requestid.Handler(baseHandler, "X-Request-ID", func() string {
//		return uuid.New().String()
//	})
//
//	// Use nano IDs for shorter identifiers
//	handler := requestid.Handler(baseHandler, "X-Trace-ID", func() string {
//		return nanoid.New()
//	})
//
// The middleware automatically:
// - Checks for existing request ID in the specified header
// - Generates a new ID if none is provided
// - Adds the ID to the response header
// - Stores the ID in the request context for downstream handlers
package requestid

import "net/http"

// Handler creates middleware that manages request ID generation and tracking.
//
// This middleware performs the following operations:
// 1. Checks if a request ID already exists in the specified header
// 2. If no ID is present, generates one using the provided function
// 3. Sets the request ID in the request header (for downstream processing)
// 4. Sets the request ID in the response header (for client reference)
// 5. Stores the request ID in the request context for handler access
//
// The request ID can be retrieved from downstream handlers using the
// requestid.Context key.
//
// Parameters:
//   - h: The HTTP handler to wrap with request ID functionality
//   - key: The header name to use for the request ID (e.g., "X-Request-ID")
//   - fn: A function that generates new request IDs when none are provided
//
// Returns:
//   - An HTTP handler that ensures every request has a tracked request ID
//
// Example:
//
//	import "github.com/google/uuid"
//
//	handler := requestid.Handler(myHandler, "X-Request-ID", func() string {
//		return uuid.New().String()
//	})
//
//	// In downstream handlers, retrieve the request ID:
//	requestID, ok := requestid.Context.Value(r.Context())
//	if ok {
//		logger.Info("processing request", "request_id", requestID)
//	}
func Handler(h http.Handler, key string, fn func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		id := r.Header.Get(key)
		if id == "" {
			id = fn()
			r.Header.Set(key, id)

		}

		w.Header().Set(key, id)
		h.ServeHTTP(w, r.WithContext(Context.WithValue(r.Context(), id)))
	})
}
