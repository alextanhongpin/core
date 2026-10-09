package lock_test

import (
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/lock"
)

func TestKeyLockContentionDoesNotBlockOtherKeys(t *testing.T) {
	var l lock.KeyLock
	held := l.Lock("a")
	waiting := make(chan struct{})
	done := make(chan struct{})
	go func() {
		close(waiting)
		next := l.Lock("a")
		next.Unlock()
		close(done)
	}()
	<-waiting
	// Repeated acquisitions allow the waiter to enter Lock while the first
	// key remains held. No key should require releasing that first lock.
	independent := make(chan struct{})
	go func() {
		for range 100 {
			other := l.Lock("b")
			other.Unlock()
		}
		close(independent)
	}()
	select {
	case <-independent:
	case <-time.After(time.Second):
		held.Unlock()
		<-done
		<-independent
		t.Fatal("contention on a blocked b")
	}
	held.Unlock()
	<-done
}

func TestTryLockZeroValue(t *testing.T) {
	var l lock.TryLock
	if !l.TryLock("a") || l.TryLock("a") {
		t.Fatal("incorrect acquisition")
	}
	l.Unlock("a")
	if !l.TryLock("a") {
		t.Fatal("unlock did not release key")
	}
	l.Unlock("a")
}

func TestKeyLockMutualExclusion(t *testing.T) {
	var l lock.KeyLock
	var wg sync.WaitGroup
	count := 0
	for range 20 {
		wg.Go(func() {
			for range 100 {
				held := l.Lock("shared")
				count++
				held.Unlock()
			}
		})
	}
	wg.Wait()
	if count != 2000 {
		t.Fatalf("got %d", count)
	}
}
