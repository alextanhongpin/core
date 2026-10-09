package idempotent

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

type failingRenewal struct{ cleanup atomic.Bool }

func (*failingRenewal) LoadOrStore(context.Context, string, string, time.Duration) (string, bool, error) {
	return "", false, nil
}
func (*failingRenewal) CompareAndSwap(context.Context, string, string, string, time.Duration) error {
	return ErrLockConflict
}
func (c *failingRenewal) CompareAndDelete(context.Context, string, string) error {
	c.cleanup.Store(true)
	return nil
}
func TestRenewalLossCancelsAndJoinsTask(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := &failingRenewal{}
		i := New(client)
		finished := false
		_, err := i.runInLock(context.Background(), "k", "token", HandlerFunc[int, int](func(ctx context.Context, _ int) (int, error) {
			<-ctx.Done()
			if client.cleanup.Load() {
				t.Error("cleanup before task finished")
			}
			finished = true
			return 42, nil
		}), 1, time.Second, time.Hour)
		if !finished || !client.cleanup.Load() || !errors.Is(err, ErrLockConflict) {
			t.Fatalf("finished=%v cleanup=%v err=%v", finished, client.cleanup.Load(), err)
		}
	})
}
