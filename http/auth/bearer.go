package auth

import (
	"fmt"
	"log/slog"
	"net/http"
)

// BearerAuthConfig holds configuration for bearer authentication
type BearerAuthConfig struct {
	// Secret is the key used to sign and verify JWTs
	Secret []byte
	// Optional determines whether authentication is required
	Optional bool
	// OnError is called when authentication fails (e.g., for custom error responses)
	OnError func(w http.ResponseWriter, r *http.Request, err error)
}

// DefaultBearerErrorHandler provides a standard error response for authentication failures
func DefaultBearerErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("WWW-Authenticate", "Bearer error=\"invalid_token\"")
	w.WriteHeader(http.StatusUnauthorized)

	if logger, ok := LoggerContext.Value(r.Context()); ok {
		logger.Error("failed to verify jwt", slog.String("err", err.Error()))
	}
}

// BearerHandler creates a middleware that validates JWT bearer tokens
func BearerHandler(h http.Handler, secret []byte) http.Handler {
	return BearerHandlerWithConfig(h, BearerAuthConfig{
		Secret:   secret,
		Optional: true,
		OnError:  DefaultBearerErrorHandler,
	})
}

// BearerHandlerWithConfig creates a middleware with customizable configuration
func BearerHandlerWithConfig(h http.Handler, config BearerAuthConfig) http.Handler {
	jwt := NewJWT(config.Secret)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := BearerAuth(r)

		// If no token and authentication is optional, proceed
		if !ok && config.Optional {
			h.ServeHTTP(w, r)
			return
		}

		// If no token but authentication is required
		if !ok && !config.Optional {
			config.OnError(w, r, fmt.Errorf("no bearer token provided"))
			return
		}

		// Verify the token
		claims, err := jwt.Verify(token)
		if err != nil {
			config.OnError(w, r, err)
			return
		}

		// Add claims to context and continue
		ctx := ClaimsContext.WithValue(r.Context(), claims)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireBearerHandler enforces that the request has valid JWT claims
func RequireBearerHandler(h http.Handler) http.Handler {
	return RequireBearerHandlerWithOptions(h, RequireBearerOptions{
		ErrorHandler: DefaultBearerErrorHandler,
	})
}

// RequireBearerOptions configures the requirement bearer handler
type RequireBearerOptions struct {
	// ErrorHandler is called when authentication is missing
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)
}

// RequireBearerHandlerWithOptions creates a customizable middleware
func RequireBearerHandlerWithOptions(h http.Handler, options RequireBearerOptions) http.Handler {
	if options.ErrorHandler == nil {
		options.ErrorHandler = DefaultBearerErrorHandler
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := ClaimsContext.Value(r.Context())
		if !ok {
			options.ErrorHandler(w, r, fmt.Errorf("missing authentication"))
			return
		}

		// Additional validation could be performed here
		// For example, checking specific claims required for this route

		h.ServeHTTP(w, r)
	})
}
