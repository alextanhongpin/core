package middleware

import (
	"net/http"

	"github.com/alextanhongpin/core/http/httputil"
)

var ClaimsContext httputil.Context[*httputil.Claims] = "claims_ctx"

func BearerAuthHandler(h http.Handler, secret []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token, ok := httputil.BearerAuth(r); ok {
			claims, err := httputil.VerifyJWT(secret, token)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)

				return
			}

			ctx := ClaimsContext.WithValue(r.Context(), claims)
			r = r.WithContext(ctx)
		}

		h.ServeHTTP(w, r)
	})
}

func RequireAuthHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ClaimsContext.Value(r.Context()); ok {
			h.ServeHTTP(w, r)

			return
		}

		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	})
}
