// Package idempotent provides Redis-backed idempotent request execution.
//
// A request identified by a key is guaranteed to run at most once; subsequent
// calls with the same key return the cached result without re-executing the
// function. This guarantee holds across multiple processes via Redis and within
// a single process via an in-memory key-level mutex.
package idempotent

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
	"uuid"

	"github.com/alextanhongpin/core/sync/cache"
	redis "github.com/redis/go-redis/v9"
)

const (
	defaultLockTTL = 10 * time.Second
	defaultKeepTTL = 24 * time.Hour
)

var (
	// ErrRequestInFlight is returned when a cross-process caller arrives while
	// the same key is already being processed.
	ErrRequestInFlight = errors.New("idempotent: request in flight")

	// ErrRequestMismatch is returned when the incoming request body differs from
	// the one stored under the same key.
	ErrRequestMismatch = errors.New("idempotent: request mismatch")

	// ErrFunctionExecutionFailed is returned when the worker goroutine's result
	// channel is closed before a value is sent (defensive; normally unreachable).
	ErrFunctionExecutionFailed = errors.New("idempotent: function execution failed")

	// ErrLockConflict is returned when a compare-and-swap fails because the
	// Redis entry was modified or expired by another process.
	ErrLockConflict = errors.New("idempotent: lock expired or is already held by another process")

	// ErrEmptyKey is returned when an empty key is provided.
	ErrEmptyKey = errors.New("idempotent: key cannot be empty")

	// lockRefreshRatio controls how early the lock TTL is renewed (70 % of TTL).
	lockRefreshRatio = 0.7
)

type (
	// Task represents an idempotent unit of work that can be executed.
	Task[K, V any] interface {
		Do(ctx context.Context, req K) (V, error)
	}

	// Handler executes tasks idempotently for a given key.
	Handler[K, V any] interface {
		Do(ctx context.Context, key string, req K) (V, bool, error)
	}
)

// client abstracts the Redis operations used by Idempotent.
type client interface {
	CompareAndDelete(ctx context.Context, key, oldValue string) error
	CompareAndSwap(ctx context.Context, key string, oldValue, newValue string, ttl time.Duration) error
	LoadOrStore(ctx context.Context, key string, value string, ttl time.Duration) (curr string, loaded bool, err error)
}

type Idempotent struct {
	client client
	cache  *cache.Cache[string, sync.Mutex]
}

type HandlerConfig struct {
	// LockTTL is how long the in-flight Redis entry is kept before expiring.
	// It should exceed the expected execution time of the handler function.
	// Defaults to 10 seconds.
	LockTTL time.Duration

	// KeepTTL is how long the completed response is cached in Redis.
	// Defaults to 24 hours.
	KeepTTL time.Duration
}

func DefaultConfig() *HandlerConfig {
	return &HandlerConfig{
		LockTTL: defaultLockTTL,
		KeepTTL: defaultKeepTTL,
	}
}

func NewWithRedis(client *redis.Client) *Idempotent {
	return New(NewClient(client))
}

func New(client client) *Idempotent {
	return &Idempotent{
		client: client,
		cache: cache.New(func(string) (*sync.Mutex, error) {
			return new(sync.Mutex), nil
		}),
	}
}

func (i *Idempotent) HandlerFunc[K, V any](fn HandlerFunc[K, V], cfg *HandlerConfig) Handler[K, V] {
	return i.Handler(fn, cfg)
}

func (i *Idempotent) Handler[K, V any](fn Task[K, V], cfg *HandlerConfig) Handler[K, V] {
	cfg = cmp.Or(cfg, DefaultConfig())
	lockTTL := cmp.Or(cfg.LockTTL, defaultLockTTL)
	keepTTL := cmp.Or(cfg.KeepTTL, defaultKeepTTL)
	if lockTTL < time.Millisecond || keepTTL < 0 {
		panic("idempotent: invalid TTL")
	}

	return idempotentHandlerFunc[K, V](func(ctx context.Context, key string, req K) (V, bool, error) {
		var zero V
		if key == "" {
			return zero, false, ErrEmptyKey
		}
		if err := ctx.Err(); err != nil {
			return zero, false, context.Cause(ctx)
		}

		mu := i.getMutex(key)
		mu.Lock()
		defer mu.Unlock()

		token := uuid.NewV7()
		payload, rawOldValue, loaded, err := i.loadOrStore(ctx, key, &data[K, V]{Token: token, Request: req}, lockTTL)
		if err != nil {
			return zero, false, err
		}

		if loaded {
			// Key already exists — parse the stored payload and return the result.
			res, err := i.parse(req, payload)
			if err != nil {
				return zero, false, err
			}
			return res, true, nil
		}

		// Key was freshly stored — run the function under the distributed lock.
		res, err := i.runInLock(ctx, key, rawOldValue, fn, req, lockTTL, keepTTL)
		return res, false, err
	})
}

func (i *Idempotent) getMutex(key string) *sync.Mutex {
	if i.cache == nil {
		return new(sync.Mutex)
	}
	mu, _, _ := i.cache.LoadOrCreate(key)
	return mu
}

