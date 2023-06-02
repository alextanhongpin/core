package middlewareutil

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

const AuthBearer = "Bearer"

var (
	ErrTokenInvalid = errors.New("invalid token")
)

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

// SignJWT signs a JWT with the given claims.
// Panics if no subject is provided.
// The expiration is added to the claims.
func SignJWT(secret []byte, claims jwt.MapClaims, valid time.Duration) (string, error) {
	if subject, ok := claims["sub"]; !ok || subject == "" {
		panic("subject required for jwt token")
	}

	claims["exp"] = time.Now().Add(valid).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	jwtToken, err := token.SignedString(secret)
	return jwtToken, err
}

func VerifyJWT(secret []byte, bearerToken string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(bearerToken, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %s", token.Header["alg"])
		}

		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}
