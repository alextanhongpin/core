// Package cache provides a Lock-based cache implementation with atomic operations.
package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/alextanhongpin/dsync/channel"
	redis "github.com/redis/go-redis/v9"
)

// Lock provides a cache-based implementation of the Storage interface.
// It wraps a Lock client and provides atomic cache operations.
type Lock struct {
	cache  *Redis
	ch     *channel.Channel
	stream bool
}

// NewRedis creates a new Lock instance with the provided Lock client.
func NewLock(client *redis.Client) *Lock {
	return &Lock{
		cache: NewRedis(client),
		ch:    channel.New(client),
	}
}

type AdvisoryLockConfig struct {
	Do           func(context.Context, string, []byte) error
	Wait         time.Duration
	Lock         time.Duration
	RefreshRatio float32
	UseStream    bool
}

func (l *Lock) AdvisoryLock(ctx context.Context, key string, cfg *AdvisoryLockConfig) error {
	start := time.Now()
	token := fmt.Appendf(nil, "lock:%s", uuid.NewV7())
	err := l.cache.StoreOnce(ctx, key, token, cfg.Lock)
	// Locked.
	if errors.Is(err, ErrExists) {
		// Lock, no wait.
		if cfg.Wait <= 0 {
			return fmt.Errorf("%w: %s", ErrLocked, key)
		}
		// No stream.
		if !cfg.UseStream {
			select {
			case <-time.After(cfg.Wait):
				return l.AdvisoryLock(ctx, key, cfg)

			case <-ctx.Done():
				return context.Cause(ctx)
			}
		}

		// Wait for unlock.
		_, err := l.ch.Recv(ctx, key, cfg.Wait)

		// Stream is closed or unlocked.
		if errors.Is(err, redis.Nil) {
			return l.AdvisoryLock(ctx, key, cfg)
		}
		return err
	}

	refresh := time.Duration(cfg.RefreshRatio) * cfg.Lock
	if refresh > cfg.Lock {
		panic("must be shorter than refresh window")
	}

	run := func(ctx context.Context) error {
		defer func() {
			// Unlock.
			_ = l.cache.CompareAndDelete(context.WithoutCancel(ctx), key, []byte(token))

			// Close channel.
			if cfg.UseStream {
				_ = l.ch.Close(context.WithoutCancel(ctx), key)
			}
		}()

		return cfg.Do(ctx, key, token)
	}

	if refresh <= 0 {
		timeout := cfg.Lock - bufferDuration(time.Since(start))
		if timeout <= 0 {
			panic("invalid timeout")
		}
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return run(ctx)
	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- run(ctx)
	}()

	t := time.NewTicker(refresh)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			// Extend.
			err := l.cache.CompareAndSwap(ctx, key, token, token, cfg.Lock)
			if err != nil {
				return err
			}

		case err := <-errCh:
			return err
		}
	}
}

func (l *Lock) LoadOrCreateT[T any](ctx context.Context, key string, cfg *LoadOrCreateConfig[T]) (curr T, loaded bool, err error) {
	b, loaded, err := l.LoadOrCreate(ctx, key, &LoadOrCreateConfig[[]byte]{
		Create: func(ctx context.Context, key string) ([]byte, time.Duration, error) {
			v, t, err := cfg.Create(ctx, key)
			if err != nil {
				return nil, 0, err
			}
			b, err := json.Marshal(v)
			if err != nil {
				return nil, 0, err
			}

			return b, t, nil
		},
		Lock:         cfg.Lock,
		RefreshRatio: cfg.RefreshRatio,
		UseStream:    cfg.UseStream,
		Wait:         cfg.Wait,
	})
	var zero T
	if err != nil {
		return zero, false, err
	}
	var v T
	err = json.Unmarshal(b, &v)
	if err != nil {
		return zero, false, err
	}
	return v, loaded, nil
}

type LoadOrCreateConfig[T any] struct {
	Create       func(context.Context, string) (T, time.Duration, error)
	Lock         time.Duration
	RefreshRatio float32
	UseStream    bool
	Wait         time.Duration
}

// LoadOrCreate returns the existing value for the key if present. Otherwise, it
// stores and returns the given value. The loaded result is true if the value
// was loaded, false if stored.
// Also see usecase here: https://github.com/golang/go/issues/33762#issuecomment-523757434
func (l *Lock) LoadOrCreate(ctx context.Context, key string, cfg *LoadOrCreateConfig[[]byte]) (curr []byte, loaded bool, err error) {
	start := time.Now()
	token := fmt.Appendf(nil, "lock:%s", uuid.NewV7())
	curr, loaded, err = l.cache.LoadOrStore(ctx, key, token, cfg.Lock)
	if err != nil {
		return nil, false, err
	}
	if loaded {
		if bytes.HasPrefix(curr, []byte("lock:")) {
			if cfg.Wait <= 0 {
				return nil, false, fmt.Errorf("%w: %s", ErrLocked, key)
			}
			if !cfg.UseStream {
				select {
				case <-time.After(cfg.Wait):
					return l.LoadOrCreate(ctx, key, cfg)
				case <-ctx.Done():
					return nil, false, context.Cause(ctx)
				}
			}

			// Wait for unlock.
			b, err := l.ch.Recv(ctx, key, cfg.Wait)
			if err != nil {
				// Stream is closed or unlocked.
				if errors.Is(err, redis.Nil) {
					return l.LoadOrCreate(ctx, key, cfg)
				}

				return nil, false, err
			}

			return b, false, nil
		}

		return curr, true, nil
	}
	refresh := time.Duration(cfg.RefreshRatio) * cfg.Lock
	if refresh > cfg.Lock {
		panic("must be shorter than refresh window")
	}

	create := func(ctx context.Context) ([]byte, error) {
		defer func() {
			// Unlock.
			_ = l.cache.CompareAndDelete(context.WithoutCancel(ctx), key, []byte(token))

			// Close channel.
			if cfg.UseStream {
				_ = l.ch.Close(context.WithoutCancel(ctx), key)
			}
		}()

		res, ttl, err := cfg.Create(ctx, key)
		if err != nil {
			return nil, err
		}

		err = l.cache.CompareAndSwap(ctx, key, token, res, ttl)
		if err != nil {
			return nil, err
		}

		if cfg.UseStream {
			err = l.ch.Send(ctx, key, res)
			if err != nil {
				return nil, err
			}
		}
		return res, nil
	}

	if refresh <= 0 {
		timeout := cfg.Lock - bufferDuration(time.Since(start))
		if timeout <= 0 {
			panic("invalid timeout")
		}
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		res, err := create(ctx)
		if err != nil {
			return nil, false, err
		}

		return res, false, nil
	}

	type result struct {
		data []byte
		err  error
	}
	resCh := make(chan result, 1)

	go func() {
		data, err := create(ctx)
		resCh <- result{data: data, err: err}
	}()

	t := time.NewTicker(refresh)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			// Extend.
			err := l.cache.CompareAndSwap(ctx, key, token, token, cfg.Lock)
			if err != nil {
				return nil, false, err
			}

		case res := <-resCh:
			if res.err != nil {
				return nil, false, res.err
			}
			return res.data, false, nil
		}
	}
}

func bufferDuration(d time.Duration) time.Duration {
	n := int64(d)
	var count int
	for n > 100 {
		n /= 10
		count++
	}
	n = n + 5
	for range count {
		n *= 10
	}
	return time.Duration(n)
}
