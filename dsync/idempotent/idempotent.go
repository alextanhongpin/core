// Package idempotent provides a mechanism for executing requests idempotently using Redis.
package idempotent

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/core/sync/promise"
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
		reqBytes, err := json.Marshal(req)
		if err != nil {
			return res, false, err
		}

		resBytes, shared, err := store.Do(ctx, key, func(ctx context.Context, _ []byte) ([]byte, error) {
			res, err := h(ctx, req)
			if err != nil {
				return nil, err
			}

			return json.Marshal(res)
		}, reqBytes, opts...)
		if err != nil {
			return res, false, err
		}

		err = json.Unmarshal(resBytes, &res)
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

func (o *Options) Valid() error {
	if o.KeepTTL <= 0 {
		return errors.New("idempotent: keep ttl must be greater than 0")
	}

	if o.LockTTL <= 0 {
		return errors.New("idempotent: lock ttl must be greater than 0")
	}

	return nil
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
	group  *promise.Group[[]byte]
	lock   *lock.Locker
	opts   *Options
}

// NewRedisStore creates a new RedisStore instance with the specified Redis
// client, lock TTL, and keep TTL.
func NewRedisStore(client *redis.Client, opts *Options) *RedisStore {
	opts = cmp.Or(opts, NewOptions())
	if err := opts.Valid(); err != nil {
		panic(err)
	}

	return &RedisStore{
		client: client,
		group:  promise.NewGroup[[]byte](),
		lock: lock.New(client, &lock.Options{
			LockTTL: opts.LockTTL,
			WaitTTL: 0,
		}),
		opts: opts,
	}
}

func (s *RedisStore) Do(ctx context.Context, key string, fn func(ctx context.Context, req []byte) ([]byte, error), req []byte, opts ...Option) (res []byte, shared bool, err error) {
	o := s.opts.Clone()
	for _, opt := range opts {
		opt(o)
	}

	lockTTL := o.LockTTL
	lock := s.lock

	token := newFencingToken()
	v, loaded, err := lock.LoadOrStore(ctx, key, token, lockTTL)
	if err != nil {
		return res, false, err
	}
	if loaded {
		// This is the only place where `shared` can be true.
		res, err := s.parse(req, []byte(v))
		return res, err == nil, err
	}

	res, err = s.group.DoAndForget(key, func() ([]byte, error) {
		return s.do(ctx, key, token, fn, req, o)
	})
	return
}

// Do executes the provided function idempotently, using the specified key and
// request.
func (s *RedisStore) do(ctx context.Context, key, token string, fn func(ctx context.Context, req []byte) ([]byte, error), req []byte, o *Options) (res []byte, err error) {
	lock := s.lock
	keepTTL := o.KeepTTL
	lockTTL := o.LockTTL

	// Any failure will just unlock the resource.
	// context.WithoutCancel ensures that the unlock is always called.
	defer lock.Unlock(context.WithoutCancel(ctx), key, token)

	ch := make(chan result[data], 1)
	go func() {
		res, err := fn(ctx, req)
		if err != nil {
			ch <- result[data]{err: err}
		} else {
			ch <- result[data]{data: data{
				Request:  hash(req),
				Response: string(res),
			}}
		}
		close(ch)
	}()

	t := time.NewTicker(lockTTL * 7 / 10)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		case res := <-ch:
			d, err := res.unwrap()
			if err != nil {
				return nil, err
			}

			b, err := json.Marshal(d)
			if err != nil {
				return nil, err
			}

			if err := lock.Replace(ctx, key, token, string(b), keepTTL); err != nil {
				return nil, err
			}

			return []byte(d.Response), nil
		case <-t.C:
			if err := lock.Extend(ctx, key, token, lockTTL); err != nil {
				return nil, err
			}
		}
	}
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

type result[T any] struct {
	data T
	err  error
}

func (r result[T]) unwrap() (T, error) {
	return r.data, r.err
}

func makeResult[T any](data T, err error) result[T] {
	return result[T]{data: data, err: err}
}

func newFencingToken() string {
	return uuid.Must(uuid.NewV7()).String()
}
