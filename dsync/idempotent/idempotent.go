// Package idempotent provides a mechanism for executing requests idempotently using Redis.
package idempotent

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

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

	// ErrKeyNotFound indicates that the specified key does not exist.
	ErrKeyNotFound = errors.New("idempotent: key not found")
)

// unlock deletes the key only if the lease id matches.
var unlock = redis.NewScript(`
	local key = KEYS[1]
	local val = ARGV[1]

	if redis.call('GET', key) == val then
		return redis.call('DEL', key)
	end

	return 0
`)

// replace sets the value to the key only if the existing lease id matches.
var replace = redis.NewScript(`
	local key = KEYS[1]
	local old = ARGV[1]
	local new = ARGV[2]
	local ttl = ARGV[3]

	if redis.call('GET', key) == old then
		return redis.call('SET', key, new, 'PX', ttl) 
	end

	return 0
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
func (i *Idempotent[K, V]) Do(ctx context.Context, key string, fn func(ctx context.Context, req K) (V, error), req K) (res V, err error) {
	// Check if the value exists in cache.
	res, err = i.load(ctx, key, req)

	// Return result if exists.
	if err == nil {
		return
	}

	// Return error if non-nil errors.
	if !errors.Is(err, redis.Nil) {
		return
	}

	// The key does not exists yet, attempt to fill the cache.

	// Lock the key to ensure there are no duplicate request.
	val := []byte(uuid.New().String())

	var ok bool
	ok, err = i.lock(ctx, key, val)

	// Lock should return true or false.
	// Otherwise, it is redis error.
	if err != nil {
		return
	}

	// Unsuccessful lock, return the existing payload.
	if !ok {
		return i.load(ctx, key, req)
	}
	// Any failure will just unlock the resource.
	defer i.unlock(ctx, key, val)

	// If the lock is successful, perform the task and save the response.
	res, err = fn(ctx, req)
	if err != nil {
		return
	}

	d := data[V]{
		Request:  i.hashRequest(req),
		Response: res,
	}

	err = i.replace(ctx, key, val, d)

	return
}

// replace sets the value of the specified key to the provided new value, if
// the existing value matches the old value.
func (i *Idempotent[K, V]) replace(ctx context.Context, key string, oldVal []byte, newVal any) error {
	newVal, err := json.Marshal(newVal)
	if err != nil {
		return err
	}

	keys := []string{key}
	argv := []any{oldVal, newVal, formatMs(i.keepTTL)}
	unk, err := replace.Run(ctx, i.client, keys, argv...).Result()
	if err != nil {
		return err
	}

	return parseScriptResult(unk)
}

// load retrieves the value of the specified key and returns it as a data
// struct.
func (i *Idempotent[K, V]) load(ctx context.Context, key string, req K) (v V, err error) {
	var b []byte
	b, err = i.client.Get(ctx, key).Bytes()
	if err != nil {
		return
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
func (i *Idempotent[K, V]) lock(ctx context.Context, key string, val []byte) (bool, error) {
	return i.client.SetNX(ctx, key, val, i.lockTTL).Result()
}

// unlock releases the lock on the specified key using the provided value.
func (i *Idempotent[K, V]) unlock(ctx context.Context, key string, val []byte) error {
	keys := []string{key}
	argv := []any{val}
	unk, err := unlock.Run(ctx, i.client, keys, argv...).Result()
	if err != nil {
		return err
	}
	return parseScriptResult(unk)
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

// formatMs converts a time duration to milliseconds.
func formatMs(dur time.Duration) int64 {
	if dur > 0 && dur < time.Millisecond {
		return 1
	}

	return int64(dur / time.Millisecond)
}

// parseScriptResult interprets the result of a Redis script and returns an error if it indicates a failure.
func parseScriptResult(unk any) error {
	if unk == nil {
		return nil
	}

	switch v := unk.(type) {
	case string:
		if v == "OK" {
			return nil
		}
	case int64:
		if v == 1 {
			return nil
		}
	}

	return ErrKeyNotFound
}
