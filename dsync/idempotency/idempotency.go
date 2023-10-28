package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

var (
	ErrRequestInFlight = errors.New("idempotency: request in flight")
	ErrRequestMismatch = errors.New("idempotency: request payload mismatch")
)

type Key string

func (k Key) Format(args ...any) string {
	return fmt.Sprintf(string(k), args...)
}

type Status string

const (
	Started Status = "started"
	Success Status = "success"
)

type data[T any] struct {
	Status   Status `json:"status"`
	Request  string `json:"request,omitempty"`
	Response T      `json:"response,omitempty"`
}

type store[T any] interface {
	Lock(ctx context.Context, idempotencyKey string, lockTimeout time.Duration) (bool, error)
	Unlock(ctx context.Context, idempotencyKey string) error
	Load(ctx context.Context, idempotencyKey string) (*data[T], error)
	Save(ctx context.Context, idempotencyKey string, d data[T], duration time.Duration) error
}

func hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	b := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(b)
}
