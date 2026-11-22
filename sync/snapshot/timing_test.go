package snapshot_test

import (
	"math"
	"testing"
	"testing/synctest"
	"time"

	"github.com/alextanhongpin/core/sync/snapshot"
)

func TestTimeFiringNotLateAfterAddFires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, stop := snapshot.MustNew(snapshot.Config{Policies: []snapshot.Policy{{Changes: 1, After: time.Second}}})
		defer stop()
		start := time.Now()
		fired := make(chan time.Duration, 10)
		s.Go(func(snapshot.Policy) { fired <- time.Since(start) })

		time.Sleep(1200 * time.Millisecond)
		s.Add(1) // fires from Add at 1.2s, off the tick cadence
		time.Sleep(100 * time.Millisecond)
		s.Add(1) // due one second after the previous firing: 2.2s
		time.Sleep(950 * time.Millisecond)
		synctest.Wait()

		if len(fired) != 2 {
			t.Fatalf("got %d notifications by 2.25s, want 2", len(fired))
		}
		<-fired
		if at := <-fired; at != 2200*time.Millisecond {
			t.Fatalf("second notification at %v, want 2.2s", at)
		}
	})
}

func TestAddIgnoresNonPositive(t *testing.T) {
	s, stop := snapshot.MustNew(snapshot.Config{Policies: []snapshot.Policy{{Changes: 2}}})
	defer stop()
	ch := s.Chan()
	s.Add(-5)
	s.Add(0)
	s.Add(1)
	select {
	case p := <-ch:
		t.Fatalf("unexpected notification %+v", p)
	case <-time.After(50 * time.Millisecond):
	}
	s.Add(1)
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("no notification after two positive changes")
	}
}

func TestAddSaturates(t *testing.T) {
	s, stop := snapshot.MustNew(snapshot.Config{Policies: []snapshot.Policy{{Changes: math.MaxInt}}})
	defer stop()
	ch := s.Chan()
	s.Add(math.MaxInt - 1)
	s.Add(5) // would overflow to a negative count without saturation
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("count overflowed instead of saturating")
	}
}

func TestEqualAfterPoliciesKeepOrder(t *testing.T) {
	policies := []snapshot.Policy{{Changes: 5, After: 0}, {Changes: 3, After: 0}, {Changes: 1, After: 0}}
	s, stop := snapshot.MustNew(snapshot.Config{Policies: policies})
	defer stop()
	ch := s.Chan()
	s.Add(3)
	if p := <-ch; p.Changes != 3 {
		t.Fatalf("got %+v, want the stable-ordered second policy", p)
	}
}
