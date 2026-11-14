package ratelimit_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	redis "github.com/redis/go-redis/v9"
)

func TestFixedWindowInvalidZeroAndRejectedBatch(t *testing.T) {
	rl := ratelimit.MustNewFixedWindow(newClient(t), ratelimit.Config{Limit: 5, Period: 1500 * time.Millisecond})
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
	rl := ratelimit.MustNewGCRA(newClient(t), ratelimit.Config{Limit: 1, Period: time.Hour, Burst: 2})
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

func TestFixedWindowLargeLimitKeepsExactCount(t *testing.T) {
	// Counts are Lua doubles; they must stay exact up to the largest allowed limit.
	const limit = 1_000_000_000_000_000
	rl := ratelimit.MustNewFixedWindow(newClient(t), ratelimit.Config{Limit: limit, Period: time.Hour})
	if ok, err := rl.AllowN(t.Context(), t.Name(), limit-1); err != nil || !ok {
		t.Fatal(ok, err)
	}
	res, err := rl.Limit(t.Context(), t.Name())
	if err != nil || !res.Allow || res.Remaining != 0 {
		t.Fatalf("last unit: %+v %v", res, err)
	}
	if ok, err := rl.Allow(t.Context(), t.Name()); err != nil || ok {
		t.Fatal("over limit admitted", err)
	}
}

func TestGCRAOversizedBatchReportsRealState(t *testing.T) {
	rl := ratelimit.MustNewGCRA(newClient(t), ratelimit.Config{Limit: 1, Period: time.Hour, Burst: 2})
	res, err := rl.LimitN(t.Context(), t.Name(), 4)
	if err != nil || res.Allow || res.Remaining != 3 || res.RetryAfter != 0 {
		t.Fatalf("oversized: %+v %v", res, err)
	}
	if ok, err := rl.Allow(t.Context(), t.Name()); err != nil || !ok {
		t.Fatal(ok, err)
	}
	res, err = rl.LimitN(t.Context(), t.Name(), 4)
	if err != nil || res.Allow || res.Remaining != 2 || res.ResetAfter <= 0 {
		t.Fatalf("oversized after use: %+v %v", res, err)
	}
}

func TestMissingSetupErrorHint(t *testing.T) {
	client := newClient(t)
	if err := client.FunctionDelete(t.Context(), ratelimit.LibraryName).Err(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ratelimit.Setup(context.Background(), client) })
	_, err := ratelimit.MustNewGCRA(client, ratelimit.Config{}).Allow(t.Context(), t.Name())
	if err == nil || !strings.Contains(err.Error(), "ratelimit.Setup") {
		t.Fatalf("want setup hint, got %v", err)
	}
	if !errors.Is(err, ratelimit.ErrNotSetup) {
		t.Fatalf("want errors.Is(err, ErrNotSetup), got %v", err)
	}
}

func TestSetupRingAndNilClient(t *testing.T) {
	ring := redis.NewRing(&redis.RingOptions{Addrs: map[string]string{"a": redistest.Addr()}})
	defer ring.Close()
	if err := ratelimit.Setup(t.Context(), ring); err != nil {
		t.Fatal(err)
	}
	if ok, err := ratelimit.MustNewFixedWindow(ring, ratelimit.Config{}).Allow(t.Context(), t.Name()); err != nil || !ok {
		t.Fatal(ok, err)
	}
	var typedNil *redis.Client
	if err := ratelimit.Setup(t.Context(), typedNil); err == nil {
		t.Fatal("typed nil client accepted")
	}
	if _, err := ratelimit.NewGCRA(typedNil, ratelimit.Config{}); err == nil {
		t.Fatal("typed nil client accepted")
	}
	if _, err := ratelimit.MustNewFixedWindow(newClient(t), ratelimit.Config{}).Allow(t.Context(), ""); err != ratelimit.ErrEmptyKey {
		t.Fatal(err)
	}
}
