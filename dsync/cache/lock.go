package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"
	"uuid"

	redis "github.com/redis/go-redis/v9"
)

// Lock runs work for a key in at most one process at a time, using a Redis
// lease (a key with a TTL set by SET NX). It is meant for cache fills
// ([Lock.LoadOrCreate]) and short critical sections ([Lock.AdvisoryLock]).
//
// Lease keys are named "cache:lease:<sha256(key)>" and never clash with your
// data keys. The lease is released when the callback returns, and expires on
// its own if the process dies.
//
// A lease is not a fencing token: after a GC pause or network partition two
// processes can briefly both believe they hold it. Callbacks receive a context
// that is cancelled when the lease is lost and must stop promptly.
//
// The client stays owned by the caller. Lock requires a single Redis primary
// (not Cluster) running Redis 8.4 or newer.
type Lock struct {
	client *redis.Client
	cache  *Redis
}

// NewLock returns a Lock that uses client. It returns an error if client is
// nil.
func NewLock(client *redis.Client) (*Lock, error) {
	r, err := NewRedis(client)
	if err != nil {
		return nil, err
	}
	return &Lock{client: client, cache: r}, nil
}

// MustNewLock is like [NewLock] but panics on error.
func MustNewLock(client *redis.Client) *Lock {
	l, err := NewLock(client)
	if err != nil {
		panic(err)
	}
	return l
}

// AdvisoryLockConfig configures [Lock.AdvisoryLock].
type AdvisoryLockConfig struct {
	// Do is the work to run while holding the lease. Required. It receives
	// the original key and the lease token. Its context is cancelled when the
	// lease is lost, so long-running work must check ctx.
	Do func(ctx context.Context, key string, token []byte) error

	// Wait is how long to wait for another holder to release the lease.
	// Zero (the default) returns ErrLocked immediately if the lease is taken.
	Wait time.Duration

	// Lock is the lease TTL. Default 10s. Without renewal (RefreshRatio 0)
	// Do must finish within it, or its context is cancelled with ErrLocked.
	Lock time.Duration

	// RefreshRatio enables lease renewal: every Lock*RefreshRatio the lease
	// is extended back to Lock. Zero (default) disables renewal; otherwise it
	// must be in (0, 1). 0.5 is a good choice for work of unknown length.
	RefreshRatio float32

	// Deprecated: ignored; admission always polls.
	UseStream bool
}

// LoadOrCreateConfig configures [Lock.LoadOrCreate] and [Lock.LoadOrCreateT].
// Wait, Lock and RefreshRatio behave as in [AdvisoryLockConfig].
type LoadOrCreateConfig[T any] struct {
	// Create builds the value on a miss and returns it with its TTL (0 means
	// no expiry). Required. Its context is cancelled if the lease is lost.
	Create func(ctx context.Context, key string) (T, time.Duration, error)

	// Lock is the lease TTL. Default 10s.
	Lock time.Duration

	// RefreshRatio enables lease renewal. Default 0 (off).
	RefreshRatio float32

	// Deprecated: ignored; admission always polls.
	UseStream bool

	// Wait is how long to wait for another process's fill. Default 0: return
	// ErrLocked immediately. Set it so concurrent callers receive the value
	// the first caller creates instead of an error.
	Wait time.Duration
}

func leaseConfig(ttl, wait time.Duration, ratio float32) (time.Duration, error) {
	if ttl == 0 {
		ttl = 10 * time.Second
	}
	r := float64(ratio)
	if ttl < time.Millisecond || wait < 0 || math.IsNaN(r) || math.IsInf(r, 0) || r < 0 || r >= 1 || (r > 0 && time.Duration(float64(ttl)*r) < time.Nanosecond) {
		return 0, errors.New("cache: invalid lease duration, wait, or refresh ratio")
	}
	return ttl, nil
}

func leaseKey(key string) string { return fmt.Sprintf("cache:lease:%x", sha256.Sum256([]byte(key))) }

