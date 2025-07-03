// Package auth provides authentication and authorization utilities for HTTP handlers.
// It supports multiple authentication methods including Basic Auth, Bearer tokens, and JWT.
package auth

import (
	"fmt"
	"log/slog"
	"net/http"
)

// BearerAuthConfig holds configuration for bearer token authentication middleware.
// This configuration allows for flexible authentication behavior including optional
// authentication and custom error handling.
type BearerAuthConfig struct {
	// Secret is the HMAC secret key used to sign and verify JWT tokens.
	// This should be a cryptographically secure random string of at least 32 bytes.
	Secret []byte

	// Optional determines whether authentication is required for the protected route.
	// When true, requests without tokens are allowed to proceed.
	// When false, all requests must have valid bearer tokens.
	Optional bool

	// OnError is called when authentication fails or tokens are invalid.
	// This allows for custom error responses and logging behavior.
	// If nil, DefaultBearerErrorHandler will be used.
	OnError func(w http.ResponseWriter, r *http.Request, err error)
}

// DefaultBearerErrorHandler provides a standard RFC 6750 compliant error response
// for bearer token authentication failures. It sets the appropriate WWW-Authenticate
// header and logs the error if a logger is available in the request context.
func DefaultBearerErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	// Set RFC 6750 compliant WWW-Authenticate header
	w.Header().Set("WWW-Authenticate", "Bearer error=\"invalid_token\"")
	w.WriteHeader(http.StatusUnauthorized)

	// Log the error if a logger is available in the context
	if logger, ok := LoggerContext.Value(r.Context()); ok {
		logger.Error("failed to verify jwt", slog.String("err", err.Error()))
	}
}

// BearerHandler creates a middleware that validates JWT bearer tokens using the provided secret.
// This is a convenience function that creates optional authentication - requests without
// tokens are allowed to proceed. For required authentication, use BearerHandlerWithConfig
// with Optional: false.
//
// Example:
//
//	handler := auth.BearerHandler(myHandler, []byte("my-secret-key"))
//	http.ListenAndServe(":8080", handler)
func BearerHandler(h http.Handler, secret []byte) http.Handler {
	return BearerHandlerWithConfig(h, BearerAuthConfig{
		Secret:   secret,
		Optional: true,
		OnError:  DefaultBearerErrorHandler,
	})
}

// BearerHandlerWithConfig creates a bearer token authentication middleware with
// customizable configuration. This allows fine-grained control over authentication
// behavior including optional vs required authentication and custom error handling.
//
// The middleware performs the following steps:
//  1. Extracts the bearer token from the Authorization header
//  2. If no token is present and authentication is optional, allows the request to proceed
//  3. If no token is present and authentication is required, calls the error handler
//  4. Verifies the JWT token signature and claims
//  5. Adds verified claims to the request context for use by downstream handlers
//
// Example:
//
//	config := auth.BearerAuthConfig{
//	    Secret:   []byte("my-secret-key"),
//	    Optional: false, // Require authentication
//	    OnError:  myCustomErrorHandler,
//	}
//	handler := auth.BearerHandlerWithConfig(myHandler, config)
func BearerHandlerWithConfig(h http.Handler, config BearerAuthConfig) http.Handler {
	// Create JWT handler with the provided secret
	jwt := NewJWT(config.Secret)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract bearer token from Authorization header
		token, ok := BearerAuth(r)

		// If no token and authentication is optional, proceed without authentication
		if !ok && config.Optional {
			h.ServeHTTP(w, r)
			return
		}

		// If no token but authentication is required, return error
		if !ok && !config.Optional {
			config.OnError(w, r, fmt.Errorf("no bearer token provided"))
			return
		}

		// Verify the JWT token and extract claims
		claims, err := jwt.Verify(token)
		if err != nil {
			config.OnError(w, r, err)
			return
		}

		// Add verified claims to request context for downstream handlers
		ctx := ClaimsContext.WithValue(r.Context(), claims)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireBearerHandler enforces that the request has valid JWT claims in its context.
// This middleware should be used after BearerHandler to ensure that only authenticated
// requests proceed. It checks for the presence of claims in the request context that
// were added by a previous authentication middleware.
//
// This is useful for protecting routes that must have authentication, while allowing
// the initial bearer handler to be optional for some routes.
//
// Example:
//
//	// Allow optional authentication
//	optionalAuth := auth.BearerHandler(myHandler, secret)
//
//	// Require authentication for specific routes
//	requiredAuth := auth.RequireBearerHandler(protectedHandler)
//
//	mux.Handle("/public", optionalAuth)
//	mux.Handle("/protected", chain.Handler(requiredAuth, optionalAuth))
func RequireBearerHandler(h http.Handler) http.Handler {
	return RequireBearerHandlerWithOptions(h, RequireBearerOptions{
		ErrorHandler: DefaultBearerErrorHandler,
	})
}

// RequireBearerOptions configures the behavior of the require bearer handler.
type RequireBearerOptions struct {
	// ErrorHandler is called when authentication is missing or invalid.
	// If nil, DefaultBearerErrorHandler will be used.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)
}

// RequireBearerHandlerWithOptions creates a customizable middleware that enforces
// the presence of valid JWT claims in the request context. This provides more
// granular control over error handling when authentication is missing.
//
// Example:
//
//	options := auth.RequireBearerOptions{
//	    ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
//	        // Custom JSON error response
//	        w.Header().Set("Content-Type", "application/json")
//	        w.WriteHeader(http.StatusUnauthorized)
//	        json.NewEncoder(w).Encode(map[string]string{
//	            "error": "authentication_required",
//	            "message": "This endpoint requires valid authentication",
//	        })
//	    },
//	}
//	handler := auth.RequireBearerHandlerWithOptions(myHandler, options)
func RequireBearerHandlerWithOptions(h http.Handler, options RequireBearerOptions) http.Handler {
	// Use default error handler if none provided
	if options.ErrorHandler == nil {
		options.ErrorHandler = DefaultBearerErrorHandler
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if claims are present in the request context
		_, ok := ClaimsContext.Value(r.Context())
		if !ok {
			options.ErrorHandler(w, r, fmt.Errorf("missing authentication"))
			return
		}

		// Claims are present, allow the request to proceed
		// Additional validation could be performed here, such as:
		// - Checking specific claims required for this route
		// - Validating user permissions
		// - Checking token expiration with custom logic

		h.ServeHTTP(w, r)
	})
}
