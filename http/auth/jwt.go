package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrClaimsInvalid = errors.New("auth: invalid claims")
	ErrTokenInvalid  = errors.New("auth: invalid token")
)

type Claims = jwt.RegisteredClaims

type JWT struct {
	Secret []byte
}

func NewJWT(secret []byte) *JWT {
	return &JWT{
		Secret: secret,
	}
}

func (j *JWT) Sign(claims Claims, ttl time.Duration) (string, error) {
	if claims.Subject == "" {
		return "", fmt.Errorf("%w: subject is required", ErrClaimsInvalid)
	}

	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(ttl))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.Secret)
}

func (j *JWT) Verify(bearerToken string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(bearerToken, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method %s", ErrTokenInvalid, token.Header["alg"])
		}

		return j.Secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenInvalid, err)
	}
	if !token.Valid {
		return nil, ErrClaimsInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("%w: failed to cast claims", ErrTokenInvalid)
	}

	return claims, nil
}
