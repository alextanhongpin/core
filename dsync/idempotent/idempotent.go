package idempotent

import (
	"bytes"
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

	// cleanupTimeout bounds the Redis calls made after the task has finished
	// (committing the result or deleting the in-flight entry). Those calls use
	// a context detached from the caller so a canceled request still cleans up.
	cleanupTimeout = 5 * time.Second
)

var (
	// ErrRequestInFlight is returned when another process (or another
	// [Idempotent] instance) is still executing the task for the same key.
	// Callers usually retry later; over HTTP this maps to 409 Conflict (or
	// 425/429 with Retry-After).
	ErrRequestInFlight = errors.New("idempotent: request in flight")

	// ErrRequestMismatch is returned when a completed result exists for the key
	// but was produced by a different request (compared by JSON encoding).
	// This is a client error; over HTTP this maps to 422 Unprocessable Entity.
	ErrRequestMismatch = errors.New("idempotent: request mismatch")

	// ErrFunctionExecutionFailed is no longer returned by this package.
	//
	// Deprecated: kept only for backward compatibility.
	ErrFunctionExecutionFailed = errors.New("idempotent: function execution failed")

	// ErrLockConflict is returned when this caller lost ownership of the
	// in-flight entry, for example because it expired (LockTTL elapsed without
	// a successful renewal) or was replaced. The task's result was NOT stored.
	ErrLockConflict = errors.New("idempotent: lock expired or is already held by another process")

	// ErrEmptyKey is returned when an empty key is provided.
	ErrEmptyKey = errors.New("idempotent: key cannot be empty")

	// lockRefreshRatio controls how early the lock TTL is renewed (70 % of TTL).
	lockRefreshRatio = 0.7
)

type (
	// Task is a unit of work that takes a request K and produces a response V.
	// Both K and V must round-trip through encoding/json, because they are
	// stored in Redis. Use [HandlerFunc] to turn a plain function into a Task.
	Task[K, V any] interface {
		Do(ctx context.Context, req K) (V, error)
	}

	// Handler runs a [Task] at most once per key while the result is retained.
	//
	// Do returns the task's response, and shared reports whether that response
	// was replayed from Redis (true) rather than produced by this call (false).
	Handler[K, V any] interface {
		Do(ctx context.Context, key string, req K) (res V, shared bool, err error)
	}
)

// Store is the storage backend used by [Idempotent]. [Client] implements it
// on top of Redis; most callers use [NewWithRedis] and never touch Store
// directly. A custom implementation (e.g. a fake for tests) must make each
// method atomic.
type Store interface {
	// CompareAndDelete deletes key only if its current value equals oldValue,
	// returning [ErrLockConflict] otherwise.
	CompareAndDelete(ctx context.Context, key, oldValue string) error
	// CompareAndSwap replaces key with newValue and sets ttl only if its
	// current value equals oldValue, returning [ErrLockConflict] otherwise.
	CompareAndSwap(ctx context.Context, key string, oldValue, newValue string, ttl time.Duration) error
	// LoadOrStore stores value with ttl if key is absent and returns
	// (value, false, nil); otherwise it returns the existing value and true.
	LoadOrStore(ctx context.Context, key string, value string, ttl time.Duration) (curr string, loaded bool, err error)
}

// Idempotent creates idempotent [Handler]s that share one [Store]. It is safe
// for concurrent use; create one per process and reuse it.
type Idempotent struct {
	client Store
	cache  *cache.Cache[string, permit]
}

// HandlerConfig configures a [Handler]. The zero value is valid and selects
// the defaults (see [DefaultConfig]).
type HandlerConfig struct {
	// LockTTL is how long the in-flight Redis entry lives without renewal.
	// While the task runs the entry is renewed every 70% of LockTTL, so the
	// task may run longer than LockTTL; LockTTL only bounds how long a crashed
	// process blocks the key. Defaults to 10 seconds; minimum 1ms.
	LockTTL time.Duration

	// KeepTTL is how long the completed response is kept in Redis and replayed
	// to repeated requests. After it expires the same key runs the task again.
	// Defaults to 24 hours; minimum 1ms.
	KeepTTL time.Duration
}

// DefaultConfig returns a HandlerConfig with LockTTL=10s and KeepTTL=24h.
func DefaultConfig() HandlerConfig {
	return HandlerConfig{
		LockTTL: defaultLockTTL,
		KeepTTL: defaultKeepTTL,
	}
}

type permit struct{ ch chan struct{} }

// WithDefaults returns a copy of c with zero fields replaced by defaults.
func (c HandlerConfig) WithDefaults() HandlerConfig {
	if c.LockTTL == 0 {
		c.LockTTL = defaultLockTTL
	}
	if c.KeepTTL == 0 {
		c.KeepTTL = defaultKeepTTL
	}
	return c
}

