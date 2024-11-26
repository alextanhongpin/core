package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/alextanhongpin/core/http/httputil"
	"github.com/google/uuid"
)

func WebhookHandler(h http.Handler, secrets ...[]byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, err := verify(r, secrets...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		h.ServeHTTP(w, r)
	})
}

func verify(r *http.Request, secrets ...[]byte) (bool, error) {
	content, err := NewWebhookPayloadFromRequest(r)
	if err != nil {
		return false, err
	}

	signatures := r.Header.Values("X-Webhook-Signature")
	for _, signature := range signatures {
		signedContent, err := base64.URLEncoding.DecodeString(signature)
		if err != nil {
			return false, err
		}
		for _, secret := range secrets {
			if content.Verify(signedContent, secret) {
				return true, nil
			}
		}
	}

	return false, nil
}

type WebhookPayload struct {
	ID   string
	At   time.Time
	Body []byte
}

func NewWebhookPayload(body []byte) *WebhookPayload {
	return &WebhookPayload{
		ID:   uuid.New().String(),
		Body: body,
		At:   time.Now(),
	}
}

func NewWebhookPayloadFromRequest(r *http.Request) (*WebhookPayload, error) {
	var (
		id = r.Header.Get("X-Webhook-Id")
		ts = r.Header.Get("X-Webhook-Timestamp")
	)

	b, err := httputil.ReadRequest(r)
	if err != nil {
		return nil, err
	}

	nsec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return nil, err
	}

	return &WebhookPayload{
		ID:   id,
		At:   time.Unix(0, nsec),
		Body: b,
	}, nil
}

func (s *WebhookPayload) SignRequest(r *http.Request, secrets ...[]byte) {
	r.Header.Set("X-Webhook-Id", s.ID)
	r.Header.Set("X-Webhook-Timestamp", fmt.Sprint(s.At.UnixNano()))
	for _, secret := range secrets {
		r.Header.Add("X-Webhook-Signature", base64.URLEncoding.EncodeToString(s.Sign(secret)))
	}
}

func (s *WebhookPayload) Sign(secret []byte) []byte {
	content := fmt.Sprintf("%s.%d.%s", s.ID, s.At.UnixNano(), string(s.Body))
	return hmacSHA256(secret, []byte(content))
}

func (s *WebhookPayload) Verify(signature, secret []byte) bool {
	return subtle.ConstantTimeCompare(s.Sign(secret), signature) == 1
}

func hmacSHA256(secret, data []byte) []byte {
	hmac := hmac.New(sha256.New, secret)
	hmac.Write([]byte(data))
	return hmac.Sum(nil)
}
