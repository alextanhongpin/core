package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/alextanhongpin/core/http/request"
	"github.com/google/uuid"
)

func Handler(h http.Handler, secrets ...[]byte) http.Handler {
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
	content, err := NewPayloadFromRequest(r)
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

	b, err := request.Read(r)
	if err != nil {
		return nil, err
	}

	nsec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return nil, err
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
