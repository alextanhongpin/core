// Package chain provides utilities for composing HTTP middleware into handler chains.
//
// This package simplifies the process of applying multiple middleware functions
// to HTTP handlers, supporting both the standard http.Handler interface and
// the more specific http.HandlerFunc type.
//
// The middleware is applied in reverse order (last middleware wraps first),
// which means the first middleware in the list will be the outermost wrapper
// and will execute first when handling requests.
//
// Example usage:
//
//	// Apply multiple middleware to an HTTP handler
//	handler := chain.Handler(myHandler,
//		loggingMiddleware,
//		authMiddleware,
//		corsMiddleware,
//	)
//
//	// For HandlerFunc-specific middleware
//	handlerFunc := chain.HandlerFunc(myHandlerFunc,
//		loggingMiddlewareFunc,
//		authMiddlewareFunc,
//	)
package chain

import "net/http"

// Middleware represents a function that wraps an http.Handler with additional functionality.
// Middleware functions can perform operations before and/or after calling the wrapped handler,
// enabling cross-cutting concerns like logging, authentication, CORS, etc.
type Middleware func(http.Handler) http.Handler

// Handler applies a sequence of middleware to an http.Handler, returning the composed handler.
//
// Middleware is applied in reverse order, meaning the last middleware in the slice
// will be applied first, making it the innermost wrapper. This ensures that the
// first middleware in the list becomes the outermost wrapper and executes first
// during request processing.
//
// Parameters:
//   - h: The base HTTP handler to wrap with middleware
//   - mws: A variadic list of middleware functions to apply
//
// Returns:
//   - An http.Handler with all middleware applied in the correct order
//
// Example:
//
//	handler := chain.Handler(baseHandler,
//		loggingMiddleware,    // Will execute first (outermost)
//		authMiddleware,      // Will execute second
//		corsMiddleware,      // Will execute third (innermost)
//	)
func Handler(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i > -1; i-- {
		h = mws[i](h)
	}

	return h
}

// MiddlewareFunc represents a function that wraps an http.HandlerFunc with additional functionality.
// This is similar to Middleware but specifically designed for http.HandlerFunc types,
// providing type safety and avoiding unnecessary type conversions.
type MiddlewareFunc func(http.HandlerFunc) http.HandlerFunc

// HandlerFunc applies a sequence of middleware functions to an http.HandlerFunc,
// returning the composed handler function.
//
// This function works similarly to Handler but is specifically designed for
// http.HandlerFunc types, avoiding the need for type conversions between
// http.Handler and http.HandlerFunc.
//
// Middleware is applied in reverse order, with the same semantics as Handler.
//
// Parameters:
//   - h: The base HTTP handler function to wrap with middleware
//   - mws: A variadic list of middleware functions to apply
//
// Returns:
//   - An http.HandlerFunc with all middleware applied in the correct order
//
// Example:
//
//	handlerFunc := chain.HandlerFunc(baseHandlerFunc,
//		loggingMiddlewareFunc,
//		authMiddlewareFunc,
//	)
func HandlerFunc(h http.HandlerFunc, mws ...MiddlewareFunc) http.HandlerFunc {
	for i := len(mws) - 1; i > -1; i-- {
		h = mws[i](h)
	}

	return h
}
