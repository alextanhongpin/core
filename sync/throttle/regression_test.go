package throttle

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

func TestBacklogTimeoutDoesNotCancelExecution(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		th := MustNew(Config{Limit: 1, BacklogTimeout: time.Second})
		err := th.Do(context.Background(), func(ctx context.Context) error {
			time.Sleep(2 * time.Second)
			return ctx.Err()
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}
func TestZeroTimeoutAndCanceledAdmission(t *testing.T) {
	th := MustNew(Config{Limit: 1})
	if err := th.Do(context.Background(), func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for range 100 {
		if err := th.Do(ctx, func(context.Context) error { t.Fatal("canceled callback ran"); return nil }); err != context.Canceled {
			t.Fatal(err)
		}
	}
}
func TestQueuedTimeoutReturnsTokens(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		th := MustNew(Config{Limit: 1, BacklogLimit: 1, BacklogTimeout: time.Second})
		started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
		go func() {
			defer close(done)
			th.Do(context.Background(), func(context.Context) error { close(started); <-release; return nil })
		}()
		<-started
		if err := th.Do(context.Background(), func(context.Context) error { t.Fatal("queued callback ran"); return nil }); err != ErrTimeout {
			t.Fatal(err)
		}
		close(release)
		<-done
		if err := th.Do(context.Background(), func(context.Context) error { return nil }); err != nil {
			t.Fatal(err)
		}
	})
}
