package lock

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

type leaseClient struct {
	released atomic.Bool
	failure  error
}

func (*leaseClient) Lock(context.Context, string, string, time.Duration) error     { return nil }
func (c *leaseClient) Extend(context.Context, string, string, time.Duration) error { return c.failure }
func (c *leaseClient) Unlock(context.Context, string, string) error {
	c.released.Store(true)
	return nil
}

type onceRetry struct{}

func (onceRetry) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }
func TestLeaseLossJoinsCallbackBeforeUnlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failure := errors.New("lease lost")
		client := &leaseClient{failure: failure}
		l := MustNew(client, Config{LockTTL: time.Second, RefreshRatio: 0.5, Retry: onceRetry{}})
		finished := false
		err := l.Do(context.Background(), "key", func(ctx context.Context) error {
			<-ctx.Done()
			if client.released.Load() {
				t.Error("lease released while callback active")
			}
			time.Sleep(time.Second)
			finished = true
			return nil
		})
		if !finished || !client.released.Load() || !errors.Is(err, failure) {
			t.Fatalf("finished=%v released=%v err=%v", finished, client.released.Load(), err)
		}
	})
}
func TestCallbackPanicPropagatesAndUnlocks(t *testing.T) {
	client := &leaseClient{}
	l := MustNew(client, Config{LockTTL: time.Second, Retry: onceRetry{}})
	defer func() {
		if recover() != "callback panic" || !client.released.Load() {
			t.Fatal("panic swallowed or lease retained")
		}
	}()
	l.Do(context.Background(), "key", func(context.Context) error { panic("callback panic") })
}
func TestCanceledLocalAdmissionReturnsPromptly(t *testing.T) {
	l := MustNew(&leaseClient{}, Config{LockTTL: time.Hour, Retry: onceRetry{}})
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		l.Do(context.Background(), "key", func(context.Context) error { close(started); <-release; return nil })
	}()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := l.Do(ctx, "key", func(context.Context) error { t.Error("canceled callback ran"); return nil })
	close(release)
	<-done
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestValueConfigOwnership(t *testing.T) {
	cfg := Config{LockTTL: time.Second, RefreshRatio: 0.5, Retry: onceRetry{}}
	l, err := New(&leaseClient{}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.LockTTL = -time.Second
	if l.cfg.LockTTL != time.Second {
		t.Fatal("caller changed effective configuration")
	}
	if _, err := New(nil, Config{}); err == nil {
		t.Fatal("nil client accepted")
	}
	if _, err := New(&leaseClient{}, Config{LockTTL: -time.Second}); err == nil {
		t.Fatal("invalid TTL accepted")
	}
}

// ambiguousClient creates the key but reports a failure, as happens when the
// caller's context expires after Redis has executed SET.
type ambiguousClient struct {
	leaseClient
	unlockedToken atomic.Value
}

func (c *ambiguousClient) Lock(context.Context, string, string, time.Duration) error {
	return context.DeadlineExceeded
}

func (c *ambiguousClient) Unlock(_ context.Context, _, token string) error {
	c.unlockedToken.Store(token)
	return nil
}

func TestAmbiguousAcquireFailureReleasesToken(t *testing.T) {
	client := &ambiguousClient{}
	l := MustNew(client, Config{LockTTL: time.Second, Retry: onceRetry{}})
	err := l.Do(context.Background(), "key", func(context.Context) error {
		t.Error("callback ran without a lease")
		return nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if tok, _ := client.unlockedToken.Load().(string); tok == "" {
		t.Fatal("possibly acquired lease was not released")
	}
}

// busyClient always reports contention; Unlock must not be called for it.
type busyClient struct{ leaseClient }

func (*busyClient) Lock(context.Context, string, string, time.Duration) error { return ErrLocked }

func TestBusyKeyDoesNotUnlock(t *testing.T) {
	client := &busyClient{}
	l := MustNew(client, Config{LockTTL: time.Second, Retry: onceRetry{}})
	err := l.Do(context.Background(), "key", func(context.Context) error { return nil })
	if !errors.Is(err, ErrLocked) || client.released.Load() {
		t.Fatalf("err=%v released=%v", err, client.released.Load())
	}
}

// hangingClient's Extend blocks until its context ends, like a stalled Redis.
type hangingClient struct{ leaseClient }

func (*hangingClient) Extend(ctx context.Context, _, _ string, _ time.Duration) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestStalledRenewalCancelsBeforeLeaseExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := &hangingClient{}
		l := MustNew(client, Config{LockTTL: time.Second, RefreshRatio: 0.5, Retry: onceRetry{}})
		start := time.Now()
		var canceledAfter time.Duration
		err := l.Do(context.Background(), "key", func(ctx context.Context) error {
			<-ctx.Done()
			canceledAfter = time.Since(start)
			return nil
		})
		if !errors.Is(err, ErrExpired) {
			t.Fatalf("want ErrExpired, got %v", err)
		}
		if canceledAfter > time.Second {
			t.Fatalf("callback kept running %v, past the 1s lease", canceledAfter)
		}
	})
}

func TestNoRetryHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := NoRetry().Do(ctx, func(context.Context) error { t.Error("called"); return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
