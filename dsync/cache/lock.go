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

// Lock coordinates cache fills using leases on a single Redis primary. The
// client is borrowed. Callbacks must observe cancellation when a lease is lost.
type Lock struct {
	client *redis.Client
	cache  *Redis
}

func NewLock(client *redis.Client) *Lock { return &Lock{client: client, cache: NewRedis(client)} }

type AdvisoryLockConfig struct {
	Do           func(context.Context, string, []byte) error
	Wait         time.Duration // Maximum total admission wait; zero fails immediately.
	Lock         time.Duration // Zero selects ten seconds.
	RefreshRatio float32       // Zero disables renewal; otherwise strictly between zero and one.
	UseStream    bool          // Deprecated: admission uses bounded polling.
}

type LoadOrCreateConfig[T any] struct {
	Create       func(context.Context, string) (T, time.Duration, error)
	Lock         time.Duration
	RefreshRatio float32
	UseStream    bool // Deprecated: admission uses bounded polling.
	Wait         time.Duration
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
		ctx, cancel = context.WithTimeout(ctx, wait)
		defer cancel()
	}
	for {
		if err := context.Cause(ctx); err != nil {
			return nil, err
		}
		ok, err := l.client.SetNX(ctx, key, token, ttl).Result()
		if err != nil {
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
					cancel(ErrLocked)
					return
				}
				if e := l.cache.CompareAndSwap(work, key, token, token, ttl); e != nil {
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

func (l *Lock) AdvisoryLock(ctx context.Context, key string, cfg *AdvisoryLockConfig) error {
	if cfg == nil || cfg.Do == nil || key == "" {
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
	return l.run(ctx, lk, token, ttl, cfg.RefreshRatio, func(ctx context.Context) error { return cfg.Do(ctx, key, token) }, func(ctx context.Context) error { return l.cache.CompareAndSwap(ctx, lk, token, token, ttl) })
}

func (l *Lock) LoadOrCreateT[T any](ctx context.Context, key string, cfg *LoadOrCreateConfig[T]) (curr T, loaded bool, err error) {
	if cfg == nil || cfg.Create == nil {
		return curr, false, errors.New("cache: create is required")
	}
	b, loaded, err := l.LoadOrCreate(ctx, key, &LoadOrCreateConfig[[]byte]{
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

const publishValue = `if redis.call('GET', KEYS[1]) ~= ARGV[1] then return 0 end
if tonumber(ARGV[3]) > 0 then redis.call('SET', KEYS[2], ARGV[2], 'PX', ARGV[3]) else redis.call('SET', KEYS[2], ARGV[2]) end
return 1`

func (l *Lock) LoadOrCreate(ctx context.Context, key string, cfg *LoadOrCreateConfig[[]byte]) (curr []byte, loaded bool, err error) {
	if cfg == nil || cfg.Create == nil || key == "" {
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
		n, e := l.client.Eval(work, publishValue, []string{lk, key}, token, curr, valueTTL.Milliseconds()).Int64()
		if e != nil {
			return e
		}
		if n != 1 {
			return ErrLocked
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return curr, loaded, nil
}
