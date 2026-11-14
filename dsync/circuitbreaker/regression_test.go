package circuitbreaker_test

import (
	"context"
	"errors"
	"strings"
	"testing"

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
