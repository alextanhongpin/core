// Package idempotent provides a mechanism for executing requests idempotently using Redis.
package idempotent

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sync/atomic"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/core/sync/promise"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

const (
	keepTTL = 24 * time.Hour
	lockTTL = 10 * time.Second
)

var (
	// ErrRequestInFlight indicates that a request is already in flight for the
	// specified key.
	ErrRequestInFlight = errors.New("idempotent: request in flight")

	// ErrRequestMismatch indicates that the request does not match the stored
	// request for the specified key.
	ErrRequestMismatch = errors.New("idempotent: request mismatch")

	// ErrHandlerNotFound indicates that the handler for the specified pattern
	// is not found.
	ErrHandlerNotFound = errors.New("idempoent: handler not found")
)

type locker interface {
	Extend(ctx context.Context, key, val string, ttl time.Duration) error
	LoadOrStore(ctx context.Context, key, token string, lockTTL time.Duration) (string, bool, error)
	Replace(ctx context.Context, key, oldVal, newVal string, ttl time.Duration) error
	Unlock(ctx context.Context, key, token string) error
}

type RedisStore struct {
	KeepTTL time.Duration
	Lock    locker
	LockTTL time.Duration
	client  *redis.Client
	group   *promise.Group[[]byte]
}

// NewRedisStore creates a new RedisStore instance with the specified Redis
// client, lock TTL, and keep TTL.
func NewRedisStore(client *redis.Client) *RedisStore {
	lock := lock.New(client)
	lock.LockTTL = 10 * time.Second
	lock.WaitTTL = 0

	return &RedisStore{
		client:  client,
		group:   promise.NewGroup[[]byte](),
		Lock:    lock,
		KeepTTL: keepTTL,
		LockTTL: lockTTL,
	}
}

// Do executes the provided function idempotently, using the specified key and
// request.
func (s *RedisStore) Do(ctx context.Context, key string, fn func(ctx context.Context, req []byte) ([]byte, error), req []byte) (res []byte, loaded bool, err error) {
	b := new(atomic.Bool)
	b.Store(true)
	res, err = s.group.DoAndForget(key, func() ([]byte, error) {
		res, loaded, err := s.do(ctx, key, fn, req)
		if !loaded {
			b.Store(loaded)
		}

		return res, err
	})
	loaded = b.Load()

	return
}

// loadOrStore returns the response for the specified key, or stores the request
func (s *RedisStore) loadOrStore(ctx context.Context, key string, req []byte) ([]byte, error) {
	v, loaded, err := s.Lock.LoadOrStore(ctx, key, newToken(), s.LockTTL)
	if err != nil {
		return nil, err
	}

	// There are two possible scenarios:
	// 1) The key/value pair exists. Process the value.
	// 2) The key/value pair does not exist. Proceed with the request.

	// 1)
	if loaded {
		return s.parse(req, []byte(v))
	}

	// 2)
	return []byte(v), errors.ErrUnsupported
}

func (s *RedisStore) runInLock(ctx context.Context, key, token string, fn func(context.Context, []byte) ([]byte, error), req []byte) ([]byte, error) {
	lock := s.Lock
	lockTTL := s.LockTTL
	keepTTL := s.KeepTTL

	// Any failure will just unlock the resource.
	// context.WithoutCancel ensures that the unlock is always called.
	// If the operation is successful, the token will be replaced with the
	// response, so the operation should fail.
	defer lock.Unlock(context.WithoutCancel(ctx), key, token)

	// Create a new channel to handle the result.
	ch := make(chan result[[]byte], 1)
	go func() {
		// Process the request in a separate goroutine.
		res, err := fn(ctx, req)
		ch <- result[[]byte]{
			err:  err,
			data: res,
		}

		close(ch)
	}()

	t := time.NewTicker(lockTTL * 7 / 10)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		case d := <-ch:
			// Extend once more to prevent token from expiring.
			if err := lock.Extend(ctx, key, token, lockTTL); err != nil {
				return nil, err
			}

			res, err := d.unwrap()
			if err != nil {
				return nil, err
			}

			b, err := json.Marshal(makeData(req, res))
			if err != nil {
				return nil, err
			}

			if err := lock.Replace(ctx, key, token, string(b), keepTTL); err != nil {
				return nil, err
			}

			return []byte(res), nil
		case <-t.C:
			if err := lock.Extend(ctx, key, token, lockTTL); err != nil {
				return nil, err
			}
		}
	}
}

func (s *RedisStore) do(ctx context.Context, key string, fn func(context.Context, []byte) ([]byte, error), req []byte) (res []byte, loaded bool, err error) {
	res, err = s.loadOrStore(ctx, key, req)
	if !errors.Is(err, errors.ErrUnsupported) {
		return res, err == nil, err
	}

	token := string(res)
	res, err = s.runInLock(ctx, key, token, fn, req)
	return res, false, err
}

// parse parses the value and returns the response if the request matches.
// There are two possible scenarios:
//  1. The value is a UUID, which means the request is in flight.
//  2. The value is a JSON object, which means the request has been processed.
//     2.1) The request does not match, return an error.
//     2.2) The request matches, return the response.
func (s *RedisStore) parse(req, value []byte) ([]byte, error) {
	// 1)
	if isPending(value) {
		return nil, ErrRequestInFlight
	}

	// 2)
	var d data
	if err := json.Unmarshal(value, &d); err != nil {
		return nil, err
	}

	// 2.1)
	if d.Request != string(hash(req)) {
		return nil, ErrRequestMismatch
	}

	// 2.2)
	return []byte(d.Response), nil
}

// hash generates a SHA-256 hash of the provided data.
// We hash the request because
// 1) The request may contain sensitive information.
// 2) The request may be too long to store in Redis.
// 3) We just need to compare the request, not the response.
func hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	b := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(b)
}

func isPending(b []byte) bool {
	return isUUID(b)
}

// isUUID checks if the provided byte slice represents a valid UUID.
func isUUID(b []byte) bool {
	_, err := uuid.ParseBytes(b)
	return err == nil
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
