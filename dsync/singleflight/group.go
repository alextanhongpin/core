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
	// ErrTimeout means a caller in another process waited WaitTTL for the
	// current leader to finish, and it had not.
	ErrTimeout = errors.New("group: timeout waiting for result")

	// ErrSubscriptionClosed means the Redis Pub/Sub connection used to wait for
	// the leader closed unexpectedly.
	ErrSubscriptionClosed = errors.New("group: subscription closed")
)

// OK is the message published on the lease channel when a leader releases its
// lease. Waiters treat it as a hint and confirm by checking the lease key.
const OK = "ok"

// Config configures a Group. Zero fields select the defaults.
type Config struct {
	// LockTTL is the lifetime of the leader's Redis lease. It is renewed every
	// 3/4 of LockTTL while the callback runs. Default 10s; minimum 1ms.
	LockTTL time.Duration
	// WaitTTL is how long a caller in another process waits for the leader
	// before giving up with ErrTimeout. Default 10s.
	WaitTTL time.Duration
	// PollInterval is how often a waiting caller checks whether the lease is
	// gone, in addition to listening on Pub/Sub. Default 10ms.
	PollInterval time.Duration
}

// WithDefaults returns c with zero durations replaced by their defaults.
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

// Validate reports whether all durations are positive and LockTTL is at least
// one millisecond. It does not apply defaults.
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

// Group makes sure that, for a given key, only one caller across all processes
// runs the callback at a time; the others wait for it to finish. It is safe for
// concurrent use. Construct it with New.
type Group struct {
	mu      sync.Mutex
	flights map[string]*flight
	cfg     Config
	client  *redis.Client
	locker  *lock.Client
}

// New returns a Group that coordinates through client. The caller keeps
// ownership of client and must close it after all calls finish. cfg is copied,
// defaulted and validated.
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

// MustNew is like New but panics on error. Use it during program startup.
func MustNew(client *redis.Client, cfg Config) *Group {
	g, err := New(client, cfg)
	if err != nil {
		panic(err)
	}
	return g
}

// leaseKey hashes key so arbitrary user keys map to a fixed-size Redis key in
// a reserved namespace. The same name is used as the Pub/Sub channel.
func leaseKey(key string) string {
	return fmt.Sprintf("singleflight:lease:%x", sha256.Sum256([]byte(key)))
}

// Do runs fn for key unless another caller is already running it, in which case
// it waits for that caller to finish instead. did is true only when this call
// ran fn and fn (plus lease renewal and release) succeeded; it is false for
// every waiter and for a leader that returns an error.
//
// Who runs fn:
//   - Callers on the same Group share one call; they all get the leader's error.
//   - Callers in other processes (or other Groups) wait until the leader's Redis
//     lease is released, then return (false, nil). They do NOT learn whether the
//     leader succeeded; use Cache to share a value.
//
// The context passed to fn is canceled when ctx is canceled or when the lease
// cannot be renewed. Do waits for fn to return even then, so fn must stop
// promptly. fn must not call Do with the same key: that deadlocks.
func (g *Group) Do(ctx context.Context, key string, fn func(context.Context) error) (did bool, err error) {
	if key == "" || fn == nil {
		return false, errors.New("singleflight: key and callback are required")
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
		// SET may have run even though we saw an error (e.g. a timeout after it
		// was sent). Delete our token so waiters are not blocked until expiry.
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = g.unlock(cleanup, key, token)
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

// wait blocks until the lease at key is gone. It returns ErrTimeout when waitTTL
// passes, or the cause of ctx when ctx ends first.
func (g *Group) wait(ctx context.Context, key string, waitTTL time.Duration) (err error) {
	wait, cancel := context.WithTimeout(ctx, waitTTL)
	defer cancel()
	defer func() {
		// Redis calls bound by the wait deadline fail with DeadlineExceeded;
		// report them consistently.
		if err == nil {
			return
		}
		if ctx.Err() != nil {
			err = context.Cause(ctx)
		} else if wait.Err() != nil {
			err = ErrTimeout
		}
	}()
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
