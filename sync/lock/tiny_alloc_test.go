package lock_test

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/lock"
)

var tinyNeighbours []*[8]byte

// Entries must not be tiny-allocated: a long-lived small neighbour in the same
// 16-byte block would otherwise keep every per-key entry alive forever.
func TestKeyLockEntriesReclaimedWithTinyNeighbours(t *testing.T) {
	l := lock.New()
	const n = 1000
	for i := range n {
		l.Lock(fmt.Sprint(i)).Unlock()
		tinyNeighbours = append(tinyNeighbours, new([8]byte))
	}
	// Cleanups run asynchronously after collection, so poll.
	deadline := time.Now().Add(2 * time.Second)
	for l.Size() > n/10 && time.Now().Before(deadline) {
		runtime.GC()
		time.Sleep(5 * time.Millisecond)
	}
	if got := l.Size(); got > n/10 {
		t.Fatalf("%d of %d idle entries survived GC", got, n)
	}
	tinyNeighbours = nil
}
