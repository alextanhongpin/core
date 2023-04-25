package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alextanhongpin/core/http/contextkey"
	"github.com/alextanhongpin/core/http/response"
	jwt "github.com/golang-jwt/jwt/v5"
)

var AuthContext contextkey.ContextKey[jwt.Claims] = "auth_ctx"

const AuthBearer = "Bearer"

var (
	ErrInvalidAuthHeader = errors.New("invalid auth header")
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token expired")
)

type Middleware func(next http.Handler) http.Handler

func BearerAuth(verifyKey []byte) Middleware {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			token, ok := ParseAuthHeader(authHeader)
			if ok {
				claims, err := ValidateAuthHeader(verifyKey, token)
				if err != nil {
					response.JSONError(w, response.ErrUnauthorized)
					return
				}

				ctx := r.Context()
				ctx = AuthContext.WithValue(ctx, claims)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func ParseAuthHeader(authHeader string) (string, bool) {
	bearer, token, ok := strings.Cut(authHeader, " ")
	token = strings.TrimSpace(token)
	return token, ok && bearer == AuthBearer && len(token) > 0
}

func ValidateAuthHeader(verifyKey []byte, bearerToken string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(bearerToken, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method %s", ErrInvalidToken, token.Header["alg"])
		}

		return verifyKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}

		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}