// Validate reports an error if either TTL is below one millisecond. Call it
// after [HandlerConfig.WithDefaults]; the constructors do both for you.
func (c HandlerConfig) Validate() error {
	if c.LockTTL < time.Millisecond || c.KeepTTL < time.Millisecond {
		return errors.New("idempotent: TTLs must be at least one millisecond")
	}
	return nil
}

// NewWithRedis returns an Idempotent backed by client. The Redis server must
// support SET ... IFDEQ and DELEX (Redis 8.4+). The client is not closed by
// this package.
func NewWithRedis(client *redis.Client) (*Idempotent, error) {
	if client == nil {
		return nil, errors.New("idempotent: nil Redis client")
	}
	return New(NewClient(client))
}

// New returns an Idempotent backed by a custom [Store].
func New(client Store) (*Idempotent, error) {
	if client == nil {
		return nil, errors.New("idempotent: nil client")
	}
	return &Idempotent{client: client, cache: cache.New(func(string) (*permit, error) { return &permit{ch: make(chan struct{}, 1)}, nil })}, nil
}

// MustNew is like [New] but panics on error. Intended for program startup.
func MustNew(client Store) *Idempotent {
	i, err := New(client)
	if err != nil {
		panic(err)
	}
	return i
}

// MustNewWithRedis is like [NewWithRedis] but panics on error. Intended for
// program startup.
func MustNewWithRedis(client *redis.Client) *Idempotent {
	i, err := NewWithRedis(client)
	if err != nil {
		panic(err)
	}
	return i
}

// HandlerFunc is like [Idempotent.Handler] but takes a plain function.
func (i *Idempotent) HandlerFunc[K, V any](fn HandlerFunc[K, V], cfg HandlerConfig) (Handler[K, V], error) {
	if fn == nil {
		return nil, errors.New("idempotent: nil task")
	}
	return i.Handler(fn, cfg)
}

// MustHandlerFunc is like [Idempotent.HandlerFunc] but panics on error.
func (i *Idempotent) MustHandlerFunc[K, V any](fn HandlerFunc[K, V], cfg HandlerConfig) Handler[K, V] {
	h, err := i.HandlerFunc(fn, cfg)
	if err != nil {
		panic(err)
	}
	return h
}

// MustHandler is like [Idempotent.Handler] but panics on error.
func (i *Idempotent) MustHandler[K, V any](fn Task[K, V], cfg HandlerConfig) Handler[K, V] {
	h, err := i.Handler(fn, cfg)
	if err != nil {
		panic(err)
	}
	return h
}

// Handler wraps fn so that, for a given key, it runs at most once while the
// result is retained (KeepTTL). Zero fields in cfg take their defaults.
//
// Calling Do(ctx, key, req):
//   - first call for key: runs fn, stores (req, response) for KeepTTL and
//     returns (response, false, nil). If fn fails, nothing is stored and the
//     key is released so the request can be retried.
//   - same key and same req afterwards: returns (stored response, true, nil)
//     without running fn.
//   - same key, different req: returns [ErrRequestMismatch].
//   - same key while fn is still running elsewhere: callers in this process
//     wait for it; callers in other processes get [ErrRequestInFlight].
func (i *Idempotent) Handler[K, V any](fn Task[K, V], cfg HandlerConfig) (Handler[K, V], error) {
	if fn == nil {
		return nil, errors.New("idempotent: nil task")
	}
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return idempotentHandlerFunc[K, V](func(ctx context.Context, key string, req K) (V, bool, error) {
		var zero V
		if key == "" {
			return zero, false, ErrEmptyKey
		}
		if err := ctx.Err(); err != nil {
			return zero, false, context.Cause(ctx)
		}

		// Serialize same-key callers within this process, so they wait for
		// the result instead of receiving ErrRequestInFlight.
		p := i.getMutex(key)
		select {
		case p.ch <- struct{}{}:
		case <-ctx.Done():
			return zero, false, context.Cause(ctx)
		}
		defer func() { <-p.ch }()
		if ctx.Err() != nil {
			return zero, false, context.Cause(ctx)
		}

		inflight, err := json.Marshal(&data[K, V]{Token: uuid.NewV7(), Request: req})
		if err != nil {
			return zero, false, fmt.Errorf("marshaling in-flight entry: %w", err)
		}
		raw, loaded, err := i.client.LoadOrStore(ctx, key, string(inflight), cfg.LockTTL)
		if err != nil {
			return zero, false, fmt.Errorf("loading or storing: %w", err)
		}

		if loaded {
			// Key already exists: replay the stored result.
			res, err := parse[K, V](req, raw)
			if err != nil {
				return zero, false, err
			}
			return res, true, nil
		}

		// Key was freshly stored: run the function while holding the lease.
		res, err := i.runInLock(ctx, key, string(inflight), fn, req, cfg.LockTTL, cfg.KeepTTL)
		return res, false, err
	}), nil
}

