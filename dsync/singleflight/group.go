package singleflight

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/core/sync/singleflight"
	redis "github.com/redis/go-redis/v9"
)

var ErrTimeout = errors.New("group: timeout waiting for result")

const OK = "ok"

type BackOff interface {
	Duration(i int) time.Duration
}

type Group struct {
	BackOff BackOff
	Client  *redis.Client
	Locker  *lock.Locker
	Group   *singleflight.Group[bool]
}

func New(client *redis.Client) *Group {
	return &Group{
		Client: client,
		Locker: lock.New(client),
		Group:  singleflight.New[bool](),
	}
}

func (g *Group) DoOrWait(ctx context.Context, key string, fn func(context.Context) error, lockTTL, waitTTL time.Duration) (doOrWait bool, err error) {
	did, shared, err := g.Group.Do(ctx, key, func(ctx context.Context) (bool, error) {
		return g.doOrWait(ctx, key, fn, lockTTL, waitTTL)
	})
	if err != nil {
		return false, err
	}

	return did && !shared, nil
}

func (g *Group) doOrWait(ctx context.Context, key string, fn func(context.Context) error, lockTTL, waitTTL time.Duration) (doOrWait bool, err error) {
	token, err := g.Locker.Lock(ctx, key, lockTTL)
	if errors.Is(err, lock.ErrLocked) {
		return false, g.wait(ctx, key, waitTTL)
	}

	if err != nil {
		return false, err
	}

	err = g.do(ctx, key, token, fn, lockTTL)
	return err == nil, err
}

func (g *Group) do(ctx context.Context, key string, token string, fn func(context.Context) error, lockTTL time.Duration) error {
	ch := make(chan error, 1)
	go func() {
		defer close(ch)

		ch <- fn(ctx)
	}()

	t := time.NewTicker(lockTTL * 3 / 4)
	defer t.Stop()

	defer g.unlock(context.WithoutCancel(ctx), key, token)

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-t.C:
			if err := g.Locker.Extend(ctx, key, token, lockTTL); err != nil {
				return err
			}
		case err := <-ch:
			return err
		}
	}
}

func (g *Group) wait(ctx context.Context, key string, waitTTL time.Duration) error {
	// Listen to done subscription.
	sub := g.Client.Subscribe(ctx, key)
	defer sub.Close()

	// Get expiry.
	duration, err := g.Client.PTTL(ctx, key).Result()
	if err != nil {
		return err
	}

	// Timeout after expiry.
	timeout := time.After(min(duration, waitTTL))

	var backOff BackOff
	if g.BackOff != nil {
		backOff = g.BackOff
	} else {
		backOff = NewExponentialBackOff(time.Second, time.Minute)
	}

	var i int
	for {
		select {
		case <-time.After(backOff.Duration(i)):
			ok, err := g.done(ctx, key)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
			i++
		case msg := <-sub.Channel():
			if msg.Channel != OK {
				continue
			}

			return nil
		case <-timeout:
			ok, err := g.done(ctx, key)
			if ok {
				return nil
			}

			return errors.Join(err, ErrTimeout)
		case <-ctx.Done():
			ok, err := g.done(ctx, key)
			if ok {
				return nil
			}

			return errors.Join(err, context.Cause(ctx))
		}
	}
}

func (g *Group) done(ctx context.Context, key string) (bool, error) {
	status, err := g.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return status == 0, nil
}

func (g *Group) unlock(ctx context.Context, key, token string) error {
	err := g.Locker.Unlock(ctx, key, token)
	if err != nil {
		return err
	}

	return g.Client.Publish(ctx, key, OK).Err()
}

type ExponentialBackOff struct {
	Base time.Duration
	Cap  time.Duration
}

func NewExponentialBackOff(base, cap time.Duration) *ExponentialBackOff {
	return &ExponentialBackOff{
		Base: base,
		Cap:  cap,
	}
}

func (b *ExponentialBackOff) Duration(i int) time.Duration {
	sleep := min(b.Cap, b.Base*time.Duration(math.Pow(2, float64(i))))
	return rand.N(sleep)
}
