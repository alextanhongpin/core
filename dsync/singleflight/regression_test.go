package singleflight_test

import (
	"context"
	"crypto/sha256"
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
