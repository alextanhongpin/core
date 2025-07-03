package auth

import (
	"crypto/subtle"
	"fmt"
	"net/http"
)

// BasicAuthConfig holds configuration for HTTP Basic Authentication middleware.
// Basic authentication uses username/password credentials encoded in the
// Authorization header as specified in RFC 7617.
type BasicAuthConfig struct {
	// Credentials maps usernames to their corresponding passwords.
	// Passwords should be stored securely (hashed) in production environments.
	Credentials map[string]string

	// Realm is the authentication realm displayed to users in browser dialogs.
	// This helps users understand what they're authenticating for.
	// If empty, defaults to "Restricted".
	Realm string
}

// BasicHandler creates an HTTP Basic Authentication middleware using the provided
// username/password credentials. This is a convenience function that uses default
// configuration with realm "User Visible Realm".
//
// The middleware performs constant-time password comparison to prevent timing attacks.
// Upon successful authentication, the username is stored in the request context
// and can be retrieved using auth.UsernameFromContext().
//
// Example:
//
//	credentials := map[string]string{
//	    "admin": "secret123",
//	    "user":  "password456",
//	}
//	handler := auth.BasicHandler(myHandler, credentials)
//	http.ListenAndServe(":8080", handler)
func BasicHandler(h http.Handler, credentials map[string]string) http.Handler {
	return BasicHandlerWithConfig(h, BasicAuthConfig{
		Credentials: credentials,
		Realm:       "User Visible Realm",
	})
}

// BasicHandlerWithConfig creates an HTTP Basic Authentication middleware with
// customizable configuration. This allows control over the authentication realm
// and credential storage.
//
// The middleware follows RFC 7617 and performs these steps:
//  1. Extracts credentials from the Authorization header
//  2. Validates credentials against the configured username/password map
//  3. Uses constant-time comparison to prevent timing attacks
//  4. Sets WWW-Authenticate header on authentication failures
//  5. Stores authenticated username in request context on success
//
// Example:
//
//	config := auth.BasicAuthConfig{
//	    Credentials: map[string]string{"admin": "secret"},
//	    Realm:       "Admin Panel",
//	}
//	handler := auth.BasicHandlerWithConfig(myHandler, config)
func BasicHandlerWithConfig(h http.Handler, config BasicAuthConfig) http.Handler {
	// Use default realm if none provided
	realm := config.Realm
	if realm == "" {
		realm = "Restricted"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract username and password from Authorization header
		username, password, ok := r.BasicAuth()
		if !ok {
			// No credentials provided, challenge the client
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Check if username exists and password matches
		expectedPassword, exists := config.Credentials[username]
		if !exists || !constantTimeCompare(expectedPassword, password) {
			// Invalid credentials, challenge the client again
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Authentication successful - store username in context for downstream handlers
		ctx := UsernameContext.WithValue(r.Context(), username)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// constantTimeCompare performs constant-time string comparison to prevent timing attacks.
// This is critical for password comparison as it ensures that the time taken to compare
// passwords doesn't leak information about correct vs incorrect passwords.
func constantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
