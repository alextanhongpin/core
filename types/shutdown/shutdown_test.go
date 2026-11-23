package shutdown_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/alextanhongpin/core/types/shutdown"
)

func TestWait(t *testing.T) {
	var calls atomic.Int32
	errA := errors.New("a")
	errB := errors.New("b")

	var s shutdown.Shutdown
	s.Append(func(context.Context) error { calls.Add(1); return errA })
	s.Append(nil)
	s.Append(func(context.Context) error { calls.Add(1); return nil })
	s.Append(func(context.Context) error { calls.Add(1); return errB })

	err := s.Wait(context.Background())
	if got := calls.Load(); got != 3 {
		t.Fatalf("calls = %d, want 3", got)
	}
	if !errors.Is(err, errA) || !errors.Is(err, errB) {
		t.Fatalf("err = %v, want both a and b", err)
	}
	if err.Error() != "a\nb" {
		t.Fatalf("err = %q, want registration order", err.Error())
	}
}

func TestWaitDropsContextErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var s shutdown.Shutdown
	s.Append(func(ctx context.Context) error { return ctx.Err() })
	s.Append(func(ctx context.Context) error { return errors.Join(errors.New("wrapped"), ctx.Err()) })

	if err := s.Wait(ctx); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}

func TestWaitEmpty(t *testing.T) {
	var s shutdown.Shutdown
	if err := s.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}
