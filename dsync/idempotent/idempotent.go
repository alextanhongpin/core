// Package idempotent provides a mechanism for executing requests idempotently using Redis.
package idempotent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/lock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/helper"
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

type RedisStore struct {
	client *redis.Client
	locker *lock.KeyLock
}

// NewRedisStore creates a new RedisStore instance with the specified Redis
// client, lock TTL, and keep TTL.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
		locker: lock.New(),
	}
}

// Do executes the provided function idempotently, using the specified key and
// request.
func (s *RedisStore) Do(ctx context.Context, key string, fn func(context.Context, []byte) ([]byte, error), req []byte, lockTTL, keepTTL time.Duration) (res []byte, loaded bool, err error) {
	l := s.locker.Lock(key)
	defer l.Unlock()

	data, loaded, err := s.loadOrStore(ctx, key, newToken(), lockTTL)
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

	res, err = s.runInLock(ctx, key, data, fn, req, lockTTL, keepTTL)
	return res, false, err
}

func (s *RedisStore) runInLock(ctx context.Context, key, token string, fn func(context.Context, []byte) ([]byte, error), req []byte, lockTTL, keepTTL time.Duration) ([]byte, error) {
	// Any failure will just unlock the resource.
	// context.WithoutCancel ensures that the unlock is always called.
	// If the operation is successful, the token will be replaced with the
	// response, so the operation should fail.
	defer func() {
		_ = s.compareAndDelete(context.WithoutCancel(ctx), key, token)
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
		select {
		case ch <- result[[]byte]{err: err, data: res}:
		case <-fnCtx.Done():
			// Context cancelled, don't send to channel
		}
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
			if err := s.compareAndSwap(ctx, key, []byte(token), []byte(token), lockTTL); err != nil {
				return nil, err
			}

			res, err := d.unwrap()
			if err != nil {
				return nil, err
			}

			b, err := json.Marshal(data{Request: req, Response: res})
			if err != nil {
				return nil, err
			}

			// Replace the token with the response.
			if err := s.compareAndSwap(ctx, key, []byte(token), b, keepTTL); err != nil {
				return nil, err
			}

			// Return the response.
			return res, nil
		case <-t.C:
			// Extend the lock to prevent the token from expiring.
			if err := s.compareAndSwap(ctx, key, []byte(token), []byte(token), lockTTL); err != nil {
				return nil, err
			}
		}
	}
}

// compareAndSwap swaps the old and new values for key if the value stored in
// the map is equal to old. The old value must be of a comparable type.
func (s *RedisStore) compareAndSwap(ctx context.Context, key string, old, value []byte, ttl time.Duration) error {
	_, err := s.client.SetIFDEQ(ctx, key, value, helper.DigestBytes(old), ttl).Result()
	if errors.Is(err, redis.Nil) {
		return ErrLockConflict
	}
	return err
}

// parse parses the value and returns the response if the request matches.
// There are two possible scenarios:
//  1. The value is a UUID, which means the request is in flight.
//  2. The value is a JSON object, which means the request has been processed.
//     2.1) The request does not match, return an error.
//     2.2) The request matches, return the response.
func (s *RedisStore) parse(req []byte, val string) ([]byte, error) {
	// 1)
	if isUUID(val) {
		return nil, ErrRequestInFlight
	}

	// 2)
	var d data
	if err := json.Unmarshal([]byte(val), &d); err != nil {
		return nil, err
	}

	// 2.1)
	if hash(d.Request) != hash(req) {
		return nil, fmt.Errorf("%w: \n%s", ErrRequestMismatch, cmp.Diff(d.Request, req))
	}

	// 2.2)
	return d.Response, nil
}

// CompareAndDelete deletes the entry for key if its value is equal to old. The
// old value must be of a comparable type.
// If there is no current value for key in the map, CompareAndDelete returns
// false (even if the old value is the nil interface value).
func (r *RedisStore) compareAndDelete(ctx context.Context, key, old string) error {
	n, err := r.client.DelExArgs(ctx, key, redis.DelExArgs{
		Mode:        "IFDEQ",
		MatchDigest: helper.DigestString(old),
	}).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return redis.Nil
	}
	return nil
}

// loadOrStore returns the existing value for the key if present. Otherwise, it
// stores and returns the given value. The loaded result is true if the value
// was loaded, false if stored.
// Also see usecase here: https://github.com/golang/go/issues/33762#issuecomment-523757434
func (r *RedisStore) loadOrStore(ctx context.Context, key string, value string, ttl time.Duration) (curr string, loaded bool, err error) {
	s, err := r.client.SetArgs(ctx, key, value, redis.SetArgs{
		Get:  true,
		Mode: string(redis.NX),
		TTL:  ttl,
	}).Result()
	// If the previous value does not exist when GET, then it will be nil.
	// But since we successfully set the value, we skip the error.
	if errors.Is(err, redis.Nil) {
		return value, false, nil
	}
	if err != nil {
		return "", false, err
	}

	return s, true, nil
}

// hash generates a uint64 of the provided data.
// We hash the request because
// 1) The request may contain sensitive information.
// 2) The request may be too long to store in Redis.
// 3) We just need to compare the request, not the response.
func hash(data []byte) uint64 {
	return helper.DigestBytes(data)
}

// isUUID checks if the provided byte slice represents a valid UUID.
func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

type data struct {
	Request  []byte `json:"request,omitempty"`
	Response []byte `json:"response,omitempty"`
}

type result[T any] struct {
	data T
	err  error
}

func (r result[T]) unwrap() (T, error) {
	return r.data, r.err
}

func newToken() string {
	return uuid.Must(uuid.NewV7()).String()
}
