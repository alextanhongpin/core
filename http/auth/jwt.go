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
	ErrTokenExpired  = errors.New("auth: token expired")
	ErrNoSecret      = errors.New("auth: no secret provided")
)

// Claims extends the standard JWT claims
type Claims = jwt.RegisteredClaims

// JWTConfig holds JWT configuration options
type JWTConfig struct {
	// Secret is the key used to sign and verify tokens
	Secret []byte
	// SigningMethod is the algorithm used to sign tokens (default: HS256)
	SigningMethod jwt.SigningMethod
	// Issuer is the 'iss' claim in the token
	Issuer string
	// ExpiryLeeway is the duration to add to token expiration time for validation
	ExpiryLeeway time.Duration
}

// JWT manages signing and verification of tokens
type JWT struct {
	Secret        []byte
	SigningMethod jwt.SigningMethod
	Issuer        string
	ExpiryLeeway  time.Duration
}

// NewJWT creates a new JWT manager with default settings
func NewJWT(secret []byte) *JWT {
	return NewJWTWithConfig(JWTConfig{
		Secret:        secret,
		SigningMethod: jwt.SigningMethodHS256,
	})
}

// NewJWTWithConfig creates a JWT manager with custom configuration
func NewJWTWithConfig(config JWTConfig) *JWT {
	if config.SigningMethod == nil {
		config.SigningMethod = jwt.SigningMethodHS256
	}

	return &JWT{
		Secret:        config.Secret,
		SigningMethod: config.SigningMethod,
		Issuer:        config.Issuer,
		ExpiryLeeway:  config.ExpiryLeeway,
	}
}

// Sign creates a signed JWT token with the provided claims
func (j *JWT) Sign(claims Claims, ttl time.Duration) (string, error) {
	if len(j.Secret) == 0 {
		return "", ErrNoSecret
	}

	if claims.Subject == "" {
		return "", fmt.Errorf("%w: subject is required", ErrClaimsInvalid)
	}

	now := time.Now()
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(ttl))

	// Set additional standard claims if not already set
	if claims.IssuedAt == nil {
		claims.IssuedAt = jwt.NewNumericDate(now)
	}

	if claims.NotBefore == nil {
		claims.NotBefore = jwt.NewNumericDate(now)
	}

	if claims.Issuer == "" && j.Issuer != "" {
		claims.Issuer = j.Issuer
	}

	token := jwt.NewWithClaims(j.SigningMethod, claims)
	return token.SignedString(j.Secret)
}

// SignWithCustomClaims creates a signed JWT with custom claims
func (j *JWT) SignWithCustomClaims(claims jwt.Claims, ttl time.Duration) (string, error) {
	if len(j.Secret) == 0 {
		return "", ErrNoSecret
	}

	token := jwt.NewWithClaims(j.SigningMethod, claims)
	return token.SignedString(j.Secret)
}

// Verify validates a token and returns the registered claims
func (j *JWT) Verify(bearerToken string) (*Claims, error) {
	if len(j.Secret) == 0 {
		return nil, ErrNoSecret
	}

	parserOptions := []jwt.ParserOption{}
	if j.ExpiryLeeway > 0 {
		parserOptions = append(parserOptions, jwt.WithLeeway(j.ExpiryLeeway))
	}

	token, err := jwt.ParseWithClaims(bearerToken, &Claims{}, func(token *jwt.Token) (any, error) {
		// Verify signing method matches expected method
		if token.Method.Alg() != j.SigningMethod.Alg() {
			return nil, fmt.Errorf("%w: unexpected signing method %s", ErrTokenInvalid, token.Header["alg"])
		}

		return j.Secret, nil
	}, parserOptions...)

	// Handle common JWT errors
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("%w: token expired", ErrTokenExpired)
		}
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
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

// VerifyWithCustomClaims validates a token with custom claims
func (j *JWT) VerifyWithCustomClaims(bearerToken string, claims jwt.Claims) error {
	if len(j.Secret) == 0 {
		return ErrNoSecret
	}

	parserOptions := []jwt.ParserOption{}
	if j.ExpiryLeeway > 0 {
		parserOptions = append(parserOptions, jwt.WithLeeway(j.ExpiryLeeway))
	}

	token, err := jwt.ParseWithClaims(bearerToken, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != j.SigningMethod.Alg() {
			return nil, fmt.Errorf("%w: unexpected signing method %s", ErrTokenInvalid, token.Header["alg"])
		}
		return j.Secret, nil
	}, parserOptions...)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return fmt.Errorf("%w: token expired", ErrTokenExpired)
		}
		return fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	if !token.Valid {
		return ErrClaimsInvalid
	}

	return nil
}
