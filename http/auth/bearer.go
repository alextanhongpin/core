package auth

import (
	"log/slog"
	"net/http"
)

func BearerHandler(h http.Handler, secret []byte) http.Handler {
	jwt := NewJWT(secret)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token, ok := BearerAuth(r); ok {
			claims, err := jwt.Verify(token)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)

				if logger, ok := LoggerContext.Value(r.Context()); ok {
					logger.Error("failed to verify jwt", slog.String("err", err.Error()))
				}

				return
			}

			ctx := ClaimsContext.WithValue(r.Context(), claims)
			r = r.WithContext(ctx)
		}

		h.ServeHTTP(w, r)
	})
}

func RequireBearerHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ClaimsContext.Value(r.Context()); ok {
			h.ServeHTTP(w, r)

			return
		}

		w.WriteHeader(http.StatusUnauthorized)
	})
}
