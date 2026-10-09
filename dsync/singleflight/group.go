package singleflight

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/core/sync/singleflight"
	redis "github.com/redis/go-redis/v9"
)

var (
	ErrTimeout            = errors.New("group: timeout waiting for result")
	ErrSubscriptionClosed = errors.New("group: subscription closed")
)

const OK = "ok"

type BackOff interface {
	Duration(i int) time.Duration
}

type flight struct {
	done chan struct{}
	err  error
}

type Group struct {
	mu      sync.Mutex
	flights map[string]*flight
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

func (g *Group) Do(ctx context.Context, key string, fn func(context.Context) error, lockTTL, waitTTL time.Duration) (did bool, err error) {
	if key == "" || fn == nil || lockTTL < time.Millisecond || waitTTL <= 0 {
		return false, fmt.Errorf("singleflight: key, callback, and positive lease/wait durations are required")
	}
	if err := context.Cause(ctx); err != nil {
		return false, err
	}
	g.mu.Lock()
	if g.flights == nil {
		g.flights = make(map[string]*flight)
	}
	if f, ok := g.flights[key]; ok {
		g.mu.Unlock()
		select {
		case <-ctx.Done():
			return false, context.Cause(ctx)
		case <-f.done:
			return false, f.err
		}
	}
	f := &flight{done: make(chan struct{})}
	g.flights[key] = f
	g.mu.Unlock()
	completed := false
	defer func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		if !completed {
			f.err = errors.New("singleflight: leader panicked")
		} else {
			f.err = err
		}
		delete(g.flights, key)
		close(f.done)
	}()
	did, err = g.doOrWait(ctx, key, fn, lockTTL, waitTTL)
	completed = true
	return did, err
}

func (g *Group) doOrWait(ctx context.Context, key string, fn func(context.Context) error, lockTTL, waitTTL time.Duration) (doOrWait bool, err error) {
	token, err := g.Locker.Lock(ctx, key, lockTTL)
	if errors.Is(err, lock.ErrLocked) {
		waitErr := g.wait(ctx, key, waitTTL)
		if waitErr == nil {
			return false, nil
		}

		ok, err := g.done(ctx, key)
		if ok {
			return false, nil
		}

		return false, errors.Join(waitErr, err)
	}

	if err != nil {
		return false, err
	}

	err = g.do(ctx, key, token, fn, lockTTL)
	return err == nil, err
}

func (g *Group) do(ctx context.Context, key string, token string, fn func(context.Context) error, lockTTL time.Duration) (err error) {
	work, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(lockTTL * 3 / 4)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-work.Done():
				return
			case <-ticker.C:
				if e := g.Locker.Extend(work, key, token, lockTTL); e != nil {
					cancel(e)
					return
				}
			}
		}
	}()
	joined := false
	join := func() {
		if !joined {
			close(stop)
			<-done
			joined = true
		}
	}
	defer func() {
		join()
		cleanup, c := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer c()
		err = errors.Join(err, g.unlock(cleanup, key, token))
	}()
	err = fn(work)
	join()
	return errors.Join(err, context.Cause(work))
}

func (g *Group) wait(ctx context.Context, key string, waitTTL time.Duration) error {
	wait, cancel := context.WithTimeout(ctx, waitTTL)
	defer cancel()
	sub := g.Client.Subscribe(wait, key)
	defer sub.Close()
	// Acknowledge subscription, then recheck for a release that raced with it.
	if _, err := sub.Receive(wait); err != nil {
		return err
	}
	messages := sub.Channel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	check := func() (bool, error) { return g.done(wait, key) }
	if ok, err := check(); ok || err != nil {
		return err
	}
	for {
		select {
		case <-ticker.C:
			if ok, err := check(); ok || err != nil {
				return err
			}
		case msg, ok := <-messages:
			if !ok {
				return ErrSubscriptionClosed
			}
			if msg.Payload == OK {
				if ok, err := check(); ok || err != nil {
					return err
				}
			}
		case <-wait.Done():
			if ctx.Err() != nil {
				return context.Cause(ctx)
			}
			return ErrTimeout
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

func (g *Group) backOffDuration(i int) time.Duration {
	if g.BackOff != nil {
		return g.BackOff.Duration(i)
	}

	return NewExponentialBackOff(time.Second, time.Minute).Duration(i)
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