func (l *Lock) acquire(ctx context.Context, key string, ttl, wait time.Duration) ([]byte, error) {
	token := fmt.Appendf(nil, "%s", uuid.NewV7())
	if wait > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeoutCause(ctx, wait, errWaitTimeout)
		defer cancel()
	}
	for {
		if err := context.Cause(ctx); err != nil {
			return nil, err
		}
		ok, err := l.client.SetNX(ctx, key, token, ttl).Result()
		if err != nil {
			if cause := context.Cause(ctx); cause != nil {
				return nil, cause
			}
			return nil, err
		}
		if ok {
			return token, nil
		}
		if wait == 0 {
			return nil, ErrLocked
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, context.Cause(ctx)
		case <-timer.C:
		}
	}
}

// run owns renewal and joins it before committing or releasing the lease.
// User code runs synchronously, including when it panics.
func (l *Lock) run(ctx context.Context, key string, token []byte, ttl time.Duration, ratio float32, fn func(context.Context) error, commit func(context.Context) error) (err error) {
	work, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		interval := ttl
		if ratio > 0 {
			interval = time.Duration(float64(ttl) * float64(ratio))
		}
		timer := time.NewTimer(interval)
		defer timer.Stop()
		for {
			select {
			case <-stop:
				return
			case <-work.Done():
				return
			case <-timer.C:
				if ratio == 0 {
					cancel(errLeaseExpired)
					return
				}
				if e := l.renew(work, key, token, ttl); e != nil {
					cancel(e)
					return
				}
				timer.Reset(interval)
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
		e := l.cache.CompareAndDelete(cleanup, key, token)
		if errors.Is(e, ErrNotExist) {
			e = nil
		}
		err = errors.Join(err, e)
	}()
	err = fn(work)
	join()
	err = errors.Join(err, context.Cause(work))
	if err == nil && commit != nil {
		err = commit(work)
	}
	return err
}

// renew extends the lease if this process still owns it.
func (l *Lock) renew(ctx context.Context, key string, token []byte, ttl time.Duration) error {
	err := l.cache.CompareAndSwap(ctx, key, token, token, ttl)
	if errors.Is(err, ErrNotExist) {
		return fmt.Errorf("%w: lease lost: %w", ErrLocked, err)
	}
	return err
}

var (
	errWaitTimeout  = fmt.Errorf("%w: wait timed out: %w", ErrLocked, context.DeadlineExceeded)
	errLeaseExpired = fmt.Errorf("%w: lease expired before the callback returned", ErrLocked)
)

// AdvisoryLock runs cfg.Do while holding the lease for key, so at most one
// process runs it at a time. The lease is released when Do returns or panics.
//
// It returns ErrLocked if the lease is held elsewhere and cfg.Wait is zero or
// elapses (the latter also matches context.DeadlineExceeded), and an error
// matching ErrLocked if the lease was lost while Do ran. Otherwise it returns
// Do's error.
func (l *Lock) AdvisoryLock(ctx context.Context, key string, cfg AdvisoryLockConfig) error {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return err
	}
	if key == "" {
		return errors.New("cache: key and callback are required")
	}
	ttl, err := leaseConfig(cfg.Lock, cfg.Wait, cfg.RefreshRatio)
	if err != nil {
		return err
	}
	lk := leaseKey(key)
	token, err := l.acquire(ctx, lk, ttl, cfg.Wait)
	if err != nil {
		return err
	}
	return l.run(ctx, lk, token, ttl, cfg.RefreshRatio, func(ctx context.Context) error { return cfg.Do(ctx, key, token) }, func(ctx context.Context) error { return l.renew(ctx, lk, token, ttl) })
}

// LoadOrCreateT is the typed form of [Lock.LoadOrCreate]: values are stored
// as JSON.
func (l *Lock) LoadOrCreateT[T any](ctx context.Context, key string, cfg LoadOrCreateConfig[T]) (curr T, loaded bool, err error) {
	if cfg.Create == nil {
		return curr, false, errors.New("cache: create is required")
	}
	b, loaded, err := l.LoadOrCreate(ctx, key, LoadOrCreateConfig[[]byte]{
		Create: func(ctx context.Context, key string) ([]byte, time.Duration, error) {
			v, ttl, err := cfg.Create(ctx, key)
			if err != nil {
				return nil, 0, err
			}
			b, err := json.Marshal(v)
			return b, ttl, err
		},
		Lock: cfg.Lock, Wait: cfg.Wait, RefreshRatio: cfg.RefreshRatio, UseStream: cfg.UseStream,
	})
	if err != nil {
		return curr, false, err
	}
	err = json.Unmarshal(b, &curr)
	return curr, loaded, err
}

