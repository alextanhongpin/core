package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWT-related errors that can occur during token operations.
var (
	// ErrClaimsInvalid indicates that the JWT claims are malformed or invalid.
	ErrClaimsInvalid = errors.New("auth: invalid claims")

	// ErrTokenInvalid indicates that the JWT token is malformed, has invalid signature, or other issues.
	ErrTokenInvalid = errors.New("auth: invalid token")

	// ErrTokenExpired indicates that the JWT token has passed its expiration time.
	ErrTokenExpired = errors.New("auth: token expired")

	// ErrNoSecret indicates that no signing secret was provided for JWT operations.
	ErrNoSecret = errors.New("auth: no secret provided")
)

// Claims extends the standard JWT RegisteredClaims with commonly used fields.
// This type alias allows for easy extension while maintaining compatibility
// with the jwt-go library's standard claims.
type Claims = jwt.RegisteredClaims

// JWTConfig holds configuration options for JWT token management.
// This allows customization of signing methods, issuers, and validation behavior.
type JWTConfig struct {
	// Secret is the HMAC secret key used to sign and verify JWT tokens.
	// For HMAC algorithms (HS256, HS384, HS512), this should be a cryptographically
	// secure random string of appropriate length (32+ bytes recommended).
	Secret []byte

	// SigningMethod specifies the algorithm used to sign tokens.
	// Common values: jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512
	// Default: HS256 if not specified.
	SigningMethod jwt.SigningMethod

	// Issuer is the 'iss' (issuer) claim added to generated tokens.
	// This identifies the service that issued the token and can be used for validation.
	Issuer string

	// ExpiryLeeway is additional time added to token expiration during validation.
	// This accounts for clock skew between systems. Common values: 30s-5m.
	ExpiryLeeway time.Duration
}

// JWT manages the signing and verification of JWT tokens with configurable options.
// It provides a high-level interface for common JWT operations while allowing
// fine-grained control over token behavior.
type JWT struct {
	Secret        []byte
	SigningMethod jwt.SigningMethod
	Issuer        string
	ExpiryLeeway  time.Duration
}

// NewJWT creates a new JWT manager with default settings (HS256 signing method).
// This is a convenience function for simple use cases where default configuration
// is sufficient.
//
// Example:
//
//	jwt := auth.NewJWT([]byte("my-secret-key"))
//	token, err := jwt.Sign(auth.Claims{Subject: "user123"}, time.Hour)
func NewJWT(secret []byte) *JWT {
	return NewJWTWithConfig(JWTConfig{
		Secret:        secret,
		SigningMethod: jwt.SigningMethodHS256,
	})
}

// NewJWTWithConfig creates a JWT manager with custom configuration options.
// This allows full control over signing methods, issuers, and validation behavior.
//
// Example:
//
//	config := auth.JWTConfig{
//	    Secret:        []byte("my-secret-key"),
//	    SigningMethod: jwt.SigningMethodHS384,
//	    Issuer:        "my-service",
//	    ExpiryLeeway:  30 * time.Second,
//	}
//	jwt := auth.NewJWTWithConfig(config)
func NewJWTWithConfig(config JWTConfig) *JWT {
	// Use default signing method if none specified
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

// Sign creates a signed JWT token with the provided claims and time-to-live duration.
// It automatically sets standard claims like issued-at, not-before, and expiration times.
// The subject claim is required and an error is returned if it's empty.
//
// Example:
//
//	claims := auth.Claims{
//	    Subject: "user123",
//	    Issuer:  "my-service", // Optional, will use JWT.Issuer if not set
//	}
//	token, err := jwt.Sign(claims, 24*time.Hour)
func (j *JWT) Sign(claims Claims, ttl time.Duration) (string, error) {
	if len(j.Secret) == 0 {
		return "", ErrNoSecret
	}

	// Subject is required for all tokens
	if claims.Subject == "" {
		return "", fmt.Errorf("%w: subject is required", ErrClaimsInvalid)
	}

	now := time.Now()

	// Set expiration time based on TTL
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(ttl))

	// Set additional standard claims if not already provided
	if claims.IssuedAt == nil {
		claims.IssuedAt = jwt.NewNumericDate(now)
	}

	if claims.NotBefore == nil {
		claims.NotBefore = jwt.NewNumericDate(now)
	}

	// Use configured issuer if not set in claims
	if claims.Issuer == "" && j.Issuer != "" {
		claims.Issuer = j.Issuer
	}

	// Create and sign the token
	token := jwt.NewWithClaims(j.SigningMethod, claims)
	return token.SignedString(j.Secret)
}

