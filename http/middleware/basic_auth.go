package middleware

import "net/http"

func BasicAuthHandler(h http.Handler, credentials map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || credentials[username] != password {

			w.Header().Set("WWW-Authenticate", `Basic realm="User Visible Realm"`)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		h.ServeHTTP(w, r)
	})
}
