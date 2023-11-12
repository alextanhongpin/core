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
	ErrRequestInFlight = errors.New("idempotent: request in flight")
	ErrRequestMismatch = errors.New("idempotent: request mismatch")
	ErrKeyNotFound     = errors.New("idempotent: key not found")
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

func (i *Idempotent[K, V]) Do(ctx context.Context, key string, fn func(ctx context.Context, req K) (V, error), req K) (res V, err error) {
	// Check if value exists in cache.
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

func (i *Idempotent[K, V]) lock(ctx context.Context, key string, val []byte) (bool, error) {
	return i.client.SetNX(ctx, key, val, i.lockTTL).Result()
}

func (i *Idempotent[K, V]) unlock(ctx context.Context, key string, val []byte) error {
	keys := []string{key}
	argv := []any{val}
	unk, err := unlock.Run(ctx, i.client, keys, argv...).Result()
	if err != nil {
		return err
	}
	return parseScriptResult(unk)
}

func hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	b := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(b)
}

func isUUID(b []byte) bool {
	_, err := uuid.ParseBytes(b)
	return err == nil
}

// copied from redis source code
func formatMs(dur time.Duration) int64 {
	if dur > 0 && dur < time.Millisecond {
		return 1
	}

	return int64(dur / time.Millisecond)
}

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
