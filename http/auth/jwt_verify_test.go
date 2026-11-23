package auth_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestVerifyRejectsTokenWithoutExpiry(t *testing.T) {
	// Regression: tokens without "exp" were accepted forever.
	j := auth.NewJWT([]byte("secret"))
	token, err := j.SignWithCustomClaims(jwt.RegisteredClaims{Subject: "john"})
	assert.NoError(t, err)

	_, err = j.Verify(token)
	assert.ErrorIs(t, err, auth.ErrTokenInvalid)
}

func TestVerifyChecksIssuer(t *testing.T) {
	// Regression: a configured Issuer was only stamped on tokens, never checked.
	secret := []byte("secret")
	other := auth.NewJWTWithConfig(auth.JWTConfig{Secret: secret, Issuer: "other"})
	token, err := other.Sign(auth.Claims{Subject: "john"}, time.Hour)
	assert.NoError(t, err)

	j := auth.NewJWTWithConfig(auth.JWTConfig{Secret: secret, Issuer: "me"})
	_, err = j.Verify(token)
	assert.ErrorIs(t, err, auth.ErrTokenInvalid)

	token, err = j.Sign(auth.Claims{Subject: "john"}, time.Hour)
	assert.NoError(t, err)
	claims, err := j.Verify(token)
	assert.NoError(t, err)
	assert.Equal(t, "me", claims.Issuer)
}

func TestVerifyRejectsOtherAlgorithms(t *testing.T) {
	secret := []byte("secret")
	hs384 := auth.NewJWTWithConfig(auth.JWTConfig{Secret: secret, SigningMethod: jwt.SigningMethodHS384})
	token, err := hs384.Sign(auth.Claims{Subject: "john"}, time.Hour)
	assert.NoError(t, err)

	_, err = auth.NewJWT(secret).Verify(token)
	assert.ErrorIs(t, err, auth.ErrTokenInvalid)

	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject:   "john",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	assert.NoError(t, err)
	_, err = auth.NewJWT(secret).Verify(none)
	assert.ErrorIs(t, err, auth.ErrTokenInvalid)
}

func TestZeroValueSigningMethod(t *testing.T) {
	// Regression: a JWT literal without SigningMethod panicked.
	j := &auth.JWT{Secret: []byte("secret")}
	token, err := j.Sign(auth.Claims{Subject: "john"}, time.Hour)
	assert.NoError(t, err)
	_, err = j.Verify(token)
	assert.NoError(t, err)
}
