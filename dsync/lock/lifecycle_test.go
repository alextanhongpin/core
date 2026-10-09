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
		l := New(client, &Config{LockTTL: time.Second, RefreshRatio: 0.5, Retry: onceRetry{}})
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
	l := New(client, &Config{LockTTL: time.Second, Retry: onceRetry{}})
	defer func() {
		if recover() != "callback panic" || !client.released.Load() {
			t.Fatal("panic swallowed or lease retained")
		}
	}()
	l.Do(context.Background(), "key", func(context.Context) error { panic("callback panic") })
}
