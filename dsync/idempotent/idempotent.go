// Package idempotent provides a mechanism for executing requests idempotently using Redis.
package idempotent

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/google/uuid"
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

type cacheable interface {
	CompareAndDelete(ctx context.Context, key string, old []byte) error
	CompareAndSwap(ctx context.Context, key string, old, value []byte, ttl time.Duration) error
	LoadOrStore(ctx context.Context, key string, value []byte, ttl time.Duration) (curr []byte, loaded bool, err error)
}

type RedisStore struct {
	cache  cacheable
	client *redis.Client
	mu     *muKey
}

// NewRedisStore creates a new RedisStore instance with the specified Redis
// client, lock TTL, and keep TTL.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		cache:  cache.New(client),
		client: client,
		mu:     newMuKey(),
	}
}

// Do executes the provided function idempotently, using the specified key and
// request.
func (s *RedisStore) Do(ctx context.Context, key string, fn func(context.Context, []byte) ([]byte, error), req []byte, lockTTL, keepTTL time.Duration) (res []byte, loaded bool, err error) {
	mu := s.mu.Key(key)
	mu.Lock()
	defer mu.Unlock()

	res, err = s.loadOrStore(ctx, key, req, lockTTL)
	if !errors.Is(err, errors.ErrUnsupported) {
		return res, err == nil, err
	}

	token := string(res)
	res, err = s.runInLock(ctx, key, token, fn, req, lockTTL, keepTTL)
	return res, false, err
}

// loadOrStore returns the response for the specified key, or stores the request
func (s *RedisStore) loadOrStore(ctx context.Context, key string, req []byte, lockTTL time.Duration) ([]byte, error) {
	b, loaded, err := s.cache.LoadOrStore(ctx, key, []byte(newToken()), lockTTL)
	if err != nil {
		return nil, err
	}

	// There are two possible scenarios:
	// 1) The key/value pair exists. Process the value.
	// 2) The key/value pair does not exist. Proceed with the request.

	// 1)
	if loaded {
		return s.parse(req, b)
	}

	// 2)
	return b, errors.ErrUnsupported
}

func (s *RedisStore) runInLock(ctx context.Context, key, token string, fn func(context.Context, []byte) ([]byte, error), req []byte, lockTTL, keepTTL time.Duration) ([]byte, error) {
	// Any failure will just unlock the resource.
	// context.WithoutCancel ensures that the unlock is always called.
	// If the operation is successful, the token will be replaced with the
	// response, so the operation should fail.
	defer s.cache.CompareAndDelete(context.WithoutCancel(ctx), key, []byte(token))

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

			b, err := json.Marshal(makeData(req, res))
			if err != nil {
				return nil, err
			}

			// Replace the token with the response.
			if err := s.compareAndSwap(ctx, key, []byte(token), b, keepTTL); err != nil {
				return nil, err
			}

			// Return the response.
			return []byte(res), nil
		case <-t.C:
			// Extend the lock to prevent the token from expiring.
			if err := s.compareAndSwap(ctx, key, []byte(token), []byte(token), lockTTL); err != nil {
				return nil, err
			}
		}
	}
}

func (s *RedisStore) compareAndSwap(ctx context.Context, key string, old, value []byte, ttl time.Duration) error {
	err := s.cache.CompareAndSwap(ctx, key, old, value, ttl)
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
	return d.getResponseBytes()
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

type muKey struct {
	mu   sync.Mutex
	keys map[string]*sync.Mutex
}

func newMuKey() *muKey {
	return &muKey{
		keys: make(map[string]*sync.Mutex),
	}
}

func (m *muKey) Key(key string) sync.Locker {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.keys[key]; !ok {
		m.keys[key] = new(sync.Mutex)
	}

	return m.keys[key]
}
