package auth

import (
	"crypto/subtle"
	"net/http"
)

func BasicHandler(h http.Handler, credentials map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || !constantTimeCompare(credentials[username], password) {

			w.Header().Set("WWW-Authenticate", `Basic realm="User Visible Realm"`)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		h.ServeHTTP(w, r)
	})
}

func constantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
