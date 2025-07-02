package auth

import (
	"crypto/subtle"
	"fmt"
	"net/http"
)

// BasicAuthConfig holds configuration for basic authentication
type BasicAuthConfig struct {
	// Credentials maps usernames to passwords
	Credentials map[string]string
	// Realm is the authentication realm displayed to the user
	Realm string
}

// BasicHandler creates a middleware that enforces HTTP Basic Authentication
func BasicHandler(h http.Handler, credentials map[string]string) http.Handler {
	return BasicHandlerWithConfig(h, BasicAuthConfig{
		Credentials: credentials,
		Realm:       "User Visible Realm",
	})
}

// BasicHandlerWithConfig creates a middleware with customizable configuration
func BasicHandlerWithConfig(h http.Handler, config BasicAuthConfig) http.Handler {
	realm := config.Realm
	if realm == "" {
		realm = "Restricted"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		expectedPassword, exists := config.Credentials[username]
		if !exists || !constantTimeCompare(expectedPassword, password) {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Store the authenticated username in the context
		ctx := UsernameContext.WithValue(r.Context(), username)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func constantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
