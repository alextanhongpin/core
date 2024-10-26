package httputil

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

const AuthBearer = "Bearer"

var ErrTokenInvalid = errors.New("httputil: invalid token")

// BearerAuth extracts the token from the authorization
// header.
// The naming is based on (*http.Request).BasicAuth
// method.
func BearerAuth(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	bearer, token, ok := strings.Cut(auth, " ")
	token = strings.TrimSpace(token)
	return token, ok && bearer == AuthBearer && len(token) > 0
}

type Claims = jwt.RegisteredClaims

// SignJWT signs a JWT with the given claims.
// Panics if no subject is provided.
// The expiration is added to the claims.
func SignJWT(secret []byte, claims Claims, valid time.Duration) (string, error) {
	if claims.Subject == "" {
		panic("httputil: subject is required")
	}

	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(valid))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	jwtToken, err := token.SignedString(secret)
	return jwtToken, err
}

func VerifyJWT(secret []byte, bearerToken string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(bearerToken, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %s", token.Header["alg"])
		}

		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenInvalid, err)
	}
	if !token.Valid {
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("%w: cannot cast claims", ErrTokenInvalid)
	}

	return claims, nil
}
