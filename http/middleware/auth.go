package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alextanhongpin/go-core-microservice/http/encoding"
	"github.com/alextanhongpin/go-core-microservice/http/types"
	jwt "github.com/golang-jwt/jwt/v5"
)

var AuthContext types.ContextKey[jwt.Claims] = "auth_ctx"

const AuthBearer = "Bearer"

var (
	ErrAuthorizationHeaderInvalid = errors.New("authorization header is invalid")
	ErrBearerInvalid              = errors.New("bearer is invalid")
	ErrTokenInvalid               = errors.New("token is invalid")
	ErrTokenMissing               = errors.New("token is missing")
	ErrTokenExpired               = errors.New("token expired")
)

type Middleware func(next http.Handler) http.Handler

func RequireAuth(verifyKey []byte) Middleware {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			claims, err := parseAndValidateAuthorizationHeader(verifyKey, authHeader)
			if err != nil {
				res := types.Result[any]{
					Error: &types.Error{
						Code:    types.ErrUnauthorized.Code,
						Message: types.ErrUnauthorized.Error(),
					},
				}

				encoding.EncodeJSON(w, res, http.StatusUnauthorized)
				return
			}

			ctx := r.Context()
			ctx = AuthContext.WithValue(ctx, claims)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func ParseAuthorizationHeader(authHeader string) (string, error) {
	bearer, token, ok := strings.Cut(authHeader, " ")
	if !ok {
		return "", ErrAuthorizationHeaderInvalid
	}

	if bearer != AuthBearer {
		return "", ErrBearerInvalid
	}

	if token == "" {
		return "", ErrTokenMissing
	}

	return token, nil
}

func ValidateAuthorizationHeader(verifyKey []byte, bearerToken string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(bearerToken, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method %s", ErrTokenInvalid, token.Header["alg"])
		}

		return verifyKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %w", ErrTokenInvalid, err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

func parseAndValidateAuthorizationHeader(verifyKey []byte, token string) (jwt.Claims, error) {
	bearerToken, err := ParseAuthorizationHeader(token)
	if err != nil {
		return nil, err
	}

	claims, err := ValidateAuthorizationHeader(verifyKey, bearerToken)
	if err != nil {
		return nil, err
	}

	return claims, nil
}
