package ratelimit_test

import (
	"github.com/alextanhongpin/core/dsync/ratelimit"
	"testing"
	"time"
)

func TestFixedWindowInvalidZeroAndRejectedBatch(t *testing.T) {
	rl := ratelimit.NewFixedWindow(newClient(t), 5, 1500*time.Millisecond)
	if _, err := rl.LimitN(t.Context(), t.Name(), -1); err != ratelimit.ErrNegative {
		t.Fatal(err)
	}
	inspect, err := rl.LimitN(t.Context(), t.Name(), 0)
	if err != nil || !inspect.Allow || inspect.Remaining != 5 {
		t.Fatalf("inspection: %+v %v", inspect, err)
	}
	if ok, err := rl.AllowN(t.Context(), t.Name(), 4); err != nil || !ok {
		t.Fatal(ok, err)
	}
	rejected, err := rl.LimitN(t.Context(), t.Name(), 2)
	if err != nil || rejected.Allow || rejected.Remaining != 1 || rejected.RetryAfter <= 0 {
		t.Fatalf("batch: %+v %v", rejected, err)
	}
	if ok, err := rl.Allow(t.Context(), t.Name()); err != nil || !ok {
		t.Fatal("rejection consumed capacity", err)
	}
}
func TestGCRABatchesAreAtomic(t *testing.T) {
	rl := ratelimit.NewGCRA(newClient(t), 1, time.Hour, 2)
	if ok, err := rl.AllowN(t.Context(), t.Name(), 4); err != nil || ok {
		t.Fatal("oversize admitted", err)
	}
	if ok, err := rl.AllowN(t.Context(), t.Name(), 2); err != nil || !ok {
		t.Fatal("initial batch rejected", err)
	}
	if ok, err := rl.AllowN(t.Context(), t.Name(), 2); err != nil || ok {
		t.Fatal("partial batch admitted", err)
	}
	if ok, err := rl.Allow(t.Context(), t.Name()); err != nil || !ok {
		t.Fatal("denied batch consumed tokens", err)
	}
}