// publishValue stores the value (KEYS[2]) only if this process still owns the
// lease (KEYS[1] == token), so a process that lost its lease cannot overwrite
// a newer fill.
var publishValue = redis.NewScript(`if redis.call('GET', KEYS[1]) ~= ARGV[1] then return 0 end
if tonumber(ARGV[3]) > 0 then redis.call('SET', KEYS[2], ARGV[2], 'PX', ARGV[3]) else redis.call('SET', KEYS[2], ARGV[2]) end
return 1`)

// LoadOrCreate returns the value stored at key (loaded=true). On a miss it
// takes the lease for key, checks again, calls cfg.Create and stores the
// result, so Create runs in one process at a time across the whole fleet.
//
// Callers that find the lease taken wait up to cfg.Wait, polling every 10ms;
// they then return the value the winner stored (loaded=true), or ErrLocked if
// Wait is zero or elapsed. The value is only stored if this process still
// owns the lease; otherwise LoadOrCreate returns an error matching ErrLocked.
//
// The value key and lease key are separate Redis keys, so this does not work
// on Redis Cluster.
func (l *Lock) LoadOrCreate(ctx context.Context, key string, cfg LoadOrCreateConfig[[]byte]) (curr []byte, loaded bool, err error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, false, err
	}
	if key == "" {
		return nil, false, errors.New("cache: key and create are required")
	}
	ttl, err := leaseConfig(cfg.Lock, cfg.Wait, cfg.RefreshRatio)
	if err != nil {
		return nil, false, err
	}
	curr, err = l.cache.Load(ctx, key)
	if err == nil {
		return curr, true, nil
	}
	if !errors.Is(err, ErrNotExist) {
		return nil, false, err
	}
	lk := leaseKey(key)
	token, err := l.acquire(ctx, lk, ttl, cfg.Wait)
	if err != nil {
		return nil, false, err
	}
	var valueTTL time.Duration
	err = l.run(ctx, lk, token, ttl, cfg.RefreshRatio, func(work context.Context) error {
		curr, err = l.cache.Load(work, key)
		if err == nil {
			loaded = true
			return nil
		}
		if !errors.Is(err, ErrNotExist) {
			return err
		}
		curr, valueTTL, err = cfg.Create(work, key)
		if err != nil {
			return err
		}
		if valueTTL < 0 || (valueTTL > 0 && valueTTL < time.Millisecond) {
			return errors.New("cache: value TTL must be zero or at least one millisecond")
		}
		return nil
	}, func(work context.Context) error {
		if loaded {
			return nil
		}
		n, e := publishValue.Run(work, l.client, []string{lk, key}, token, curr, valueTTL.Milliseconds()).Int64()
		if e != nil {
			return e
		}
		if n != 1 {
			return fmt.Errorf("%w: lease lost before the value was stored", ErrLocked)
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return curr, loaded, nil
}

// WithDefaults returns a copy of c with Lock defaulted to 10s.
func (c AdvisoryLockConfig) WithDefaults() AdvisoryLockConfig {
	if c.Lock == 0 {
		c.Lock = 10 * time.Second
	}
	return c
}

// Validate reports whether c is usable.
func (c AdvisoryLockConfig) Validate() error {
	if c.Do == nil {
		return errors.New("cache: callback is required")
	}
	_, err := leaseConfig(c.Lock, c.Wait, c.RefreshRatio)
	return err
}

// WithDefaults returns a copy of c with Lock defaulted to 10s.
func (c LoadOrCreateConfig[T]) WithDefaults() LoadOrCreateConfig[T] {
	if c.Lock == 0 {
		c.Lock = 10 * time.Second
	}
	return c
}

// Validate reports whether c is usable.
func (c LoadOrCreateConfig[T]) Validate() error {
	if c.Create == nil {
		return errors.New("cache: create is required")
	}
	_, err := leaseConfig(c.Lock, c.Wait, c.RefreshRatio)
	return err
}
