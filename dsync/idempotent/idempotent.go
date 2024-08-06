// Package idempotent provides a mechanism for executing requests idempotently using Redis.
package idempotent

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
)

var (
	// ErrRequestInFlight indicates that a request is already in flight for the
	// specified key.
	ErrRequestInFlight = errors.New("idempotent: request in flight")

	// ErrRequestMismatch indicates that the request does not match the stored
	// request for the specified key.
	ErrRequestMismatch = errors.New("idempotent: request mismatch")

	// ErrLeaseInvalid indicates that the caller doesn't hold the correct token
	// for the lease and failed to write.
	ErrLeaseInvalid = errors.New("idempotent: lease invalid")
)

type data struct {
	Request  string `json:"request,omitempty"`
	Response string `json:"response,omitempty"`
}

type Handler[K, V any] func(ctx context.Context, key string, req K) (V, bool, error)

func (h Handler[K, V]) Do(ctx context.Context, key string, req K) (V, bool, error) {
	return h(ctx, key, req)
}

func MakeHandler[K, V any](store Store, h func(context.Context, K) (V, error), opts ...Option) Handler[K, V] {
	return func(ctx context.Context, key string, req K) (res V, shared bool, err error) {
		b, err := json.Marshal(req)
		if err != nil {
			return res, false, err
		}

		b, shared, err = store.Do(ctx, key, func(ctx context.Context, _ []byte) ([]byte, error) {
			res, err = h(ctx, req)
			if err != nil {
				return nil, err
			}

			return json.Marshal(res)
		}, b, opts...)
		if err != nil {
			return res, false, err
		}
		if !shared {
			return res, false, nil
		}

		if err := json.Unmarshal(b, &res); err != nil {
			return res, false, err
		}

		return res, shared, err
	}
}

type Options struct {
	KeepTTL time.Duration
	LockTTL time.Duration
}

func (o *Options) Clone() *Options {
	return &Options{
		KeepTTL: o.KeepTTL,
		LockTTL: o.LockTTL,
	}
}

func NewOptions() *Options {
	return &Options{
		KeepTTL: 24 * time.Hour,
		LockTTL: 10 * time.Second,
	}
}

type Option func(*Options)

func WithKeepTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.KeepTTL = ttl
	}
}

func WithLockTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.LockTTL = ttl
	}
}

type Store interface {
	Do(ctx context.Context, key string, fn func(ctx context.Context, req []byte) ([]byte, error), req []byte, opts ...Option) (res []byte, shared bool, err error)
}

type RedisStore struct {
	client *redis.Client
	opts   *Options
}

// NewRedisStore creates a new RedisStore instance with the specified Redis
// client, lock TTL, and keep TTL.
func NewRedisStore(client *redis.Client, opts *Options) *RedisStore {
	return &RedisStore{
		client: client,
		opts:   cmp.Or(opts, NewOptions()),
	}
}

// Do executes the provided function idempotently, using the specified key and
// request.
func (s *RedisStore) Do(ctx context.Context, key string, fn func(ctx context.Context, req []byte) ([]byte, error), req []byte, opts ...Option) (res []byte, shared bool, err error) {
	o := s.opts.Clone()
	for _, opt := range opts {
		opt(o)
	}

	token := uuid.New().String()
	b, loaded, err := s.loadOrStore(ctx, key, token, o.LockTTL)
	if err != nil {
		return res, false, err
	}
	if loaded {
		res, err := s.parse(req, b)
		return res, err == nil, err
	}

	// Any failure will just unlock the resource.
	// context.WithoutCancel ensures that the unlock is always called.
	defer s.unlock(context.WithoutCancel(ctx), key, token)

	ch := make(chan data)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return s.refresh(ctx, ch, key, token, o.LockTTL, o.KeepTTL)
	})

	g.Go(func() error {
		res, err = fn(ctx, req)
		if err != nil {
			return err
		}

		ch <- data{
			Request:  hash(req),
			Response: string(res),
		}
		close(ch)

		return nil
	})

	if err := g.Wait(); err != nil {
		return res, false, err
	}

	return res, false, nil
}

