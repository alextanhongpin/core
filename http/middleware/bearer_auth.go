package middleware

import (
	"net/http"

	"github.com/alextanhongpin/core/http/httputil"
)

var ClaimsContext httputil.Context[*httputil.Claims] = "claims_ctx"

func BearerAuthHandler(h http.Handler, secret []byte, strict bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := httputil.BearerAuth(r)
		if !ok {
			if strict {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)

				return
			}

			h.ServeHTTP(w, r)

			return
		}

		claims, err := httputil.VerifyJWT(secret, token)
		if err != nil {
                        http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			
			return
		}

		h.ServeHTTP(w, r.WithContext(ClaimsContext.WithValue(r.Context(), claims)))
	})
}
