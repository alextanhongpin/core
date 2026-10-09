package goroutine_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/goroutine"
)

func TestGoroutine(t *testing.T) {
	var started atomic.Int64
	var stopped atomic.Int64
	var done atomic.Int64

	ctx := t.Context()
	fn := func(ctx context.Context) {
		started.Add(1)

		select {
		case <-ctx.Done():
			stopped.Add(1)

		case <-time.After(10 * time.Millisecond):
			done.Add(1)
		}
	}
	n := 10
	g := goroutine.New()
	for range n {
		g.Start(ctx, fn)
		g.Stop()
	}
	g.Start(ctx, fn)
	time.Sleep(15 * time.Millisecond)
	g.Stop()

	if want, got := int64(n+1), started.Load(); want != got {
		t.Errorf("want %d, got %d", want, got)
	}

	if want, got := int64(n), stopped.Load(); want != got {
		t.Errorf("want %d, got %d", want, got)
	}

	if want, got := int64(1), done.Load(); want != got {
		t.Errorf("want %d, got %d", want, got)
	}
}

func TestCompletedCallbackReleasesContext(t *testing.T) {
	g := goroutine.New()
	contexts := make(chan context.Context, 1)
	g.Start(t.Context(), func(ctx context.Context) { contexts <- ctx })
	ctx := <-contexts
	defer g.Stop()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("completed callback retained its context")
	}
	if ctx.Err() != context.Canceled {
		t.Fatal("callback context was not canceled")
	}
}
