package circuitbreaker_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	redis "github.com/redis/go-redis/v9"
)

func TestOldFailureCannotChangeNewState(t *testing.T) {
	cb := circuitbreaker.MustNew(newClient(t), circuitbreaker.Config{})
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		cb.Do(t.Context(), t.Name(), func() error { close(started); <-release; return errors.New("old failure") })
	}()
	<-started
	if err := cb.SetStatus(t.Context(), t.Name(), circuitbreaker.HalfOpen); err != nil {
		t.Fatal(err)
	}
	close(release)
	<-done
	status, err := cb.Status(t.Context(), t.Name())
	if err != nil || status != circuitbreaker.HalfOpen {
		t.Fatalf("%v %v", status, err)
	}
}
func TestHalfOpenClosesAtExactThreshold(t *testing.T) {
	cfg := circuitbreaker.DefaultConfig()
	cfg.SuccessThreshold = 2
	cb := circuitbreaker.MustNew(newClient(t), cfg)
	if err := cb.SetStatus(t.Context(), t.Name(), circuitbreaker.HalfOpen); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := cb.Do(t.Context(), t.Name(), func() error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
	status, err := cb.Status(t.Context(), t.Name())
	if err != nil || status != circuitbreaker.Closed {
		t.Fatalf("%v %v", status, err)
	}
}
func TestSetOpenedRejectsImmediately(t *testing.T) {
	cb := circuitbreaker.MustNew(newClient(t), circuitbreaker.Config{})
	if err := cb.SetStatus(t.Context(), t.Name(), circuitbreaker.Opened); err != nil {
		t.Fatal(err)
	}
	if err := cb.Do(t.Context(), t.Name(), func() error { t.Error("opened callback ran"); return nil }); !errors.Is(err, circuitbreaker.ErrOpened) {
		t.Fatal(err)
	}
}

func TestSuccessfulProbeIsNotReportedAsFailure(t *testing.T) {
	client := newClient(t)
	cb := circuitbreaker.MustNew(client, circuitbreaker.Config{})
	if err := cb.SetStatus(t.Context(), t.Name(), circuitbreaker.HalfOpen); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = circuitbreaker.Setup(context.Background(), client) })
	err := cb.Do(t.Context(), t.Name(), func() error {
		// Make recording the result fail after the operation succeeded.
		return client.FunctionDelete(t.Context(), circuitbreaker.LibraryName).Err()
	})
	if err != nil {
		t.Fatalf("successful call reported as failed: %v", err)
	}
	err = cb.Do(t.Context(), t.Name(), func() error { return nil })
	if err == nil || !strings.Contains(err.Error(), "circuitbreaker.Setup") {
		t.Fatalf("want setup hint, got %v", err)
	}
	if !errors.Is(err, circuitbreaker.ErrNotSetup) {
		t.Fatalf("want ErrNotSetup, got %v", err)
	}
	if err := cb.SetStatus(t.Context(), t.Name(), circuitbreaker.Closed); !errors.Is(err, circuitbreaker.ErrNotSetup) {
		t.Fatalf("SetStatus: want ErrNotSetup, got %v", err)
	}
}

func TestSetupRingAndNilClient(t *testing.T) {
	ring := redis.NewRing(&redis.RingOptions{Addrs: map[string]string{"a": redistest.Addr()}})
	defer ring.Close()
	if err := circuitbreaker.Setup(t.Context(), ring); err != nil {
		t.Fatal(err)
	}
	if err := circuitbreaker.MustNew(ring, circuitbreaker.Config{}).Do(t.Context(), t.Name(), func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	var typedNil *redis.Client
	if err := circuitbreaker.Setup(t.Context(), typedNil); err == nil {
		t.Fatal("typed nil client accepted")
	}
	if _, err := circuitbreaker.New(typedNil, circuitbreaker.Config{}); err == nil {
		t.Fatal("typed nil client accepted")
	}
}

func TestStatusString(t *testing.T) {
	if got := circuitbreaker.HalfOpen.String(); got != "half-open" {
		t.Fatal(got)
	}
	if got := circuitbreaker.Status(42).String(); got != "Status(42)" {
		t.Fatal(got)
	}
}

// Healthy breakers must not leave a hash in Redis forever: a successful call
// on a new key stores nothing, and a breaker that closes again expires once
// its counter and generation fence lapse.
func TestClosedBreakerKeysExpire(t *testing.T) {
	client := newClient(t)
	cb := circuitbreaker.MustNew(client, circuitbreaker.Config{
		FailureThreshold: 1,
		FailurePeriod:    50 * time.Millisecond,
		SuccessThreshold: 1,
		OpenTimeout:      50 * time.Millisecond,
	})
	ctx, key := t.Context(), t.Name()
	exists := func() bool {
		t.Helper()
		n, err := client.Exists(ctx, key).Result()
		if err != nil {
			t.Fatal(err)
		}
		return n == 1
	}
	if err := cb.Do(ctx, key, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if exists() {
		t.Fatal("successful call on a new key created a hash")
	}

	boom := errors.New("boom")
	if err := cb.Do(ctx, key, func() error { return boom }); !errors.Is(err, boom) {
		t.Fatal(err)
	}
	if status, _ := cb.Status(ctx, key); status != circuitbreaker.Opened {
		t.Fatalf("want opened, got %v", status)
	}
	time.Sleep(60 * time.Millisecond)
	if err := cb.Do(ctx, key, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if status, _ := cb.Status(ctx, key); status != circuitbreaker.Closed {
		t.Fatalf("want closed, got %v", status)
	}
	deadline := time.Now().Add(2 * time.Second)
	for exists() {
		if time.Now().After(deadline) {
			t.Fatal("closed breaker hash never expired")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A HalfOpen probe that finishes after the breaker closed and its hash
// expired must not be counted against the next generation.
func TestStaleProbeIgnoredAfterExpiry(t *testing.T) {
	client := newClient(t)
	cb := circuitbreaker.MustNew(client, circuitbreaker.Config{
		FailureThreshold: 1,
		OpenTimeout:      20 * time.Millisecond,
	})
	ctx, key := t.Context(), t.Name()
	if err := cb.SetStatus(ctx, key, circuitbreaker.HalfOpen); err != nil {
		t.Fatal(err)
	}
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- cb.Do(ctx, key, func() error { close(started); <-release; return errors.New("stale") })
	}()
	<-started
	if err := cb.SetStatus(ctx, key, circuitbreaker.Closed); err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond) // status and generation expire
	close(release)
	<-done
	if status, _ := cb.Status(ctx, key); status != circuitbreaker.Closed {
		t.Fatalf("stale probe changed the state to %v", status)
	}
}
