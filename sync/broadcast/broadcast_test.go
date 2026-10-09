package broadcast_test

import (
	"github.com/alextanhongpin/core/sync/broadcast"
	"sync"
	"testing"
	"time"
)

func TestStopWithUnreadSubscriber(t *testing.T) {
	b, stop := broadcast.New[int]()
	ch := b.Chan()
	b.Send(42)
	done := make(chan struct{})
	go func() { stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stop blocked on unread subscriber")
	}
	if _, ok := <-ch; ok {
		t.Fatal("subscriber remains open")
	}
}

func TestConcurrentGoAndStop(t *testing.T) {
	for range 100 {
		b, stop := broadcast.New[int]()
		var wg sync.WaitGroup
		wg.Go(func() { b.Go(func(int) {}) })
		wg.Go(stop)
		wg.Wait()
		stop()
	}
}
