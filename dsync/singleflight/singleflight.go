package singleflight

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/core/sync/promise"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

var (
	ErrConflict        = errors.New("singleflight: lock expired or acquired by another process")
	ErrLockWaitTimeout = errors.New("singleflight: failed to acquire lock within the wait duration")
)

type Options struct {
	KeepTTL time.Duration
	LockTTL time.Duration
	WaitTTL time.Duration
}

func NewOptions() *Options {
	return &Options{
		KeepTTL: 24 * time.Hour,
		LockTTL: 1 * time.Minute,
		WaitTTL: 30 * time.Second,
	}
}

func (o *Options) Valid() error {
	if o.LockTTL <= 0 {
		return errors.New("singleflight: lock ttl must be greater than 0")
	}
	if o.KeepTTL <= 0 {
		return errors.New("singleflight: keep ttl must be greater than 0")
	}
	if o.WaitTTL < 0 {
		o.WaitTTL = 0
	}

	return nil
}

type Group[T any] struct {
	client *redis.Client
	group  *promise.Group[T]
	lock   *lock.Locker
	opts   *Options
}

func New[T any](client *redis.Client, opts *Options) *Group[T] {
	opts = cmp.Or(opts, NewOptions())
	if err := opts.Valid(); err != nil {
		panic(err)
	}

	return &Group[T]{
		client: client,
		group:  promise.NewGroup[T](),
		opts:   opts,
		lock: lock.New(client, &lock.Options{
			LockTTL: opts.LockTTL,
			WaitTTL: 0,
		}),
	}
}

func (g *Group[T]) Do(ctx context.Context, key string, fn func(context.Context) (T, error)) (v T, err error, shared bool) {
	lockTTL := g.opts.LockTTL
	waitTTL := g.opts.WaitTTL

	token := newFencingToken()
	s, loaded, err := g.lock.LoadOrStore(ctx, key, token, lockTTL)
	if err != nil {
		return v, err, false
	}
	b := []byte(s)

	if loaded {
		// Completely loaded.
		if !g.isPending(b) {
			err = json.Unmarshal(b, &v)
			return v, err, err == nil
		}

		v, err = g.group.DoAndForget(key, func() (T, error) {
			return g.wait(ctx, key, waitTTL)
		})
		return v, err, err == nil
	}

	v, err = g.do(ctx, key, token, fn)
	return
}

func (g *Group[T]) do(ctx context.Context, key, token string, fn func(context.Context) (T, error)) (v T, err error) {
	lockTTL := g.opts.LockTTL
	keepTTL := g.opts.KeepTTL
	lock := g.lock

	// Use a separate context to avoid cancellation.
	defer lock.Unlock(context.WithoutCancel(ctx), key, token)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create a buffer of 1 to prevent goroutine leak.
	ch := make(chan result[T], 1)
	go func() {
		ch <- makeResult(fn(ctx))
		close(ch)
	}()

	t := time.NewTicker(lockTTL * 7 / 10)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return v, context.Cause(ctx)
		case res := <-ch:
			v, err := res.unwrap()
			if err != nil {
				return v, err
			}

			b, err := json.Marshal(v)
			if err != nil {
				return v, err
			}
			if err := lock.Replace(ctx, key, token, string(b), keepTTL); err != nil {
				return v, err
			}

			return v, nil
		case <-t.C:
			if err := lock.Extend(ctx, key, token, lockTTL); err != nil {
				return v, err
			}
		}
	}
}

func (g *Group[T]) wait(ctx context.Context, key string, ttl time.Duration) (v T, err error) {
	ctx, cancel := context.WithTimeoutCause(ctx, ttl, ErrLockWaitTimeout)
	defer cancel()

	var i int
	for {
		select {
		case <-ctx.Done():
			return v, context.Cause(ctx)
		case <-time.After(exponentialGrowthDecay(i)):
			i++
			b, err := g.client.Get(ctx, key).Bytes()
			if err != nil {
				return v, fmt.Errorf("wait: %w", err)
			}

			// Is pending.
			if g.isPending(b) {
				continue
			}

			err = json.Unmarshal(b, &v)
			return v, err
		}
	}
}

func (g *Group[T]) isPending(b []byte) bool {
	return isUUID(b)
}

// isUUID checks if the provided byte slice represents a valid UUID.
func isUUID(b []byte) bool {
	_, err := uuid.ParseBytes(b)
	return err == nil
}

func exponentialGrowthDecay(i int) time.Duration {
	x := float64(i)
	base := 1.0 + rand.Float64()
	switch {
	case x < 4: // intersection point rounded to 4
		base *= math.Pow(2, x)
	case x < 10:
		base *= 5 * math.Log(-0.9*x+10)
	default:
	}

	return time.Duration(base*100) * time.Millisecond
}

func newFencingToken() string {
	return uuid.Must(uuid.NewV7()).String()
}

type result[T any] struct {
	data T
	err  error
}

func (r *result[T]) unwrap() (T, error) {
	return r.data, r.err
}

func makeResult[T any](data T, err error) result[T] {
	return result[T]{data: data, err: err}
}
