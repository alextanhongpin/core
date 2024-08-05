// Package idempotent provides a mechanism for executing requests idempotently using Redis.
package idempotent

import (
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

	errDone = errors.New("idempotent: done")
)

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

type data[T any] struct {
	Request  string `json:"request,omitempty"`
	Response T      `json:"response,omitempty"`
}

type Idempotent[K comparable, V any] struct {
	client  *redis.Client
	lockTTL time.Duration
	keepTTL time.Duration
}

type Option struct {
	LockTTL time.Duration
	KeepTTL time.Duration
}

// New creates a new Idempotent instance with the specified Redis client, lock
// TTL, and keep TTL.
func New[K comparable, V any](client *redis.Client, opt *Option) *Idempotent[K, V] {
	if opt == nil {
		opt = &Option{
			LockTTL: 30 * time.Second,
			KeepTTL: 24 * time.Hour,
		}
	}
	return &Idempotent[K, V]{
		client:  client,
		lockTTL: opt.LockTTL,
		keepTTL: opt.KeepTTL,
	}
}

// Do executes the provided function idempotently, using the specified key and
// request.
func (i *Idempotent[K, V]) Do(ctx context.Context, key string, fn func(ctx context.Context, req K) (V, error), req K) (V, error) {
	// Check if the value exists in cache.
	res, err := i.load(ctx, key, req)

	// Return result if exists.
	if err == nil {
		return res, nil
	}

	// Return error if non-nil errors.
	if !errors.Is(err, redis.Nil) {
		return res, err
	}

	// The key does not exists yet, attempt to fill the cache.

	// Lock the key to ensure there are no duplicate request.
	val := []byte(uuid.New().String())

	ok, err := i.lock(ctx, key, val, i.lockTTL)

	// Lock should return true or false.
	// Otherwise, it is redis error.
	if err != nil {
		return res, err
	}

	// Unsuccessful lock, return the existing payload.
	if !ok {
		return i.load(ctx, key, req)
	}
	// Any failure will just unlock the resource.
	// context.WithoutCancel ensures that the unlock is always called.
	defer i.unlock(context.WithoutCancel(ctx), key, val)

	// If the lock is successful, perform the task and save the response.

	g, gctx := errgroup.WithContext(ctx)
	gctx, cancel := context.WithCancelCause(gctx)

	g.Go(func() error {
		err := i.refresh(gctx, key, val, i.lockTTL)
		if errors.Is(err, errDone) {
			return nil
		}

		return err
	})

	g.Go(func() error {
		// Cancel the refresh once this is done.
		defer cancel(errDone)

		res, err = fn(gctx, req)
		if err != nil {
			return err
		}

		// extend one more time allow enough time for the response to be written.
		return i.extend(gctx, key, val, i.lockTTL)
	})

	if err := g.Wait(); err != nil {
		return res, err
	}

	newval, err := json.Marshal(data[V]{
		Request:  i.hashRequest(req),
		Response: res,
	})
	if err != nil {
		return res, err
	}

	err = i.replace(ctx, key, val, newval, i.keepTTL)
	if err != nil {
		return res, err
	}

	return res, nil
}

// replace sets the value of the specified key to the provided new value, if
// the existing value matches the old value.
func (i *Idempotent[K, V]) replace(ctx context.Context, key string, oldVal, newVal []byte, ttl time.Duration) error {
	keys := []string{key}
	argv := []any{oldVal, newVal, ttl.Milliseconds()}
	err := replace.Run(ctx, i.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("replace: %w", ErrLeaseInvalid)
	}
	if err != nil {
		return fmt.Errorf("replace: %w", err)
	}

	return nil
}

func (i *Idempotent[K, V]) refresh(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	t := time.NewTicker(ttl * 7 / 10)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-t.C:
			if err := i.extend(ctx, key, val, i.lockTTL); err != nil {
				return err
			}
		}
	}
}

func (i *Idempotent[K, V]) extend(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	keys := []string{key}
	argv := []any{val, ttl.Milliseconds()}
	err := extend.Run(ctx, i.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("extend: %w", ErrLeaseInvalid)
	}
	if err != nil {
		return fmt.Errorf("extend: %w", err)
	}

	return nil
}

// load retrieves the value of the specified key and returns it as a data
// struct.
func (i *Idempotent[K, V]) load(ctx context.Context, key string, req K) (V, error) {
	var v V
	b, err := i.client.Get(ctx, key).Bytes()
	if err != nil {
		return v, err
	}

	// Check if the request is pending.
	if isUUID(b) {
		return v, ErrRequestInFlight
	}

	var d data[V]
	if err = json.Unmarshal(b, &d); err != nil {
		return v, err
	}

	// Check if the request matches.
	if d.Request != i.hashRequest(req) {
		return v, ErrRequestMismatch
	}

	return d.Response, nil
}

// hashRequest generates a hash of the provided request.
func (i *Idempotent[K, V]) hashRequest(req K) string {
	// We hash the request for several reasons
	// - we want to fix the size of the request, regardless of how large it is
	// - we do not need to diff if the request does not match
	// - we do not want to keep confidential data
	b, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	return hash(b)
}

// lock acquires a lock on the specified key using the provided value.
func (i *Idempotent[K, V]) lock(ctx context.Context, key string, val []byte, ttl time.Duration) (bool, error) {
	return i.client.SetNX(ctx, key, val, ttl).Result()
}

// unlock releases the lock on the specified key using the provided value.
func (i *Idempotent[K, V]) unlock(ctx context.Context, key string, val []byte) error {
	keys := []string{key}
	argv := []any{val}
	err := unlock.Run(ctx, i.client, keys, argv...).Err()
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
