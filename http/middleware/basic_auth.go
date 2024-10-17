package middleware

import "net/http"

func BasicAuthHandler(h http.Handler, credentials map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "Bad credentials", http.StatusForbidden)

			return
		}

		if credentials[username] != password {
			http.Error(w, "Bad credentials", http.StatusForbidden)

			return
		}

		h.ServeHTTP(w, r)
	})
}
