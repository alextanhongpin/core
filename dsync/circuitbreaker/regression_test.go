package circuitbreaker_test

import (
	"errors"
	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"testing"
)

func TestOldFailureCannotChangeNewState(t *testing.T) {
	cb := circuitbreaker.New(newClient(t), nil)
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
	cb := circuitbreaker.New(newClient(t), cfg)
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
	cb := circuitbreaker.New(newClient(t), nil)
	if err := cb.SetStatus(t.Context(), t.Name(), circuitbreaker.Opened); err != nil {
		t.Fatal(err)
	}
	if err := cb.Do(t.Context(), t.Name(), func() error { t.Error("opened callback ran"); return nil }); !errors.Is(err, circuitbreaker.ErrOpened) {
		t.Fatal(err)
	}
}
