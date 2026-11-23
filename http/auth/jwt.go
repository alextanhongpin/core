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
	// When set, Verify also rejects tokens issued by anyone else.
	Issuer string

	// ExpiryLeeway is additional time added to token expiration during validation.
	// This accounts for clock skew between systems. Common values: 30s-5m.
	ExpiryLeeway time.Duration
}

// JWT manages the signing and verification of JWT tokens with configurable options.
// It provides a high-level interface for common JWT operations while allowing
// fine-grained control over token behavior. The fields mirror JWTConfig; a nil
// SigningMethod means HS256.
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
	token := jwt.NewWithClaims(j.signingMethod(), claims)
	return token.SignedString(j.Secret)
}

// SignWithCustomClaims signs a token with a custom claims type. Unlike Sign it
// does not set any registered claims; set ExpiresAt yourself, because Verify
// and VerifyWithCustomClaims reject tokens without an expiry.
//
// Example:
//
//	type CustomClaims struct {
//	    jwt.RegisteredClaims
//	    Role string `json:"role"`
//	}
//
//	token, err := j.SignWithCustomClaims(CustomClaims{
//	    RegisteredClaims: jwt.RegisteredClaims{
//	        Subject:   "user123",
//	        ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
//	    },
//	    Role: "admin",
//	})
func (j *JWT) SignWithCustomClaims(claims jwt.Claims) (string, error) {
	if len(j.Secret) == 0 {
		return "", ErrNoSecret
	}

	return jwt.NewWithClaims(j.signingMethod(), claims).SignedString(j.Secret)
}

// Verify validates a JWT token string and returns the parsed registered claims.
// It checks:
//   - the signature, using the configured secret and signing method only
//     (other algorithms, including "none", are rejected);
//   - that the token has an expiry ("exp") that has not passed, allowing
//     ExpiryLeeway for clock skew, and that "nbf" and "iat" are not in the
//     future;
//   - the issuer ("iss"), when Issuer is configured.
//
// Errors wrap ErrTokenExpired or ErrTokenInvalid.
func (j *JWT) Verify(bearerToken string) (*Claims, error) {
	claims := new(Claims)
	if err := j.VerifyWithCustomClaims(bearerToken, claims); err != nil {
		return nil, err
	}

	return claims, nil
}

// VerifyWithCustomClaims is like Verify, but decodes into claims, which must
// be a pointer to a type implementing jwt.Claims.
//
// Example:
//
//	var claims CustomClaims
//	if err := j.VerifyWithCustomClaims(tokenString, &claims); err != nil {
//	    return err
//	}
func (j *JWT) VerifyWithCustomClaims(bearerToken string, claims jwt.Claims) error {
	if len(j.Secret) == 0 {
		return ErrNoSecret
	}

	method := j.signingMethod()
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{method.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	}
	if j.ExpiryLeeway > 0 {
		opts = append(opts, jwt.WithLeeway(j.ExpiryLeeway))
	}
	if j.Issuer != "" {
		opts = append(opts, jwt.WithIssuer(j.Issuer))
	}

	_, err := jwt.ParseWithClaims(bearerToken, claims, func(*jwt.Token) (any, error) {
		return j.Secret, nil
	}, opts...)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return fmt.Errorf("%w: %w", ErrTokenExpired, err)
		}
		return fmt.Errorf("%w: %w", ErrTokenInvalid, err)
	}

	return nil
}

func (j *JWT) signingMethod() jwt.SigningMethod {
	if j.SigningMethod == nil {
		return jwt.SigningMethodHS256
	}

	return j.SigningMethod
}
