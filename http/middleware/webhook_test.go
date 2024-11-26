package middleware_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/http/middleware"
	"github.com/alextanhongpin/testdump/httpdump"
	"github.com/stretchr/testify/assert"
)

func TestWebhookPayload(t *testing.T) {
	var (
		body   = []byte(`{"message":"hello"}`)
		secret = []byte("supersecret12345")
	)

	content := middleware.NewWebhookPayload(body)
	signature := content.Sign(secret)
	is := assert.New(t)
	is.True(content.Verify(signature, secret))
	is.False(content.Verify(signature, []byte("wrongsecret12345")))
}

func TestWebhook(t *testing.T) {
	var (
		body    = []byte(`{"message":"hello"}`)
		secret  = []byte("supersecret12345")
		content = middleware.NewWebhookPayload(body)
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

		wh := middleware.WebhookHandler(h, secret)
		httpdump.Handler(t, wh, httpdump.IgnoreRequestHeaders("X-Webhook-Id", "X-Webhook-Timestamp", "X-Webhook-Signature")).ServeHTTP(w, r)
	})

	t.Run("invalid", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(body))
		content.SignRequest(r, secret)

		wh := middleware.WebhookHandler(h, []byte("wrongsecret12345"))
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

			wh := middleware.WebhookHandler(h, oldSecret)
			httpdump.Handler(t, wh, httpdump.IgnoreRequestHeaders("X-Webhook-Id", "X-Webhook-Timestamp", "X-Webhook-Signature")).ServeHTTP(w, r)
		})

		t.Run("valid 2", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(body))
			content.SignRequest(r, oldSecret, newSecret)

			wh := middleware.WebhookHandler(h, newSecret)
			httpdump.Handler(t, wh, httpdump.IgnoreRequestHeaders("X-Webhook-Id", "X-Webhook-Timestamp", "X-Webhook-Signature")).ServeHTTP(w, r)
		})

		t.Run("valid 3", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(body))
			content.SignRequest(r, oldSecret, newSecret)

			wh := middleware.WebhookHandler(h, oldSecret, notSecret)
			httpdump.Handler(t, wh, httpdump.IgnoreRequestHeaders("X-Webhook-Id", "X-Webhook-Timestamp", "X-Webhook-Signature")).ServeHTTP(w, r)
		})

		t.Run("invalid", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(body))
			content.SignRequest(r, oldSecret, newSecret)

			wh := middleware.WebhookHandler(h, notSecret)
			httpdump.Handler(t, wh, httpdump.IgnoreRequestHeaders("X-Webhook-Id", "X-Webhook-Timestamp", "X-Webhook-Signature")).ServeHTTP(w, r)
		})
	})
}
