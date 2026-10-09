package promise

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
)

func TestObserveStopsUnresolvedWaiters(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pending := Deferred[int](context.Background())
		_, stop := observe([]*Promise[int]{pending})
		synctest.Wait()
		stop()
		if pending.Status() != StatusPending {
			t.Fatal("combinator aborted input")
		}
	})
}
func TestAllFailsWithoutWaitingForPendingInput(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pending := Deferred[int](context.Background())
		_, err := All(pending, Reject[int](context.Background(), errors.ErrUnsupported))
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Fatal(err)
		}
		if pending.Status() != StatusPending {
			t.Fatal("pending input aborted")
		}
	})
}
func TestAllPreservesInputOrder(t *testing.T) {
	ctx := context.Background()
	got, err := All(Resolve(ctx, 3), Resolve(ctx, 1), Resolve(ctx, 2))
	if err != nil || len(got) != 3 || got[0] != 3 || got[1] != 1 || got[2] != 2 {
		t.Fatalf("%v %v", got, err)
	}
}
