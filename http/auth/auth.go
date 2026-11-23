// Package auth provides authentication and authorization utilities for HTTP handlers.
package auth

import (
	"net/http"
	"strings"
)

// Bearer is the expected authorization scheme for bearer token authentication.
const Bearer = "Bearer"

// BearerAuth returns the token from an "Authorization: Bearer <token>"
// header. The scheme is matched case-insensitively (RFC 9110) and surrounding
// whitespace is ignored. It returns false when the header is missing, uses
// another scheme, or carries an empty token.
//
//	// Authorization: Bearer abc123xyz
//	token, ok := auth.BearerAuth(r) // "abc123xyz", true
func BearerAuth(r *http.Request) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(r.Header.Get("Authorization")), " ")
	if !ok || !strings.EqualFold(scheme, Bearer) {
		return "", false
	}

	token = strings.TrimSpace(token)
	return token, token != ""
}
