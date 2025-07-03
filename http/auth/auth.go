// Package auth provides authentication and authorization utilities for HTTP handlers.
package auth

import (
	"net/http"
	"strings"
)

// Bearer is the expected authorization scheme for bearer token authentication.
const Bearer = "Bearer"

// BearerAuth extracts and validates a bearer token from the Authorization header.
//
// This function parses the Authorization header looking for a bearer token in the
// format "Bearer <token>". It returns the token and a boolean indicating whether
// a valid bearer token was found.
//
// Parameters:
//   - r: The HTTP request to extract the bearer token from
//
// Returns:
//   - The bearer token string (empty if not found or invalid)
//   - A boolean indicating whether a valid bearer token was found
//
// Example:
//
//	// Authorization: Bearer abc123xyz
//	token, ok := auth.BearerAuth(r)
//	if ok {
//		// token = "abc123xyz"
//		// Validate token...
//	}
//
// The function validates that:
// - The Authorization header is present
// - The header starts with "Bearer "
// - There is a non-empty token after "Bearer "
func BearerAuth(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	bearer, token, ok := strings.Cut(auth, " ")
	return token, ok && bearer == Bearer && len(token) > 0
}
