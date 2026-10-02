// Package idempotent provides a mechanism for executing requests idempotently using Redis.
package idempotent

import (
	"context"
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"uuid"

	"github.com/alextanhongpin/core/sync/lock"
	"github.com/google/go-cmp/cmp"
	redis "github.com/redis/go-redis/v9"
)

var (
	// ErrRequestInFlight indicates that a request is already in flight for the
	// specified key.
	ErrRequestInFlight = errors.New("idempotent: request in flight")

	// ErrRequestMismatch indicates that the request does not match the stored
	// request for the specified key.
	ErrRequestMismatch = errors.New("idempotent: request mismatch")

	// ErrFunctionExecutionFailed indicates that the function execution failed.
	ErrFunctionExecutionFailed = errors.New("idempotent: function execution failed")

	// ErrEmptyKey indicates that an empty key was provided.
	ErrEmptyKey = errors.New("idempotent: key cannot be empty")

	// ErrLockConflict indicates that the lock has expired or is already held by another process.
	ErrLockConflict = errors.New("idempotent: lock expired or is already held by another process")

	// lockRefreshRatio defines when to refresh the lock (70% of TTL)
	lockRefreshRatio = 0.7
)

type client interface {
	CompareAndDelete(ctx context.Context, key, oldValue string) error
	CompareAndSwap(ctx context.Context, key string, oldValue, newValue string, ttl time.Duration) error
	LoadOrStore(ctx context.Context, key string, value string, ttl time.Duration) (curr string, loaded bool, err error)
}

type RedisStore struct {
	client client
	locker *lock.KeyLock
}

// NewRedisStore creates a new RedisStore instance with the specified Redis
// client, lock TTL, and keep TTL.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: NewClient(client),
		locker: lock.New(),
	}
}

// Do executes the provided function idempotently, using the specified key and
// request.
func (s *RedisStore) Do(ctx context.Context, key string, fn fun[JSON, JSON], req []byte, lockTTL, keepTTL time.Duration) (res []byte, loaded bool, err error) {
	l := s.locker.Lock(key)
	defer l.Unlock()

	token := uuid.NewV7()
	data, loaded, err := s.loadOrStore(ctx, key, &data{Token: token, Request: req}, lockTTL)
	if err != nil {
		return nil, false, err
	}

	// There are two possible scenarios:
	// 1) The key/value pair exists. Process the value.
	// 2) The key/value pair does not exist. Proceed with the request.
	if loaded {
		// 1)
		res, err := s.parse(req, data)
		if err != nil {
			return nil, false, err
		}

		return res, true, nil
	}
	// 2)
	res, err = s.runInLock(ctx, key, token, fn, req, lockTTL, keepTTL)
	return res, false, err
}

func (s *RedisStore) loadOrStore[T any](ctx context.Context, key string, val T, ttl time.Duration) (T, bool, error) {
	var zero T
	b, err := json.Marshal(val)
	if err != nil {
		return zero, false, fmt.Errorf("marshaling value of type %T: %w", val, err)
	}
	data, loaded, err := s.client.LoadOrStore(ctx, key, string(b), ttl)
	if err != nil {
		return zero, false, fmt.Errorf("loading or storing: %w", err)
	}
	var v T
	err = json.Unmarshal([]byte(data), &v)
	if err != nil {
		return zero, false, fmt.Errorf("unmarshaling value of type %T: %w", v, err)
	}
	return v, loaded, nil
}

func (s *RedisStore) runInLock(ctx context.Context, key string, token uuid.UUID, fn fun[JSON, JSON], req []byte, lockTTL, keepTTL time.Duration) ([]byte, error) {
	oldValueBytes, err := json.Marshal(&data{Token: token, Request: req})
	if err != nil {
		return nil, fmt.Errorf("marshaling: %w", err)
	}
	oldValue := string(oldValueBytes)
	// Any failure will just unlock the resource.
	// context.WithoutCancel ensures that the unlock is always called.
	// If the operation is successful, the token will be replaced with the
	// response, so the operation should fail.
	var done bool
	defer func() {
		// The value will change after successful execution, so ignore it.
		if done {
			return
		}
		err := s.client.CompareAndDelete(context.WithoutCancel(ctx), key, oldValue)
		if err != nil {
			slog.ErrorContext(ctx, "comparing and deleting", "err", err)
		}
	}()

	// Create a new channel to handle the result.
	ch := make(chan result[[]byte], 1)

	// Use a context with cancellation to ensure goroutine cleanup
	fnCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		defer close(ch)
		// Process the request in a separate goroutine.
		res, err := fn(fnCtx, req)
		ch <- result[JSON]{err: err, data: res}
	}()

	t := time.NewTicker(time.Duration(float64(lockTTL) * lockRefreshRatio))
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		case d, ok := <-ch:
			if !ok {
				return nil, ErrFunctionExecutionFailed
			}
			// Extend once more to prevent token from expiring.
			if err := s.client.CompareAndSwap(ctx, key, oldValue, oldValue, lockTTL); err != nil {
				return nil, fmt.Errorf("extending lease: %w", err)
			}

			res, err := d.unwrap()
			if err != nil {
				return nil, fmt.Errorf("executing function: %w", err)
			}

			newValueBytes, err := json.Marshal(&data{Request: req, Response: res})
			if err != nil {
				return nil, fmt.Errorf("marshaling response: %w", err)
			}

			// Replace the token with the response.
			if err := s.client.CompareAndSwap(ctx, key, oldValue, string(newValueBytes), keepTTL); err != nil {
				return nil, fmt.Errorf("updating final value: %w", err)
			}
			done = true

			// Return the response.
			return res, nil
		case <-t.C:
			// Extend the lock to prevent the token from expiring.
			if err := s.client.CompareAndSwap(ctx, key, oldValue, oldValue, lockTTL); err != nil {
				return nil, fmt.Errorf("extending lease: %w", err)
			}
		}
	}
}

// parse parses the value and returns the response if the request matches.
// There are two possible scenarios:
//  1. The value is a UUID, which means the request is in flight.
//  2. The value is a JSON object, which means the request has been processed.
//     2.1) The request does not match, return an error.
//     2.2) The request matches, return the response.
func (s *RedisStore) parse(req []byte, payload *data) ([]byte, error) {
	// 1)
	if payload.Token != uuid.Nil() {
		return nil, ErrRequestInFlight
	}

	// 2)
	// 2.1)
	err := jsonBytesDiff(payload.Request, req)
	if err != nil {
		return nil, err
	}

	// 2.2)
	return payload.Response, nil
}

func jsonBytesDiff(a, b []byte) error {
	var c, d any
	err := json.Unmarshal(a, &c)
	if err != nil {
		return fmt.Errorf("unmarshaling %q: %w", a, err)
	}
	err = json.Unmarshal(b, &d)
	if err != nil {
		return fmt.Errorf("unmarshaling %q: %w", b, err)
	}

	if diff := cmp.Diff(a, b); diff != "" {
		return fmt.Errorf("%w: %s", ErrRequestMismatch, diff)
	}
	return nil
}

type data struct {
	Token    uuid.UUID      `json:"token,omitzero"`
	Request  jsontext.Value `json:"request,omitempty"`
	Response jsontext.Value `json:"response,omitempty"`
}

type result[T any] struct {
	data T
	err  error
}

func (r result[T]) unwrap() (T, error) {
	return r.data, r.err
}