// loadOrStore marshals val, stores it in Redis under key (SET NX with ttl),
// and returns the unmarshaled payload, the raw JSON string, and a loaded flag.
func (i *Idempotent) loadOrStore[T any](ctx context.Context, key string, val T, ttl time.Duration) (curr T, raw string, loaded bool, err error) {
	var zero T
	b, err := json.Marshal(val)
	if err != nil {
		return zero, "", false, fmt.Errorf("marshaling value of type %T: %w", val, err)
	}
	raw, loaded, err = i.client.LoadOrStore(ctx, key, string(b), ttl)
	if err != nil {
		return zero, "", false, fmt.Errorf("loading or storing: %w", err)
	}
	if !loaded {
		return val, string(b), false, nil
	}
	var v T
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return zero, raw, true, fmt.Errorf("unmarshaling value of type %T: %w", v, err)
	}
	return v, raw, true, nil
}

// runInLock executes fn while holding the distributed Redis lock.
// A background goroutine periodically refreshes the lock TTL so it does not expire
// during long operations. On success the in-flight entry is atomically replaced with
// the completed response. On any failure or panic the in-flight entry is deleted so
// the lock is released.
func (i *Idempotent) runInLock[K, V any](
	ctx context.Context,
	key string,
	oldValue string,
	fn Task[K, V],
	req K,
	lockTTL, keepTTL time.Duration,
) (value V, err error) {
	var zero V
	workCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	stop, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(time.Duration(float64(lockTTL) * lockRefreshRatio))
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-workCtx.Done():
				return
			case <-ticker.C:
				if err := i.client.CompareAndSwap(workCtx, key, oldValue, oldValue, lockTTL); err != nil {
					cancel(err)
					return
				}
			}
		}
	}()
	stopRefresh := sync.OnceFunc(func() { close(stop); <-stopped })
	succeeded := false
	defer func() {
		stopRefresh()
		if !succeeded {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cleanupCancel()
			if cleanupErr := i.client.CompareAndDelete(cleanupCtx, key, oldValue); cleanupErr != nil {
				err = errors.Join(err, cleanupErr)
			}
		}
	}()
	res, err := fn.Do(workCtx, req)
	stopRefresh()
	if workCtx.Err() != nil {
		return zero, errors.Join(err, context.Cause(workCtx))
	}
	if err != nil {
		return zero, fmt.Errorf("executing function: %w", err)
	}
	b, err := json.Marshal(&data[K, V]{Done: true, Request: req, Response: res})
	if err != nil {
		return zero, fmt.Errorf("marshaling completed entry: %w", err)
	}
	if err := i.client.CompareAndSwap(workCtx, key, oldValue, string(b), keepTTL); err != nil {
		return zero, fmt.Errorf("storing completed response: %w", err)
	}
	succeeded = true
	return res, nil
}

// parse interprets a stored payload for an existing key:
//  1. Not done → request is still in flight; return [ErrRequestInFlight].
//  2. Done, request mismatch → return [ErrRequestMismatch].
//  3. Done, request matches → return the cached response.
func (i *Idempotent) parse[K, V any](req K, payload *data[K, V]) (V, error) {
	var zero V
	if payload == nil {
		return zero, ErrRequestMismatch
	}
	if !payload.Done {
		return zero, ErrRequestInFlight
	}
	if err := jsonEqual(payload.Request, req); err != nil {
		return zero, err
	}
	return payload.Response, nil
}

// jsonEqual reports whether two request payloads serialize to equivalent JSON bytes,
// avoiding panics on structs with unexported fields and handling JSON normalization.
func jsonEqual[K any](stored, incoming K) error {
	storedBytes, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("marshaling stored request: %w", err)
	}
	incomingBytes, err := json.Marshal(incoming)
	if err != nil {
		return fmt.Errorf("marshaling incoming request: %w", err)
	}
	if !bytes.Equal(storedBytes, incomingBytes) {
		return ErrRequestMismatch
	}
	return nil
}

// data is the JSON value stored in Redis for an idempotent key.
//
// In-flight entry:  Done=false, Token=<uuid>, Request=<req>
// Completed entry:  Done=true,  Token=<zero>, Request=<req>, Response=<res>
type data[K, V any] struct {
	Done     bool      `json:"done"`
	Token    uuid.UUID `json:"token,omitzero"`
	Request  K         `json:"request"`
	Response V         `json:"response"`
}

type result[T any] struct {
	data T
	err  error
}

func (r result[T]) unwrap() (T, error) {
	return r.data, r.err
}

type HandlerFunc[K, V any] func(ctx context.Context, req K) (V, error)

func (h HandlerFunc[K, V]) Do(ctx context.Context, req K) (V, error) {
	return h(ctx, req)
}

type idempotentHandlerFunc[K, V any] func(ctx context.Context, key string, req K) (V, bool, error)

func (h idempotentHandlerFunc[K, V]) Do(ctx context.Context, key string, req K) (V, bool, error) {
	return h(ctx, key, req)
}
