// Package webhook provides secure webhook handling with HMAC signature verification,
// supporting multiple secrets for key rotation, timestamp validation for replay attack
// prevention, and compatibility with popular webhook providers like GitHub and Stripe.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alextanhongpin/core/http/request"
	"github.com/google/uuid"
)

func Handler(h http.Handler, secrets ...[]byte) http.Handler {
	config := DefaultConfig(secrets...)
	return HandlerWithConfig(h, config)
}

// Config holds webhook verification configuration
type Config struct {
	// Secrets for signature verification (supports key rotation)
	Secrets [][]byte
	// MaxAge is the maximum age of a webhook payload to accept (default: 5 minutes)
	MaxAge time.Duration
	// ErrorHandler handles verification errors (optional)
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig(secrets ...[]byte) Config {
	return Config{
		Secrets: secrets,
		MaxAge:  5 * time.Minute,
	}
}

// HandlerWithConfig creates a webhook handler with custom configuration
func HandlerWithConfig(h http.Handler, config Config) http.Handler {
	if config.MaxAge == 0 {
		config.MaxAge = 5 * time.Minute
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, err := verifyWithConfig(r, config)
		if err != nil {
			if config.ErrorHandler != nil {
				config.ErrorHandler(w, r, err)
				return
			}
			http.Error(w, "Webhook verification failed", http.StatusUnauthorized)
			return
		}
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func verifyWithConfig(r *http.Request, config Config) (bool, error) {
	content, err := NewPayloadFromRequest(r)
	if err != nil {
		return false, err
	}

	// Validate timestamp to prevent replay attacks
	if config.MaxAge > 0 {
		age := time.Since(content.At)
		if age > config.MaxAge {
			return false, fmt.Errorf("webhook payload too old: %v", age)
		}
		if age < -time.Minute {
			return false, fmt.Errorf("webhook payload from future: %v", -age)
		}
	}

	signatures := getSignaturesFromRequest(r)
	if len(signatures) == 0 {
		return false, fmt.Errorf("no webhook signature provided")
	}

	for _, signature := range signatures {
		signedContent, err := base64.URLEncoding.DecodeString(signature)
		if err != nil {
			continue // Try next signature
		}
		for _, secret := range config.Secrets {
			if content.Verify(signedContent, secret) {
				return true, nil
			}
		}
	}

	return false, nil
}

// getSignaturesFromRequest extracts signatures from various standard webhook headers
func getSignaturesFromRequest(r *http.Request) []string {
	var signatures []string

	// Custom format: X-Webhook-Signature (base64 encoded)
	if sigs := r.Header.Values("X-Webhook-Signature"); len(sigs) > 0 {
		signatures = append(signatures, sigs...)
	}

	// GitHub format: X-Hub-Signature-256 (sha256=<hex>)
	if sig := r.Header.Get("X-Hub-Signature-256"); sig != "" {
		// Convert from "sha256=<hex>" to base64
		if strings.HasPrefix(sig, "sha256=") {
			hexSig := sig[7:] // Remove "sha256=" prefix
			if decoded, err := hex.DecodeString(hexSig); err == nil {
				signatures = append(signatures, base64.URLEncoding.EncodeToString(decoded))
			}
		}
	}

	// Stripe format: X-Signature-256
	if sig := r.Header.Get("X-Signature-256"); sig != "" {
		signatures = append(signatures, sig)
	}

	return signatures
}

type Payload struct {
	ID   string
	At   time.Time
	Body []byte
}

func NewPayload(body []byte) *Payload {
	return &Payload{
		ID:   uuid.New().String(),
		Body: body,
		At:   time.Now(),
	}
}

func NewPayloadFromRequest(r *http.Request) (*Payload, error) {
	var (
		id = r.Header.Get("X-Webhook-Id")
		ts = r.Header.Get("X-Webhook-Timestamp")
	)

	// Validate required headers
	if id == "" {
		return nil, fmt.Errorf("missing X-Webhook-Id header")
	}
	if ts == "" {
		return nil, fmt.Errorf("missing X-Webhook-Timestamp header")
	}

	b, err := request.Read(r)
	if err != nil {
		return nil, err
	}

	nsec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp format: %w", err)
	}

	return &Payload{
		ID:   id,
		At:   time.Unix(0, nsec),
		Body: b,
	}, nil
}

func (s *Payload) SignRequest(r *http.Request, secrets ...[]byte) {
	r.Header.Set("X-Webhook-Id", s.ID)
	r.Header.Set("X-Webhook-Timestamp", fmt.Sprint(s.At.UnixNano()))
	for _, secret := range secrets {
		r.Header.Add("X-Webhook-Signature", base64.URLEncoding.EncodeToString(s.Sign(secret)))
	}
}

func (s *Payload) Sign(secret []byte) []byte {
	content := fmt.Sprintf("%s.%d.%s", s.ID, s.At.UnixNano(), string(s.Body))
	return hmacSHA256(secret, []byte(content))
}

func (s *Payload) Verify(signature, secret []byte) bool {
	return subtle.ConstantTimeCompare(s.Sign(secret), signature) == 1
}

func hmacSHA256(secret, data []byte) []byte {
	hmac := hmac.New(sha256.New, secret)
	hmac.Write([]byte(data))
	return hmac.Sum(nil)
}
