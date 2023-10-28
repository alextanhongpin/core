package lock

import (
	"context"
	"errors"
	"fmt"
	"time"

	_ "embed"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

//go:embed lock.lua
var lock string

var ErrLocked = errors.New("lock: already locked")

type Locker struct {
	client *redis.Client
	prefix string
}

func New(client *redis.Client, prefix string) *Locker {
	fw := &Locker{
		client: client,
		prefix: prefix,
	}

	registerFunction(client)

	return fw
}

func (r *Locker) Lock(ctx context.Context, key, val string, ttl time.Duration) error {
	return r.lock(ctx, key, val, ttl)
}

func (r *Locker) Unlock(ctx context.Context, key, val string) error {
	return r.unlock(ctx, key, val)
}

func (r *Locker) Extend(ctx context.Context, key, val string, ttl time.Duration) error {
	return r.extend(ctx, key, val, ttl)
}

func (r *Locker) LockFunc(ctx context.Context, key string, fn func(ctx context.Context) error) error {
	val := uuid.New().String()
	ttl := 60 * time.Second
	if err := r.lock(ctx, key, val, ttl); err != nil {
		return err
	}
	defer r.unlock(ctx, key, val)

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
				if err := r.extend(ctx, key, val, ttl); err != nil {
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

func (r *Locker) lock(ctx context.Context, key, val string, ttl time.Duration) error {
	keys := []string{r.buildKey(key)}
	args := []any{val, ttl.Seconds()}
	err := r.client.FCall(ctx, "lock", keys, args...).Err()
	if errors.Is(err, redis.Nil) {
		return ErrLocked
	}

	return err
}

func (r *Locker) unlock(ctx context.Context, key, val string) error {
	keys := []string{r.buildKey(key)}
	args := []any{val}
	return r.client.FCall(ctx, "unlock", keys, args...).Err()
}

func (r *Locker) extend(ctx context.Context, key, val string, ttl time.Duration) error {
	keys := []string{r.buildKey(key)}
	args := []any{val, ttl.Seconds()}
	return r.client.FCall(ctx, "extend", keys, args...).Err()
}

func (r *Locker) buildKey(key string) string {
	if r.prefix != "" {
		return fmt.Sprintf("%s:%s", r.prefix, key)
	}

	return key
}

func registerFunction(client *redis.Client) {
	_, err := client.FunctionLoadReplace(context.Background(), lock).Result()
	if err != nil {
		if exists(err) {
			return
		}

		panic(err)
	}
}

func exists(err error) bool {
	// The ERR part is trimmed from prefix comparison.
	return redis.HasErrorPrefix(err, "Library 'lock' already exists")
}
