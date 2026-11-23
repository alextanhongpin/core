package singleflight_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/singleflight"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func leaseKeyFor(key string) string {
	return fmt.Sprintf("singleflight:lease:%x", sha256.Sum256([]byte(key)))
}

// failAfterSet lets the first SET reach Redis, then reports a timeout, as when
// the reply is lost after the command executed.
type failAfterSet struct{ fired atomic.Bool }

func (*failAfterSet) DialHook(next redis.DialHook) redis.DialHook { return next }
func (*failAfterSet) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *failAfterSet) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if cmd.Name() == "set" && h.fired.CompareAndSwap(false, true) {
			return context.DeadlineExceeded
		}
		return err
	}
}

var _ redis.Hook = (*failAfterSet)(nil)

func TestAmbiguousAcquireDoesNotLeaveLease(t *testing.T) {
	client := redistest.Client(t)
	hooked := redistest.Client(t)
	hooked.AddHook(&failAfterSet{})

	g := singleflight.MustNew(hooked, singleflight.Config{LockTTL: time.Minute})
	_, err := g.Do(t.Context(), t.Name(), func(context.Context) error {
		t.Error("callback ran without a confirmed lease")
		return nil
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.EqualValues(t, 0, client.Exists(t.Context(), leaseKeyFor(t.Name())).Val(),
		"orphaned lease would block other processes for LockTTL")
}

func TestRemoteWaitTimeoutReturnsErrTimeout(t *testing.T) {
	client := redistest.Client(t)
	// Simulate a leader in another process holding the lease.
	require.NoError(t, client.Set(t.Context(), leaseKeyFor(t.Name()), "other", time.Minute).Err())
	t.Cleanup(func() { client.Del(context.Background(), leaseKeyFor(t.Name())) })

	for _, wait := range []time.Duration{time.Nanosecond, 50 * time.Millisecond} {
		g := singleflight.MustNew(client, singleflight.Config{WaitTTL: wait})
		did, err := g.Do(t.Context(), t.Name(), func(context.Context) error {
			t.Error("follower ran")
			return nil
		})
		require.False(t, did)
		require.ErrorIs(t, err, singleflight.ErrTimeout, "WaitTTL=%v", wait)
	}
}

func TestLoadOrStoreRejectsSubMillisecondTTL(t *testing.T) {
	c := singleflight.MustNewCache[string](redistest.Client(t), singleflight.CacheConfig{})
	getter := func(context.Context) (string, error) { return "v", nil }
	for _, ttl := range []time.Duration{-time.Second, 500 * time.Microsecond} {
		_, loaded, err := c.LoadOrStore(t.Context(), t.Name(), getter, ttl)
		require.False(t, loaded)
		require.ErrorContains(t, err, "ttl must be 0 (no expiry) or at least 1ms")
	}
	_, _, err := c.LoadOrStore(t.Context(), "", getter, time.Second)
	require.ErrorContains(t, err, "key and getter are required")
}

func TestDoDidIsFalseForFailingLeaderAndLocalWaiters(t *testing.T) {
	g := singleflight.MustNew(redistest.Client(t), singleflight.Config{})
	boom := errors.New("boom")
	started := make(chan struct{})
	release := make(chan struct{})
	type result struct {
		did bool
		err error
	}
	leader := make(chan result, 1)
	go func() {
		did, err := g.Do(t.Context(), t.Name(), func(context.Context) error { close(started); <-release; return boom })
		leader <- result{did, err}
	}()
	<-started
	waiter := make(chan result, 1)
	go func() {
		did, err := g.Do(t.Context(), t.Name(), func(context.Context) error { t.Error("waiter ran"); return nil })
		waiter <- result{did, err}
	}()
	time.Sleep(20 * time.Millisecond)
	close(release)
	l, w := <-leader, <-waiter
	require.False(t, l.did)
	require.ErrorIs(t, l.err, boom)
	require.False(t, w.did)
	require.ErrorIs(t, w.err, boom)
}

// A caller in another process whose leader's getter failed must try the fill
// itself rather than return redis.Nil.
func TestCacheRemoteWaiterRetriesAfterLeaderFails(t *testing.T) {
	// Two Caches have separate Groups, like two processes.
	a := singleflight.MustNewCache[string](redistest.Client(t), singleflight.CacheConfig{})
	b := singleflight.MustNewCache[string](redistest.Client(t), singleflight.CacheConfig{})
	boom := errors.New("boom")
	started, release := make(chan struct{}), make(chan struct{})
	leaderErr := make(chan error, 1)
	go func() {
		_, _, err := a.LoadOrStore(t.Context(), t.Name(), func(context.Context) (string, error) {
			close(started)
			<-release
			return "", boom
		}, time.Minute)
		leaderErr <- err
	}()
	<-started
	type result struct {
		v      string
		loaded bool
		err    error
	}
	waiter := make(chan result, 1)
	go func() {
		v, loaded, err := b.LoadOrStore(t.Context(), t.Name(), func(context.Context) (string, error) {
			return "fresh", nil
		}, time.Minute)
		waiter <- result{v, loaded, err}
	}()
	time.Sleep(30 * time.Millisecond) // let b start waiting on a's lease
	close(release)
	require.ErrorIs(t, <-leaderErr, boom)
	w := <-waiter
	require.NoError(t, w.err)
	require.Equal(t, "fresh", w.v)
	require.False(t, w.loaded)
}
