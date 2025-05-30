package lock

import (
	"cmp"
	"context"
	"errors"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type PubSub struct {
	Locker *Locker
	client *redis.Client
}

func NewPubSub(client *redis.Client) *PubSub {
	return &PubSub{
		Locker: New(client),
		client: client,
	}
}

func (l *PubSub) Do(ctx context.Context, key string, fn func(ctx context.Context) error, opts *LockOption) error {
	mu := l.Locker.mu.Key(key)
	mu.Lock()
	defer mu.Unlock()

	if opts == nil {
		opts = NewLockOption()
	}
	token := cmp.Or(opts.Token, newToken())

	var cancel context.CancelFunc
	if opts.RefreshRatio <= 0 {
		ctx, cancel = context.WithTimeoutCause(ctx, opts.Lock, ErrLockTimeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	if err := l.Lock(ctx, key, token, opts.Lock, opts.Wait); err != nil {
		return err
	}
	defer func() {
		if unlockErr := l.Unlock(context.WithoutCancel(ctx), key, token); unlockErr != nil {
			l.Locker.Logger.Error("failed to unlock", "key", key, "error", unlockErr)
		}
	}()

	// Create a channel with a buffer of 1 to prevent goroutine leak.
	ch := make(chan error, 1)

	go func() {
		ch <- fn(ctx)
		close(ch)
	}()

	t := time.NewTicker(time.Duration(float64(opts.Lock) * opts.RefreshRatio))
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case err := <-ch:
			return err
		case <-t.C:
			if err := l.Locker.Extend(ctx, key, token, opts.Lock); err != nil {
				return err
			}
		}
	}
}

func (l *PubSub) Lock(ctx context.Context, key, token string, ttl, wait time.Duration) error {
	if wait <= 0 {
		return l.Locker.TryLock(ctx, key, token, ttl)
	}

	// Fire at the timeout moment before the wait duration.
	timeout := time.After(wait)

	pubsub := l.client.Subscribe(ctx, key)
	defer pubsub.Close()

	var i int
	for {
		sleep := l.Locker.BackOff.BackOff(i)

		// Sleep for the remaining time before the key expires.
		select {
		case msg := <-pubsub.Channel():
			if msg.Payload != payload {
				continue
			}

			err := l.Locker.TryLock(ctx, key, token, ttl)
			if errors.Is(err, ErrLocked) {
				continue
			}

			return err
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-timeout:
			err := l.Locker.TryLock(ctx, key, token, ttl)
			if errors.Is(err, ErrLocked) {
				return ErrLockWaitTimeout
			}

			return err
		case <-time.After(sleep):
			err := l.Locker.TryLock(ctx, key, token, ttl)
			if errors.Is(err, ErrLocked) {
				i++
				continue
			}

			return err
		}
	}
}

// Unlocks the key with the given token.
func (l *PubSub) Unlock(ctx context.Context, key, token string) error {
	err := l.Locker.Unlock(ctx, key, token)
	if err != nil {
		return err
	}

	return l.client.Publish(ctx, key, payload).Err()
}
