package auth

import (
	"net/http"
	"strings"
)

const Bearer = "Bearer"

func BearerAuth(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	bearer, token, ok := strings.Cut(auth, " ")
	return token, ok && bearer == Bearer && len(token) > 0
}
