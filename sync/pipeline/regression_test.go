package pipeline_test

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/alextanhongpin/core/sync/pipeline"
)

func TestSourceChanClosesWithInput(t *testing.T) {
	in := make(chan int, 1)
	in <- 42
	close(in)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	out := pipeline.SourceChan(ctx, in)
	if got := <-out; got != 42 {
		t.Fatalf("got %d", got)
	}
	select {
	case _, ok := <-out:
		if ok {
			t.Fatal("output still open")
		}
	case <-time.After(time.Second):
		t.Fatal("closed input did not close output")
	}
}

func TestBatchTimeoutStartsWithFirstItem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		in := make(chan int)
		out := pipeline.Batch(in, 10, time.Second)
		in <- 1
		time.Sleep(500 * time.Millisecond)
		in <- 2
		var batch []int
		go func() { batch = <-out }()
		time.Sleep(500 * time.Millisecond)
		synctest.Wait()
		if len(batch) != 2 {
			t.Fatalf("batch did not flush at first item's deadline: %v", batch)
		}
		close(in)
		pipeline.Flush(out)
	})
}

func TestPipeNRejectsNonpositiveWorkers(t *testing.T) {
	for _, n := range []int{0, -1} {
		t.Run(time.Duration(n).String(), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()
			pipeline.PipeN(make(chan int), func(v int) (int, bool) { return v, true }, n)
		})
	}
}

func TestRateLimitRejectsInvalidArguments(t *testing.T) {
	for _, tc := range []struct {
		every    int
		interval time.Duration
	}{{0, time.Second}, {-1, time.Second}, {1, 0}, {2, time.Nanosecond}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("expected synchronous panic")
				}
			}()
			pipeline.RateLimit(make(chan int), tc.every, tc.interval)
		}()
	}
}

func TestSemaphoreBoundsPendingWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		in := make(chan int)
		release := make(chan struct{})
		var sent atomic.Int64
		out := pipeline.Semaphore(in, func(v int) int { <-release; return v }, 2)
		go func() {
			defer close(in)
			for i := range 20 {
				in <- i
				sent.Add(1)
			}
		}()
		synctest.Wait()
		if got := sent.Load(); got != 3 {
			t.Errorf("accepted %d items while workers blocked; want 3", got)
		}
		close(release)
		if got := pipeline.Count(out); got != 20 {
			t.Fatalf("got %d results", got)
		}
	})
}
