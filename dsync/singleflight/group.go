package singleflight

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"
	"uuid"

	"github.com/alextanhongpin/core/dsync/lock"
	redis "github.com/redis/go-redis/v9"
)

var (
	ErrTimeout            = errors.New("group: timeout waiting for result")
	ErrSubscriptionClosed = errors.New("group: subscription closed")
)

const OK = "ok"

// Config is copied at construction. Zero durations select defaults.
type Config struct{ LockTTL, WaitTTL, PollInterval time.Duration }

func (c Config) WithDefaults() Config {
	if c.LockTTL == 0 {
		c.LockTTL = 10 * time.Second
	}
	if c.WaitTTL == 0 {
		c.WaitTTL = 10 * time.Second
	}
	if c.PollInterval == 0 {
		c.PollInterval = 10 * time.Millisecond
	}
	return c
}
func (c Config) Validate() error {
	if c.LockTTL < time.Millisecond || c.WaitTTL <= 0 || c.PollInterval <= 0 {
		return errors.New("singleflight: positive lease, wait, and polling durations are required")
	}
	return nil
}

type flight struct {
	done chan struct{}
	err  error
}
type lease struct{ key, token string }
type leaseContextKey struct{}

// Group borrows a Redis client and owns local flight coordination. Callbacks
// execute synchronously; followers may cancel their wait independently.
type Group struct {
	mu      sync.Mutex
	flights map[string]*flight
	cfg     Config
	client  *redis.Client
	locker  *lock.Client
}

func New(client *redis.Client, cfg Config) (*Group, error) {
	if client == nil {
		return nil, errors.New("singleflight: Redis client is required")
	}
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Group{client: client, locker: lock.NewClient(client), cfg: cfg, flights: make(map[string]*flight)}, nil
}
func MustNew(client *redis.Client, cfg Config) *Group {
	g, err := New(client, cfg)
	if err != nil {
		panic(err)
	}
	return g
}
func leaseKey(key string) string {
	return fmt.Sprintf("singleflight:lease:%x", sha256.Sum256([]byte(key)))
}

func (g *Group) Do(ctx context.Context, key string, fn func(context.Context) error) (did bool, err error) {
	if key == "" || fn == nil {
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
	did, err = g.doOrWait(ctx, leaseKey(key), fn, g.cfg.LockTTL, g.cfg.WaitTTL)
	completed = true
	return did, err
}

func (g *Group) doOrWait(ctx context.Context, key string, fn func(context.Context) error, lockTTL, waitTTL time.Duration) (doOrWait bool, err error) {
	token := fmt.Sprint(uuid.NewV7())
	err = g.locker.Lock(ctx, key, token, lockTTL)
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
	work, cancel := context.WithCancelCause(context.WithValue(ctx, leaseContextKey{}, lease{key: key, token: token}))
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
				if e := g.locker.Extend(work, key, token, lockTTL); e != nil {
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
	err = errors.Join(err, context.Cause(work))
	if err == nil {
		err = g.locker.Extend(work, key, token, lockTTL)
	}
	return err
}

func (g *Group) wait(ctx context.Context, key string, waitTTL time.Duration) error {
	wait, cancel := context.WithTimeout(ctx, waitTTL)
	defer cancel()
	sub := g.client.Subscribe(wait, key)
	defer sub.Close()
	// Acknowledge subscription, then recheck for a release that raced with it.
	if _, err := sub.Receive(wait); err != nil {
		return err
	}
	messages := sub.Channel()
	ticker := time.NewTicker(g.cfg.PollInterval)
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
	status, err := g.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return status == 0, nil
}

func (g *Group) unlock(ctx context.Context, key, token string) error {
	err := g.locker.Unlock(ctx, key, token)
	if err != nil {
		return err
	}

	return g.client.Publish(ctx, key, OK).Err()
}
