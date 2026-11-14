package lock

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"time"
	"uuid"

	"github.com/alextanhongpin/core/sync/cache"
	"github.com/alextanhongpin/core/sync/retry"
)

var (
	// ErrExpired means the lease was no longer held when Locker tried to renew
	// or release it: it expired, or it could not be renewed before it expired.
	// Work done under the lease may have overlapped with another owner.
	ErrExpired = errors.New("lock: lock expired")

	// ErrLockTimeout cancels the callback context when renewal is disabled
	// (RefreshRatio is zero) and the callback runs longer than LockTTL.
	ErrLockTimeout = errors.New("lock: exceeded lock duration")

	// ErrLocked means the key is held by another token. Acquisition returns it
	// when the key is busy; renewal returns it when the key was deleted or taken
	// over by someone else.
	ErrLocked = errors.New("lock: another process has acquired the lock")
)

// Retry runs the acquisition attempt, retrying it as it sees fit.
// *retry.Retry from github.com/alextanhongpin/core/sync/retry implements it.
type Retry interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

// Config configures a Locker. Use DefaultConfig for the recommended settings,
// or a Config literal whose zero fields are filled by WithDefaults.
type Config struct {
	// LockTTL is how long the Redis key lives before it expires on its own.
	// Zero selects 30 seconds; it must be at least one millisecond.
	LockTTL time.Duration

	// RefreshRatio controls renewal. When it is in (0, 1), the lease is
	// extended every LockTTL*RefreshRatio while the callback runs. Zero
	// disables renewal: the callback context is then canceled with
	// ErrLockTimeout once LockTTL has elapsed.
	RefreshRatio float64

	// Retry controls waiting for a busy key. Nil selects DefaultRetry.
	Retry Retry

	// Deprecated: Logger is ignored. Locker reports every failure through the
	// error returned by Do.
	Logger *slog.Logger
}

// Validate reports whether c is usable. It does not apply defaults.
func (c Config) Validate() error {
	if c.LockTTL < time.Millisecond {
		return errors.New("lock: lock duration must be at least one millisecond")
	}
	if math.IsNaN(c.RefreshRatio) || math.IsInf(c.RefreshRatio, 0) || c.RefreshRatio < 0 || c.RefreshRatio >= 1 {
		return errors.New("lock: refresh ratio must be in the range [0, 1)")
	}

	if c.RefreshRatio > 0 && time.Duration(float64(c.LockTTL)*c.RefreshRatio) <= 0 {
		return errors.New("lock: refresh interval must be positive")
	}
	return nil
}

// DefaultRetry waits for a busy key: it retries ErrLocked up to 10 times with
// full-jitter exponential backoff (retry n sleeps a random duration in
// [0, min(50ms*2^n, 2s))), so it waits at most about 13.1s in total and about
// 6.5s on average. It then gives up with an error matching both ErrLocked and
// retry.ErrLimitExceeded. Other errors are returned immediately.
func DefaultRetry() Retry {
	cfg := retry.DefaultConfig()
	cfg.MaxRetries = 10
	cfg.Backoff = retry.NewExponentialBackoff(50*time.Millisecond, 2*time.Second)
	cfg.Throttler = retry.NewNoopThrottler()
	cfg.Retryable = func(err error) (error, bool) { return err, errors.Is(err, ErrLocked) }
	r, _ := retry.New(cfg)
	return r
}

// NoRetry makes a single acquisition attempt. Use it in Config.Retry to fail
// fast with ErrLocked instead of waiting when the key is busy.
func NoRetry() Retry { return noRetry{} }

type noRetry struct{}

func (noRetry) Do(ctx context.Context, fn func(context.Context) error) error {
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	return fn(ctx)
}

// DefaultConfig returns the recommended configuration: a 30s lease renewed
// every 24s (RefreshRatio 0.8) and DefaultRetry.
func DefaultConfig() Config {
	return Config{
		LockTTL:      30 * time.Second,
		RefreshRatio: 0.8,
		Retry:        DefaultRetry(),
	}
}

