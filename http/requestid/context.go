// Package requestid provides HTTP middleware for request ID generation and tracking.
package requestid

import "github.com/alextanhongpin/core/http/contextkey"

// Context is the typed context key used to store and retrieve request IDs.
//
// This key is used by the requestid.Handler middleware to store the request ID
// in the request context, making it available to downstream handlers and middleware.
//
// Usage in handlers:
//
//	func myHandler(w http.ResponseWriter, r *http.Request) {
//		requestID, ok := requestid.Context.Value(r.Context())
//		if ok {
//			// Use request ID for logging, tracing, etc.
//			logger.Info("handling request", "request_id", requestID)
//		}
//	}
var Context contextkey.Key[string] = "request_id"
