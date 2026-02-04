package bench

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

var atom = MustNewGCRA(100, time.Second, 10)

var mutex = ratelimit.MustNewGCRA(100, time.Second, 10)

func BenchmarkGCRAAtom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		atom.Allow() // Code to measure
	}
}

func BenchmarkGCRAMutex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mutex.Allow() // Code to measure
	}
}
