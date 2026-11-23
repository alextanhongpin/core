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

func TestChanAfterStopIsClosed(t *testing.T) {
	b, stop := broadcast.New[int]()
	stop()
	ch := b.Chan()
	if ch == nil {
		t.Fatal("nil channel")
	}
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected closed channel")
		}
	case <-time.After(time.Second):
		t.Fatal("channel not closed")
	}
	ch2, unsub := b.Subscribe()
	unsub()
	if _, ok := <-ch2; ok {
		t.Fatal("expected closed channel")
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	b, stop := broadcast.New[int]()
	defer stop()
	ch, unsub := b.Subscribe()
	other := b.Chan()
	go b.Send(1)
	if v := <-ch; v != 1 {
		t.Fatalf("got %d", v)
	}
	if v := <-other; v != 1 {
		t.Fatalf("got %d", v)
	}
	unsub()
	unsub() // idempotent
	if _, ok := <-ch; ok {
		t.Fatal("channel not closed after unsubscribe")
	}
	go b.Send(2)
	select {
	case v := <-other:
		if v != 2 {
			t.Fatalf("got %d", v)
		}
	case <-time.After(time.Second):
		t.Fatal("remaining subscriber starved")
	}
}

func TestUnsubscribeWhileSendBlocked(t *testing.T) {
	b, stop := broadcast.New[int]()
	defer stop()
	_, unsub := b.Subscribe() // never read: Send blocks delivering to it
	other := b.Chan()
	sent := make(chan struct{})
	go func() { b.Send(1); b.Send(2); close(sent) }()
	time.Sleep(50 * time.Millisecond) // let the dispatcher block on the first subscriber
	unsubDone := make(chan struct{})
	go func() { unsub(); close(unsubDone) }()
	select {
	case <-unsubDone:
	case <-time.After(time.Second):
		t.Fatal("unsubscribe blocked behind in-progress Send")
	}
	for want := 1; want <= 2; want++ {
		select {
		case v := <-other:
			if v != want {
				t.Fatalf("got %d want %d", v, want)
			}
		case <-time.After(time.Second):
			t.Fatal("delivery did not resume after unsubscribe")
		}
	}
	<-sent
}

func TestUnsubscribeStress(t *testing.T) {
	b, stop := broadcast.New[int]()
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 50 {
				ch, unsub := b.Subscribe()
				select {
				case <-ch:
				default:
				}
				unsub()
			}
		})
	}
	wg.Go(func() {
		for i := range 200 {
			b.Send(i)
		}
	})
	wg.Wait()
	stop()
}
