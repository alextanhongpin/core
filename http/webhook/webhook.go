// Package webhook signs and verifies webhook requests with HMAC-SHA256.
//
// A signature covers "<id>.<unix-nano-timestamp>.<body>" and is sent in
// X-Webhook-Signature (base64url), with the ID and timestamp in X-Webhook-Id
// and X-Webhook-Timestamp. Several signature headers may be sent and several
// secrets configured, to rotate keys. This is a custom scheme: it does not
// verify GitHub or Stripe signatures.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/alextanhongpin/core/http/request"
	"github.com/google/uuid"
)

// Verification errors. Verify wraps one of these.
var (
	ErrMissingHeader    = errors.New("webhook: missing header")
	ErrInvalidTimestamp = errors.New("webhook: invalid timestamp")
	ErrExpired          = errors.New("webhook: payload outside the accepted age")
	ErrInvalidSignature = errors.New("webhook: invalid signature")
	ErrBodyTooLarge     = errors.New("webhook: body too large")
)

// Handler verifies requests with DefaultConfig(secrets...) before calling h.
func Handler(h http.Handler, secrets ...[]byte) http.Handler {
	return HandlerWithConfig(h, DefaultConfig(secrets...))
}

// Config holds webhook verification configuration.
type Config struct {
	// Secrets for signature verification (supports key rotation).
	Secrets [][]byte
	// MaxAge is the maximum age of a payload (default 5 minutes; negative
	// disables the check). While the check is enabled, payloads more than a
	// minute in the future are rejected too.
	MaxAge time.Duration
	// MaxBodySize limits the body read for verification (default and upper
	// bound: request.MaxBodySize).
	MaxBodySize int64
	// ErrorHandler handles verification errors. By default it replies 413
	// for ErrBodyTooLarge and 401 otherwise.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig(secrets ...[]byte) Config {
	return Config{
		Secrets: secrets,
		MaxAge:  5 * time.Minute,
	}
}

// HandlerWithConfig verifies each request with config before calling h. The
// body is restored for h. Payload IDs are not tracked: deduplicate by
// X-Webhook-Id in h if replays within MaxAge matter.
func HandlerWithConfig(h http.Handler, config Config) http.Handler {
	onError := config.ErrorHandler
	if onError == nil {
		onError = defaultErrorHandler
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := Verify(r, config); err != nil {
			onError(w, r, err)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func defaultErrorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	if errors.Is(err, ErrBodyTooLarge) {
		http.Error(w, "Webhook payload too large", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, "Webhook verification failed", http.StatusUnauthorized)
}

// Verify checks the headers, age and signature of r and restores its body.
// Header and age checks run before the body is read.
func Verify(r *http.Request, config Config) error {
	if config.MaxAge == 0 {
		config.MaxAge = 5 * time.Minute
	}
	if config.MaxBodySize <= 0 || config.MaxBodySize > request.MaxBodySize {
		config.MaxBodySize = request.MaxBodySize
	}

	id, at, err := parseHeaders(r)
	if err != nil {
		return err
	}

	if config.MaxAge > 0 {
		age := time.Since(at)
		if age > config.MaxAge || age < -time.Minute {
			return fmt.Errorf("%w: %v", ErrExpired, age)
		}
	}

	signatures := r.Header.Values("X-Webhook-Signature")
	if len(signatures) == 0 {
		return fmt.Errorf("%w: X-Webhook-Signature", ErrMissingHeader)
	}

	if r.ContentLength > config.MaxBodySize {
		return ErrBodyTooLarge
	}
	b, err := request.Read(r)
	if errors.Is(err, request.ErrBodyTooLarge) || int64(len(b)) > config.MaxBodySize {
		return ErrBodyTooLarge
	}
	if err != nil {
		return err
	}

	content := &Payload{ID: id, At: at, Body: b}
	for _, signature := range signatures {
		signed, err := base64.URLEncoding.DecodeString(signature)
		if err != nil {
			continue
		}
		for _, secret := range config.Secrets {
			if content.Verify(signed, secret) {
				return nil
			}
		}
	}

	return ErrInvalidSignature
}

func parseHeaders(r *http.Request) (string, time.Time, error) {
	id := r.Header.Get("X-Webhook-Id")
	if id == "" {
		return "", time.Time{}, fmt.Errorf("%w: X-Webhook-Id", ErrMissingHeader)
	}

	ts := r.Header.Get("X-Webhook-Timestamp")
	if ts == "" {
		return "", time.Time{}, fmt.Errorf("%w: X-Webhook-Timestamp", ErrMissingHeader)
	}

	nsec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: %w", ErrInvalidTimestamp, err)
	}

	return id, time.Unix(0, nsec), nil
}

// Payload is a webhook message and the metadata covered by its signature.
type Payload struct {
	ID   string
	At   time.Time
	Body []byte
}

// NewPayload returns a payload with a random ID, timestamped now.
func NewPayload(body []byte) *Payload {
	return &Payload{
		ID:   uuid.New().String(),
		Body: body,
		At:   time.Now(),
	}
}

// NewPayloadFromRequest reads the ID, timestamp and body (up to
// request.MaxBodySize) of a signed request without verifying it.
func NewPayloadFromRequest(r *http.Request) (*Payload, error) {
	id, at, err := parseHeaders(r)
	if err != nil {
		return nil, err
	}

	b, err := request.Read(r)
	if err != nil {
		return nil, err
	}

	return &Payload{ID: id, At: at, Body: b}, nil
}

// SignRequest sets the ID and timestamp headers on r and adds one
// X-Webhook-Signature per secret. r's body must equal s.Body.
func (s *Payload) SignRequest(r *http.Request, secrets ...[]byte) {
	r.Header.Set("X-Webhook-Id", s.ID)
	r.Header.Set("X-Webhook-Timestamp", strconv.FormatInt(s.At.UnixNano(), 10))
	for _, secret := range secrets {
		r.Header.Add("X-Webhook-Signature", base64.URLEncoding.EncodeToString(s.Sign(secret)))
	}
}

// Sign returns the HMAC-SHA256 of "<id>.<unix-nano>.<body>" under secret.
func (s *Payload) Sign(secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	fmt.Fprintf(mac, "%s.%d.", s.ID, s.At.UnixNano())
	mac.Write(s.Body)
	return mac.Sum(nil)
}

// Verify reports, in constant time, whether signature is s signed with secret.
func (s *Payload) Verify(signature, secret []byte) bool {
	return hmac.Equal(s.Sign(secret), signature)
}
