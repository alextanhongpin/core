package webhook_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/webhook"
	"github.com/stretchr/testify/assert"
)

var ok = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestWebhookIgnoresProviderHeaders(t *testing.T) {
	// Regression: X-Hub-Signature-256 and X-Signature-256 were accepted, but
	// checked against this package's "<id>.<ts>.<body>" scheme, so real
	// GitHub/Stripe signatures never verified while the package claimed to
	// support them.
	body := []byte(`{"message":"hello"}`)
	secret := []byte("supersecret12345")
	content := webhook.NewPayload(body)

	r := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	r.Header.Set("X-Webhook-Id", content.ID)
	r.Header.Set("X-Webhook-Timestamp", fmt.Sprint(content.At.UnixNano()))
	r.Header.Set("X-Hub-Signature-256", fmt.Sprintf("sha256=%x", content.Sign(secret)))

	err := webhook.Verify(r, webhook.DefaultConfig(secret))
	assert.ErrorIs(t, err, webhook.ErrMissingHeader)
}

func TestWebhookInvalidSignatureUsesErrorHandler(t *testing.T) {
	// Regression: a wrong signature skipped ErrorHandler and wrote a bare 401.
	body := []byte(`{}`)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	webhook.NewPayload(body).SignRequest(r, []byte("other"))

	var got error
	h := webhook.HandlerWithConfig(ok, webhook.Config{
		Secrets: [][]byte{[]byte("secret")},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			got = err
			w.WriteHeader(http.StatusForbidden)
		},
	})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.ErrorIs(t, got, webhook.ErrInvalidSignature)
}

func TestWebhookBodyLimit(t *testing.T) {
	// Regression: the whole body was read into memory without a limit.
	body := bytes.Repeat([]byte("a"), 100)
	secret := []byte("secret")
	r := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	webhook.NewPayload(body).SignRequest(r, secret)

	h := webhook.HandlerWithConfig(ok, webhook.Config{Secrets: [][]byte{secret}, MaxBodySize: 10})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestWebhookChecksHeadersBeforeBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/", errReader{})
	r.Header.Set("X-Webhook-Id", "id")
	r.Header.Set("X-Webhook-Timestamp", "not-a-number")

	err := webhook.Verify(r, webhook.DefaultConfig([]byte("secret")))
	assert.ErrorIs(t, err, webhook.ErrInvalidTimestamp)
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("body must not be read") }
