package snapshot_test

import (
	"github.com/alextanhongpin/core/sync/snapshot"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func TestZeroDurationAndConfigOwnership(t *testing.T) {
	cfg := &snapshot.Config{Policies: []snapshot.Policy{{Changes: 1, After: 0}}}
	s, stop := snapshot.New(cfg)
	defer stop()
	cfg.Policies[0].Changes = 100
	s.Policies[0].Changes = 200
	ch := s.Chan()
	s.Inc()
	select {
	case p := <-ch:
		if p.Changes != 1 {
			t.Fatalf("got %+v", p)
		}
	case <-time.After(time.Second):
		t.Fatal("zero-duration policy not delivered")
	}
}

func TestStopWithUnreadSubscriber(t *testing.T) {
	s, stop := snapshot.New(&snapshot.Config{Policies: []snapshot.Policy{{Changes: 1}}})
	s.Chan()
	s.Inc()
	done := make(chan struct{})
	go func() { stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown blocked")
	}
}

func TestConcurrentAddAndStop(t *testing.T) {
	for range 100 {
		s, stop := snapshot.New(snapshot.DefaultConfig())
		var wg sync.WaitGroup
		wg.Go(func() {
			for range 100 {
				s.Inc()
			}
		})
		wg.Go(stop)
		wg.Wait()
	}
}

func TestPolicyTimerSurvivesContinuousChanges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, stop := snapshot.New(&snapshot.Config{Policies: []snapshot.Policy{{Changes: 1, After: time.Second}}})
		defer stop()
		ch := s.Chan()
		for range 9 {
			s.Inc()
			time.Sleep(100 * time.Millisecond)
		}
		time.Sleep(100 * time.Millisecond)
		if p := <-ch; p.After != time.Second {
			t.Fatalf("got %+v", p)
		}
	})
}
