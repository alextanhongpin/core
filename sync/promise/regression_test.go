package promise_test

import (
	"context"
	"errors"
	"github.com/alextanhongpin/core/sync/promise"
	"testing"
	"time"
)

func TestEarlyCombinators(t *testing.T) {
	for _, name := range []string{"race", "any"} {
		t.Run(name, func(t *testing.T) {
			pending := promise.Deferred[int](t.Context())
			defer pending.Abort(context.Canceled)
			ready := promise.Resolve(t.Context(), 42)
			done := make(chan struct{})
			go func() {
				var v int
				var err error
				if name == "race" {
					v, err = promise.Race(pending, ready)
				} else {
					v, err = promise.Any(pending, ready)
				}
				if v != 42 || err != nil {
					t.Errorf("got %d, %v", v, err)
				}
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("waited for unresolved loser")
			}
		})
	}
}

func TestAbortStatusAndLateResolve(t *testing.T) {
	p := promise.Deferred[int](t.Context())
	p.Abort(errors.ErrUnsupported)
	p.Resolve(42)
	if p.Status() != promise.StatusRejected {
		t.Fatal("aborted promise must be rejected")
	}
	if _, err := p.Await(); !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("got %v", err)
	}
}

func TestAbortCancelsHandler(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})
	p := promise.New(t.Context(), func(ctx context.Context) (int, error) {
		close(started)
		<-ctx.Done()
		close(finished)
		return 0, context.Cause(ctx)
	})
	<-started
	p.Abort(errors.ErrUnsupported)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("handler context not canceled")
	}
}