func (s *RedisStore) refresh(ctx context.Context, ch chan data, key, value string, lockTTL, keepTTL time.Duration) error {
	t := time.NewTicker(lockTTL * 7 / 10)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case d := <-ch:
			b, err := json.Marshal(d)
			if err != nil {
				return err
			}

			return s.replace(ctx, key, value, string(b), keepTTL)
		case <-t.C:
			if err := s.extend(ctx, key, value, lockTTL); err != nil {
				return err
			}
		}
	}
}

// replace sets the value of the specified key to the provided new value, if
// the existing value matches the old value.
func (s *RedisStore) replace(ctx context.Context, key string, oldVal, newVal string, ttl time.Duration) error {
	keys := []string{key}
	argv := []any{oldVal, newVal, ttl.Milliseconds()}
	err := replace.Run(ctx, s.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("replace: %w", ErrLeaseInvalid)
	}
	if err != nil {
		return fmt.Errorf("replace: %w", err)
	}

	return nil
}

func (s *RedisStore) extend(ctx context.Context, key, value string, ttl time.Duration) error {
	keys := []string{key}
	argv := []any{value, ttl.Milliseconds()}
	err := extend.Run(ctx, s.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("extend: %w", ErrLeaseInvalid)
	}
	if err != nil {
		return fmt.Errorf("extend: %w", err)
	}

	return nil
}

func (s *RedisStore) parse(req, b []byte) ([]byte, error) {
	// Check if the request is pending.
	if isUUID(b) {
		return nil, ErrRequestInFlight
	}

	var d data
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err
	}

	// Check if the request matches.
	if d.Request != string(hash(req)) {
		return nil, ErrRequestMismatch
	}

	return []byte(d.Response), nil
}

func (s *RedisStore) loadOrStore(ctx context.Context, key, value string, ttl time.Duration) ([]byte, bool, error) {
	v, err := s.client.Do(ctx, "SET", key, value, "NX", "GET", "PX", ttl.Milliseconds()).Result()
	// If the previous value does not exist when GET, then it will be nil.
	if errors.Is(err, redis.Nil) {
		return []byte(value), false, nil
	}

	if err != nil {
		return nil, false, err
	}

	return []byte(v.(string)), true, nil
}

// unlock releases the lock on the specified key using the provided value.
func (s *RedisStore) unlock(ctx context.Context, key, value string) error {
	keys := []string{key}
	argv := []any{value}
	err := unlock.Run(ctx, s.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("unlock: %w", ErrLeaseInvalid)
	}
	if err != nil {
		return fmt.Errorf("unlock: %w", err)
	}

	return nil
}

// hash generates a SHA-256 hash of the provided data.
func hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	b := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(b)
}

// isUUID checks if the provided byte slice represents a valid UUID.
func isUUID(b []byte) bool {
	_, err := uuid.ParseBytes(b)
	return err == nil
}

// unlock deletes the key only if the lease id matches.
var unlock = redis.NewScript(`
	-- KEYS[1]: The idempotency key
	-- ARGV[1]: The value value for optimistic locking
	local key = KEYS[1]
	local val = ARGV[1]

	if redis.call('GET', key) == val then
		return redis.call('DEL', key)
	end

	return nil
`)

// replace sets the value to the key only if the existing lease id matches.
var replace = redis.NewScript(`
	-- KEYS[1]: The idempotency key
	-- ARGV[1]: The old value for optimisic locking
	-- ARGV[2]: The new value
	-- ARGV[3]: How long to keep the idempotency key-value pair
	local key = KEYS[1]
	local old = ARGV[1]
	local new = ARGV[2]
	local ttl = ARGV[3]

	if redis.call('GET', key) == old then
		return redis.call('SET', key, new, 'XX', 'PX', ttl) 
	end

	return nil
`)

// extend extends the lock duration only if the lease id matches.
var extend = redis.NewScript(`
	-- KEYS[1]: key
	-- ARGV[1]: value
	-- ARGV[2]: lock duration in milliseconds
	local key = KEYS[1]
	local val = ARGV[1]
	local ttl_ms = tonumber(ARGV[2]) or 60000 -- Default 60s

	if redis.call('GET', key) == val then
		return redis.call('PEXPIRE', key, ttl_ms, 'GT')
	end

	return nil
`)