// WithDefaults returns c with a zero LockTTL set to 30s and a nil Retry set to
// DefaultRetry. It leaves RefreshRatio alone, so Config{} has renewal disabled;
// use DefaultConfig to get renewal.
func (c Config) WithDefaults() Config {
	if c.LockTTL == 0 {
		c.LockTTL = 30 * time.Second
	}
	if c.Retry == nil {
		c.Retry = DefaultRetry()
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	return c
}

type permit struct{ ch chan struct{} }

// Locker runs callbacks while holding a Redis lease on a key. It is safe for
// concurrent use; callers in the same process that use the same key queue
// locally instead of all polling Redis. Construct it with New.
//
// Locker also exposes its Backend's Lock, Extend and Unlock methods for callers
// that manage leases by hand.
type Locker struct {
	cfg     Config
	permits *cache.Cache[string, permit]
	client
}

// New returns a Locker that stores leases through c (usually NewClient).
// cfg is copied, defaulted with WithDefaults and validated.
func New(c Backend, cfg Config) (*Locker, error) {
	if c == nil {
		return nil, errors.New("lock: nil client")
	}
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Locker{cfg: cfg, client: c, permits: cache.New(func(string) (*permit, error) { return &permit{ch: make(chan struct{}, 1)}, nil })}, nil
}

// MustNew is like New but panics on error. Use it during program startup.
func MustNew(c Backend, cfg Config) *Locker {
	l, err := New(c, cfg)
	if err != nil {
		panic(err)
	}
	return l
}

// Do acquires the lease for key, runs fn, and releases the lease.
//
// While fn runs, the lease is renewed in the background (if RefreshRatio > 0).
// The context passed to fn is canceled when ctx is canceled, when renewal fails,
// or, with renewal disabled, when LockTTL elapses; context.Cause tells which.
// Do always waits for fn to return before releasing the lease, so fn must stop
// promptly when its context is canceled. fn must not call Do with the same key:
// that deadlocks.
//
// The returned error joins fn's error, any lease-loss cause and any release
// error; test it with errors.Is. If the key stays busy, Do returns an error
// matching ErrLocked. A panic in fn propagates after the lease is released.
func (l *Locker) Do(ctx context.Context, key string, fn func(context.Context) error) (err error) {
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	if key == "" || fn == nil {
		return errors.New("lock: key and callback are required")
	}
	p, _, _ := l.permits.LoadOrCreate(key)
	select {
	case p.ch <- struct{}{}:
	case <-ctx.Done():
		return context.Cause(ctx)
	}
	defer func() { <-p.ch }()
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	tok := uuid.NewV7().String()
	var (
		acquiredAt time.Time
		// uncertain is set when an attempt may have created the key even though
		// it did not report success (for example, a timeout after SET was sent).
		uncertain bool
	)
	err = l.cfg.Retry.Do(ctx, func(ctx context.Context) error {
		start := time.Now()
		err := l.Lock(ctx, key, tok, l.cfg.LockTTL)
		switch {
		case err == nil:
			acquiredAt = start
			uncertain = true
		case !errors.Is(err, ErrLocked):
			uncertain = true
		}
		return err
	})
	if err != nil {
		if uncertain {
			// Release a key this token may own so it does not block others
			// until it expires. Unlock is token-checked, so this is safe.
			cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			_ = l.Unlock(cleanupCtx, key, tok)
			cleanupCancel()
		}
		return err
	}
	// Deadlines are measured from before the SET was sent, so the local view of
	// the lease never outlives the key in Redis.
	expiresAt := acquiredAt.Add(l.cfg.LockTTL)

	workCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	refresh := time.Duration(float64(l.cfg.LockTTL) * l.cfg.RefreshRatio)
	if refresh <= 0 {
		timeoutCtx, timeoutCancel := context.WithDeadlineCause(workCtx, expiresAt, ErrLockTimeout)
		defer timeoutCancel()
		workCtx = timeoutCtx
	}
	stop, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		if refresh <= 0 {
			return
		}
		ticker := time.NewTicker(refresh)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-workCtx.Done():
				return
			case <-ticker.C:
				// A renewal that cannot finish before the current lease expires is
				// useless: by then another process may own the key.
				extendCtx, extendCancel := context.WithDeadlineCause(workCtx, expiresAt, ErrExpired)
				start := time.Now()
				err := l.Extend(extendCtx, key, tok, l.cfg.LockTTL)
				if err != nil && errors.Is(context.Cause(extendCtx), ErrExpired) {
					err = errors.Join(ErrExpired, err)
				}
				extendCancel()
				if err != nil {
					cancel(err)
					return
				}
				expiresAt = start.Add(l.cfg.LockTTL)
			}
		}
	}()
	defer func() {
		close(stop)
		<-stopped
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cleanupCancel()
		err = errors.Join(err, l.Unlock(cleanupCtx, key, tok))
	}()
	err = fn(workCtx)
	if workCtx.Err() != nil {
		err = errors.Join(err, context.Cause(workCtx))
	}
	return err
}
