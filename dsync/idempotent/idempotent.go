// Package idempotent provides a mechanism for executing requests idempotently using Redis.
package idempotent

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
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

	// ErrHandlerNotFound indicates that the handler for the specified pattern
	// is not found.
	ErrHandlerNotFound = errors.New("idempoent: handler not found")
)

type data struct {
	Request  string `json:"request,omitempty"`
	Response string `json:"response,omitempty"`
}

type Options struct {
	KeepTTL time.Duration
	LockTTL time.Duration
}

func NewOptions() *Options {
	return &Options{
		KeepTTL: 24 * time.Hour,
		LockTTL: 10 * time.Second,
	}
}

func (o *Options) Apply(opts ...Option) *Options {
	for _, opt := range opts {
		opt(o)
	}

	return o
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
	Do(ctx context.Context, pattern, key string, req []byte, opts ...Option) (res []byte, shared bool, err error)
	HandleFunc(pattern string, fn func(ctx context.Context, req []byte) ([]byte, error))
}

type locker interface {
	Extend(ctx context.Context, key, val string, ttl time.Duration) error
	LoadOrStore(ctx context.Context, key, token string, lockTTL time.Duration) (string, bool, error)
	Replace(ctx context.Context, key, oldVal, newVal string, ttl time.Duration) error
	Unlock(ctx context.Context, key, token string) error
}

type RedisStore struct {
	client *redis.Client
	group  *promise.Group[[]byte]
	Lock   locker
	*Router
}

// NewRedisStore creates a new RedisStore instance with the specified Redis
// client, lock TTL, and keep TTL.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
		group:  promise.NewGroup[[]byte](),
		Lock: lock.New(client, &lock.Options{
			LockTTL: 10 * time.Second,
			WaitTTL: 0,
		}),
		Router: NewRouter(),
	}
}

// Do executes the provided function idempotently, using the specified key and
// request.
func (s *RedisStore) Do(ctx context.Context, pattern, key string, req []byte, opts ...Option) (res []byte, shared bool, err error) {
	fn, ok := s.Router.Handler(pattern)
	if !ok {
		return nil, false, fmt.Errorf("%w: %s", ErrHandlerNotFound, pattern)
	}

	b := new(atomic.Bool)
	b.Store(true)
	res, err = s.group.DoAndForget(key, func() ([]byte, error) {
		res, shared, err := s.do(ctx, key, fn, req, opts...)
		if !shared {
			b.Store(shared)
		}

		return res, err
	})
	shared = b.Load()

	return
}

func (s *RedisStore) do(ctx context.Context, key string, h Handler, req []byte, opts ...Option) (res []byte, shared bool, err error) {
	o := NewOptions()
	o.Apply(opts...)
	if err := o.Valid(); err != nil {
		return nil, false, err
	}

	keepTTL := o.KeepTTL
	lock := s.Lock
	lockTTL := o.LockTTL

	token := newToken()
	v, loaded, err := lock.LoadOrStore(ctx, key, token, lockTTL)
	if err != nil {
		return res, false, err
	}
	if loaded {
		// This is the only place where `shared` can be true.
		res, err := s.parse(req, []byte(v))
		return res, err == nil, err
	}

	// Any failure will just unlock the resource.
	// context.WithoutCancel ensures that the unlock is always called.
	defer lock.Unlock(context.WithoutCancel(ctx), key, token)

	ch := make(chan result[data], 1)
	go func() {
		res, err := h.Handle(ctx, req)
		if err != nil {
			ch <- result[data]{
				err: err,
			}
		} else {
			ch <- result[data]{
				data: data{
					Request:  hash(req),
					Response: string(res),
				},
			}
		}
		close(ch)
	}()

	t := time.NewTicker(lockTTL * 7 / 10)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, false, context.Cause(ctx)
		case res := <-ch:
			// Extend once more to prevent token from expiring.
			if err := lock.Extend(ctx, key, token, lockTTL); err != nil {
				return nil, false, err
			}

			d, err := res.unwrap()
			if err != nil {
				return nil, false, err
			}

			b, err := json.Marshal(d)
			if err != nil {
				return nil, false, err
			}

			if err := lock.Replace(ctx, key, token, string(b), keepTTL); err != nil {
				return nil, false, err
			}

			return []byte(d.Response), false, nil
		case <-t.C:
			if err := lock.Extend(ctx, key, token, lockTTL); err != nil {
				return nil, false, err
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

func newToken() string {
	return uuid.Must(uuid.NewV7()).String()
}
