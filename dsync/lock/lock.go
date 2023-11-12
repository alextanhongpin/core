package lock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

var (
	ErrLocked      = errors.New("lock: already locked")
	ErrKeyNotFound = errors.New("lock: key not found")
)

type locker interface {
	Extend(ctx context.Context, ttl time.Duration) error
	Unlock(ctx context.Context) error
}

type Locker struct {
	client *redis.Client
	prefix string
}

func New(client *redis.Client, prefix string) *Locker {
	fw := &Locker{
		client: client,
		prefix: prefix,
	}

	return fw
}

func (l *Locker) Lock(ctx context.Context, key string, ttl time.Duration) (locker, error) {
	// Generate a random uuid as the lock value.
	val := uuid.New().String()

	if err := l.lock(ctx, key, val, ttl); err != nil {
		return nil, err
	}

	return &lockable{
		key:    key,
		val:    val,
		Locker: l,
	}, nil
}

func (l *Locker) Do(ctx context.Context, key string, ttl time.Duration, fn func(ctx context.Context) error) error {
	locker, err := l.Lock(ctx, key, ttl)
	if err != nil {
		return err
	}
	defer locker.Unlock(ctx)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		select {
		case <-ctx.Done():
			return
		case errCh <- fn(ctx):
			return
		}
	}()

	go func() {
		// Periodically extend the lock duration until the operation is completed.
		t := time.NewTicker(ttl * 9 / 10)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				if err := locker.Extend(ctx, ttl); err != nil {
					errCh <- err
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return <-errCh
}

func (l *Locker) lock(ctx context.Context, key, val string, ttl time.Duration) error {
	keys := []string{l.buildKey(key)}
	argv := []any{val, formatMs(ttl)}
	err := lock.Run(ctx, l.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return ErrLocked
	}

	return err
}

func (l *Locker) unlock(ctx context.Context, key, val string) error {
	keys := []string{l.buildKey(key)}
	argv := []any{val}
	i, err := unlock.Run(ctx, l.client, keys, argv...).Result()
	if err != nil {
		return err
	}
	if i64 := i.(int64); i64 == 0 {
		return ErrKeyNotFound
	}

	return nil
}

func (l *Locker) extend(ctx context.Context, key, val string, ttl time.Duration) error {
	keys := []string{l.buildKey(key)}
	argv := []any{val, formatMs(ttl)}
	i, err := extend.Run(ctx, l.client, keys, argv...).Result()
	if err != nil {
		return err
	}

	if i64 := i.(int64); i64 == 0 {
		return ErrKeyNotFound
	}

	return nil
}

func (l *Locker) buildKey(key string) string {
	if l.prefix != "" {
		return fmt.Sprintf("%s:%s", l.prefix, key)
	}

	return key
}

type lockable struct {
	key string
	val string
	*Locker
}

func (l *lockable) Unlock(ctx context.Context) error {
	return l.unlock(ctx, l.key, l.val)
}

func (l *lockable) Extend(ctx context.Context, ttl time.Duration) error {
	return l.extend(ctx, l.key, l.val, ttl)
}

// copied from redis source code
func formatMs(dur time.Duration) int64 {
	if dur > 0 && dur < time.Millisecond {
		return 1
	}

	return int64(dur / time.Millisecond)
}
