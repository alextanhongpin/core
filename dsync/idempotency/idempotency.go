package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

var (
	ErrParallelRequest = errors.New("idempotency: parallel request running")
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

var keyTemplate = Key("idempotency:%s")

type Option[T comparable, V any] struct {
	ExecTimeout     time.Duration
	RetentionPeriod time.Duration
	Handler         func(ctx context.Context, req T) (V, error)
}

type Group[T comparable, V any] struct {
	client          *redis.Client
	execTimeout     time.Duration
	retentionPeriod time.Duration
	handler         func(ctx context.Context, req T) (V, error)
}

func New[T comparable, V any](client *redis.Client, opt Option[T, V]) *Group[T, V] {
	if opt.ExecTimeout == 0 {
		opt.ExecTimeout = 1 * time.Minute
	}
	if opt.RetentionPeriod == 0 {
		opt.RetentionPeriod = 24 * time.Hour
	}

	return &Group[T, V]{
		client:          client,
		execTimeout:     opt.ExecTimeout,
		retentionPeriod: opt.RetentionPeriod,
		handler:         opt.Handler,
	}
}

// TODO: Idempotent exec.
func (r *Group[T, V]) Query(ctx context.Context, idempotencyKey string, req T) (V, error) {
	k := keyTemplate.Format(idempotencyKey)

	// Set the idempotency operation status to started.
	var v V
	ok, err := r.client.SetNX(ctx, k, fmt.Sprintf(`{"status":%q}`, Started), r.execTimeout).Result()
	if err != nil {
		return v, err
	}

	// Successfully set, now do the idempotent operation and store the request/response.
	if ok {
		v, err = r.handler(ctx, req)
		if err != nil {
			// If there is an error, delete the key.
			return v, errors.Join(err, r.client.Del(ctx, k).Err())
		}

		return v, r.save(ctx, idempotencyKey, req, v, r.retentionPeriod)
	}

	// Already exists, fetch from cache.
	// The status may still be started.
	return r.load(ctx, idempotencyKey, req)
}

func (r *Group[T, V]) load(ctx context.Context, key string, req T) (V, error) {
	k := keyTemplate.Format(key)

	// Compare the request field first.
	var v V
	b, err := r.client.Get(ctx, k).Bytes()
	if err != nil {
		return v, err
	}

	var d data[T, V]
	if err := json.Unmarshal(b, &d); err != nil {
		return v, err
	}

	if d.Status == Started {
		return v, ErrParallelRequest
	}

	if d.Request != req {
		return v, ErrRequestMismatch
	}

	return d.Response, nil
}

func (r *Group[T, V]) save(ctx context.Context, key string, req T, res V, duration time.Duration) error {
	d := data[T, V]{
		Status:   Success,
		Request:  req,
		Response: res,
	}

	b, err := json.Marshal(d)
	if err != nil {
		return err
	}

	k := keyTemplate.Format(key)
	return r.client.Set(ctx, k, string(b), duration).Err()
}

type data[T, V any] struct {
	Status   Status `json:"status"`
	Request  T      `json:"request,omitempty"`
	Response V      `json:"response,omitempty"`
}