// SignWithCustomClaims creates a signed JWT token with custom claims structure.
// This allows for non-standard claims beyond the registered claims set.
// Note: TTL parameter is not used here as custom claims should handle expiration.
//
// Example:
//
//	type CustomClaims struct {
//	    jwt.RegisteredClaims
//	    Role        string   `json:"role"`
//	    Permissions []string `json:"permissions"`
//	}
//
//	claims := CustomClaims{
//	    RegisteredClaims: jwt.RegisteredClaims{
//	        Subject:   "user123",
//	        ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
//	    },
//	    Role:        "admin",
//	    Permissions: []string{"read", "write"},
//	}
//	token, err := jwt.SignWithCustomClaims(claims, time.Hour)
func (j *JWT) SignWithCustomClaims(claims jwt.Claims, ttl time.Duration) (string, error) {
	if len(j.Secret) == 0 {
		return "", ErrNoSecret
	}

	// Create and sign the token with custom claims
	token := jwt.NewWithClaims(j.SigningMethod, claims)
	return token.SignedString(j.Secret)
}

// Verify validates a JWT token string and returns the parsed registered claims.
// It performs comprehensive validation including:
//   - Signature verification using the configured secret
//   - Signing method validation to prevent algorithm confusion attacks
//   - Expiration time validation with optional leeway for clock skew
//   - Standard claims validation (nbf, iat, etc.)
//
// Example:
//
//	claims, err := jwt.Verify(tokenString)
//	if err != nil {
//	    // Handle invalid token
//	    return
//	}
//	userID := claims.Subject
func (j *JWT) Verify(bearerToken string) (*Claims, error) {
	if len(j.Secret) == 0 {
		return nil, ErrNoSecret
	}

	// Configure parser options for expiry leeway if specified
	parserOptions := []jwt.ParserOption{}
	if j.ExpiryLeeway > 0 {
		parserOptions = append(parserOptions, jwt.WithLeeway(j.ExpiryLeeway))
	}

	// Parse and validate the token
	token, err := jwt.ParseWithClaims(bearerToken, &Claims{}, func(token *jwt.Token) (any, error) {
		// Verify signing method matches expected method to prevent algorithm confusion attacks
		if token.Method.Alg() != j.SigningMethod.Alg() {
			return nil, fmt.Errorf("%w: unexpected signing method %s", ErrTokenInvalid, token.Header["alg"])
		}

		return j.Secret, nil
	}, parserOptions...)

	// Handle common JWT parsing errors with more specific error types
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("%w: token expired", ErrTokenExpired)
		}
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	// Additional validation - ensure token is marked as valid
	if !token.Valid {
		return nil, ErrClaimsInvalid
	}

	// Extract and type-assert the claims
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("%w: failed to cast claims", ErrTokenInvalid)
	}

	return claims, nil
}

// VerifyWithCustomClaims validates a JWT token with custom claims structure.
// Unlike Verify(), this method allows you to parse tokens with non-standard claims
// while still performing all security validations. The claims parameter should be
// a pointer to your custom claims struct.
//
// Example:
//
//	type CustomClaims struct {
//	    jwt.RegisteredClaims
//	    Role        string   `json:"role"`
//	    Permissions []string `json:"permissions"`
//	}
//
//	var claims CustomClaims
//	err := jwt.VerifyWithCustomClaims(tokenString, &claims)
//	if err != nil {
//	    // Handle invalid token
//	    return
//	}
//	userRole := claims.Role
func (j *JWT) VerifyWithCustomClaims(bearerToken string, claims jwt.Claims) error {
	if len(j.Secret) == 0 {
		return ErrNoSecret
	}

	// Configure parser options for expiry leeway if specified
	parserOptions := []jwt.ParserOption{}
	if j.ExpiryLeeway > 0 {
		parserOptions = append(parserOptions, jwt.WithLeeway(j.ExpiryLeeway))
	}

	// Parse and validate the token with custom claims
	token, err := jwt.ParseWithClaims(bearerToken, claims, func(token *jwt.Token) (any, error) {
		// Verify signing method matches expected method to prevent algorithm confusion attacks
		if token.Method.Alg() != j.SigningMethod.Alg() {
			return nil, fmt.Errorf("%w: unexpected signing method %s", ErrTokenInvalid, token.Header["alg"])
		}
		return j.Secret, nil
	}, parserOptions...)

	// Handle common JWT parsing errors
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return fmt.Errorf("%w: token expired", ErrTokenExpired)
		}
		return fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	// Additional validation - ensure token is marked as valid
	if !token.Valid {
		return ErrClaimsInvalid
	}

	return nil
}
