package webhook_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/webhook"
	"github.com/alextanhongpin/testdump/httpdump"
	"github.com/stretchr/testify/assert"
)

func TestWebhookPayload(t *testing.T) {
	var (
		body   = []byte(`{"message":"hello"}`)
		secret = []byte("supersecret12345")
	)

	content := webhook.NewPayload(body)
	signature := content.Sign(secret)
	is := assert.New(t)
	is.True(content.Verify(signature, secret))
	is.False(content.Verify(signature, []byte("wrongsecret12345")))
}

func TestWebhook(t *testing.T) {
	var (
		body    = []byte(`{"message":"hello"}`)
		secret  = []byte("supersecret12345")
		content = webhook.NewPayload(body)
	)

	is := assert.New(t)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		is.True(bytes.Equal(b, body))

		w.WriteHeader(http.StatusOK)
	})

	t.Run("valid", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(body))
		content.SignRequest(r, secret)

		wh := webhook.Handler(h, secret)
		httpdump.Handler(t, wh, httpdump.IgnoreRequestHeaders("X-Webhook-Id", "X-Webhook-Timestamp", "X-Webhook-Signature")).ServeHTTP(w, r)
	})

	t.Run("invalid", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(body))
		content.SignRequest(r, secret)

		wh := webhook.Handler(h, []byte("wrongsecret12345"))
		httpdump.Handler(t, wh, httpdump.IgnoreRequestHeaders("X-Webhook-Id", "X-Webhook-Timestamp", "X-Webhook-Signature")).ServeHTTP(w, r)
	})

	t.Run("multiple signature", func(t *testing.T) {
		var (
			oldSecret = []byte("supersecret12345")
			newSecret = []byte("supersecret54321")
			notSecret = []byte("wrongsecret12345")
		)

		t.Run("valid 1", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(body))
			content.SignRequest(r, oldSecret, newSecret)

			wh := webhook.Handler(h, oldSecret)
			httpdump.Handler(t, wh, httpdump.IgnoreRequestHeaders("X-Webhook-Id", "X-Webhook-Timestamp", "X-Webhook-Signature")).ServeHTTP(w, r)
		})

		t.Run("valid 2", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(body))
			content.SignRequest(r, oldSecret, newSecret)

			wh := webhook.Handler(h, newSecret)
			httpdump.Handler(t, wh, httpdump.IgnoreRequestHeaders("X-Webhook-Id", "X-Webhook-Timestamp", "X-Webhook-Signature")).ServeHTTP(w, r)
		})

		t.Run("valid 3", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(body))
			content.SignRequest(r, oldSecret, newSecret)

			wh := webhook.Handler(h, oldSecret, notSecret)
			httpdump.Handler(t, wh, httpdump.IgnoreRequestHeaders("X-Webhook-Id", "X-Webhook-Timestamp", "X-Webhook-Signature")).ServeHTTP(w, r)
		})

		t.Run("invalid", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(body))
			content.SignRequest(r, oldSecret, newSecret)

			wh := webhook.Handler(h, notSecret)
			httpdump.Handler(t, wh, httpdump.IgnoreRequestHeaders("X-Webhook-Id", "X-Webhook-Timestamp", "X-Webhook-Signature")).ServeHTTP(w, r)
		})
	})
}

func TestWebhookTimestampValidation(t *testing.T) {
	var (
		body   = []byte(`{"message":"hello"}`)
		secret = []byte("supersecret12345")
	)

	is := assert.New(t)

	t.Run("recent timestamp should pass", func(t *testing.T) {
		content := webhook.NewPayload(body)

		config := webhook.Config{
			Secrets: [][]byte{secret},
			MaxAge:  5 * time.Minute,
		}

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
		content.SignRequest(req, secret)

		handler := webhook.HandlerWithConfig(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}), config)

		handler.ServeHTTP(w, req)
		is.Equal(http.StatusOK, w.Code)
	})

	t.Run("old timestamp should fail", func(t *testing.T) {
		content := &webhook.Payload{
			ID:   "test-id",
			At:   time.Now().Add(-10 * time.Minute), // 10 minutes ago
			Body: body,
		}

		config := webhook.Config{
			Secrets: [][]byte{secret},
			MaxAge:  5 * time.Minute,
		}

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
		content.SignRequest(req, secret)

		handler := webhook.HandlerWithConfig(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}), config)

		handler.ServeHTTP(w, req)
		is.Equal(http.StatusUnauthorized, w.Code)
	})
}

func TestWebhookGitHubSignature(t *testing.T) {
	var (
		body   = []byte(`{"message":"hello"}`)
		secret = []byte("supersecret12345")
	)

	is := assert.New(t)
	content := webhook.NewPayload(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))

	// Set headers manually for GitHub format
	req.Header.Set("X-Webhook-Id", content.ID)
	req.Header.Set("X-Webhook-Timestamp", fmt.Sprint(content.At.UnixNano()))

	// Calculate signature and set as GitHub format
	signature := content.Sign(secret)
	req.Header.Set("X-Hub-Signature-256", fmt.Sprintf("sha256=%x", signature))

	handler := webhook.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), secret)

	handler.ServeHTTP(w, req)
	is.Equal(http.StatusOK, w.Code)
}
