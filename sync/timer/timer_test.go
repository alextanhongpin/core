package timer_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/timer"
)

func TestSetIntervalInvalidDuration(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected caller panic")
		}
	}()
	timer.SetInterval(func() {}, 0)
}

func TestSetIntervalStopWaitsForCallback(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int64
	stop := timer.SetInterval(func() {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
	}, time.Millisecond)
	<-started
	stopped := make(chan struct{})
	go func() { stop(); close(stopped) }()
	select {
	case <-stopped:
		t.Fatal("stop returned during callback")
	default:
	}
	close(release)
	<-stopped
	count := calls.Load()
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() { defer wg.Done(); stop() }()
	}
	wg.Wait()
	if calls.Load() != count {
		t.Fatal("callback after stop")
	}
}

func TestSetTimeoutCancel(t *testing.T) {
	cancel := timer.SetTimeout(func() { t.Error("canceled callback ran") }, time.Hour)
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() { defer wg.Done(); cancel() }()
	}
	wg.Wait()
}