func (i *Idempotent) getMutex(key string) *permit {
	p, _, _ := i.cache.LoadOrCreate(key)
	return p
}

// runInLock executes fn while holding the distributed Redis lease.
//
// A background goroutine renews the lease every lockRefreshRatio*lockTTL
// until fn returns. If a renewal fails, the task context is canceled and the
// result is discarded. If fn succeeds, the in-flight entry is atomically
// replaced with the completed response, even if the caller's context was
// canceled in the meantime (the work has already happened, so it must not be
// repeated). On any failure or panic the in-flight entry is deleted.
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

	// Renewal is tied to fn's lifetime, not the caller's context: a task that
	// keeps running after the caller cancels still owns the key.
	detached := context.WithoutCancel(ctx)
	var leaseErr error
	stop, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(time.Duration(float64(lockTTL) * lockRefreshRatio))
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				rctx, rcancel := context.WithTimeout(detached, lockTTL)
				err := i.client.CompareAndSwap(rctx, key, oldValue, oldValue, lockTTL)
				rcancel()
				if err != nil {
					leaseErr = fmt.Errorf("renewing lease: %w", err)
					cancel(leaseErr)
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
			cleanupCtx, cleanupCancel := context.WithTimeout(detached, cleanupTimeout)
			defer cleanupCancel()
			if cleanupErr := i.client.CompareAndDelete(cleanupCtx, key, oldValue); cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("releasing lease: %w", cleanupErr))
			}
		}
	}()

	res, err := fn.Do(workCtx, req)
	stopRefresh() // leaseErr is safe to read after this.
	if leaseErr != nil {
		return zero, errors.Join(err, leaseErr)
	}
	if err != nil {
		if cause := context.Cause(workCtx); cause != nil && !errors.Is(err, cause) {
			err = errors.Join(err, cause)
		}
		return zero, fmt.Errorf("executing function: %w", err)
	}
	b, err := json.Marshal(&data[K, V]{Done: true, Request: req, Response: res})
	if err != nil {
		return zero, fmt.Errorf("marshaling completed entry: %w", err)
	}
	commitCtx, commitCancel := context.WithTimeout(detached, cleanupTimeout)
	defer commitCancel()
	if err := i.client.CompareAndSwap(commitCtx, key, oldValue, string(b), keepTTL); err != nil {
		return zero, fmt.Errorf("storing completed response: %w", err)
	}
	succeeded = true
	return res, nil
}

// parse interprets the stored value for an existing key:
//  1. Not done → request is still in flight; return [ErrRequestInFlight].
//  2. Done, request mismatch → return [ErrRequestMismatch].
//  3. Done, request matches → return the cached response.
//
// Requests are compared on the exact JSON bytes that were stored, rather than
// after decoding into K, so lossy round-trips (e.g. large integers decoded as
// float64 when K is an interface) do not produce false mismatches.
func parse[K, V any](req K, raw string) (V, error) {
	var zero V
	var stored struct {
		Done     bool            `json:"done"`
		Request  json.RawMessage `json:"request"`
		Response json.RawMessage `json:"response"`
	}
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return zero, fmt.Errorf("unmarshaling stored entry: %w", err)
	}
	if !stored.Done {
		return zero, ErrRequestInFlight
	}
	incoming, err := json.Marshal(req)
	if err != nil {
		return zero, fmt.Errorf("marshaling incoming request: %w", err)
	}
	if !bytes.Equal(stored.Request, incoming) {
		return zero, ErrRequestMismatch
	}
	var res V
	if len(stored.Response) > 0 {
		if err := json.Unmarshal(stored.Response, &res); err != nil {
			return zero, fmt.Errorf("unmarshaling stored response of type %T: %w", res, err)
		}
	}
	return res, nil
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

// HandlerFunc adapts an ordinary function to the [Task] interface.
type HandlerFunc[K, V any] func(ctx context.Context, req K) (V, error)

// Do calls h(ctx, req).
func (h HandlerFunc[K, V]) Do(ctx context.Context, req K) (V, error) {
	return h(ctx, req)
}

type idempotentHandlerFunc[K, V any] func(ctx context.Context, key string, req K) (V, bool, error)

func (h idempotentHandlerFunc[K, V]) Do(ctx context.Context, key string, req K) (V, bool, error) {
	return h(ctx, key, req)
}
