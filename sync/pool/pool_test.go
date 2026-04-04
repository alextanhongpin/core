package pool_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/pool"
)

func TestTTL(t *testing.T) {
	var before []int64
	var after []int64
	var a atomic.Int64
	fn := func() (int64, func()) {
		n := a.Add(1)
		before = append(before, n)
		return n, func() {
			after = append(after, n)
		}
	}
	p := pool.New(fn, 1, time.Second)
	defer p.Done()

	for range 2 {
		_, done := p.Borrow()
		go func() {
			time.Sleep(200 * time.Millisecond)
			done()
		}()
	}

	p.Done()

	for i := range before {
		want, got := before[i], after[i]
		if want != got {
			t.Fatalf("want %d, got %d", want, got)
		}
	}
}
