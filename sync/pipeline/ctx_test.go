package pipeline_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/alextanhongpin/core/sync/pipeline"
)

// ctxCase builds a stage over an endless, never-drained producer. Outputs are
// never read, so without cancellation the stage blocks on send forever.
// After cancel, every output must close and no goroutine may remain.
func runCtxCase[V any](t *testing.T, build func(ctx context.Context, in <-chan int) []<-chan V) {
	t.Helper()
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		in := make(chan int)
		go func() { // endless producer that also exits on cancel
			for i := 0; ; i++ {
				select {
				case in <- i:
				case <-ctx.Done():
					return
				}
			}
		}()
		outs := build(ctx, in)
		synctest.Wait() // stage is now blocked sending to an unread output
		cancel()
		synctest.Wait()
		for _, out := range outs {
			// Blocks (and synctest reports a deadlock) unless the output closed.
			for range out {
			}
		}
		// synctest.Test fails if goroutines are still blocked when it returns
		// (deadlock), so reaching here proves nothing is left blocked on send.
	})
}

func TestBatchCtxCancel(t *testing.T) {
	runCtxCase(t, func(ctx context.Context, in <-chan int) []<-chan []int {
		return []<-chan []int{pipeline.BatchCtx(ctx, in, 2, time.Hour)}
	})
}

func TestPipeCtxCancel(t *testing.T) {
	runCtxCase(t, func(ctx context.Context, in <-chan int) []<-chan int {
		return []<-chan int{pipeline.PipeCtx(ctx, in, func(v int) (int, bool) { return v, true })}
	})
}

func TestPipeNCtxCancel(t *testing.T) {
	runCtxCase(t, func(ctx context.Context, in <-chan int) []<-chan int {
		return []<-chan int{pipeline.PipeNCtx(ctx, in, func(v int) (int, bool) { return v, true }, 4)}
	})
}

func TestSemaphoreCtxCancel(t *testing.T) {
	runCtxCase(t, func(ctx context.Context, in <-chan int) []<-chan int {
		return []<-chan int{pipeline.SemaphoreCtx(ctx, in, func(v int) int { return v }, 3)}
	})
}

func TestTeeCtxCancel(t *testing.T) {
	runCtxCase(t, func(ctx context.Context, in <-chan int) []<-chan int {
		a, b := pipeline.TeeCtx(ctx, in)
		return []<-chan int{a, b}
	})
}

func TestCtxVariantsCancelWhileIdle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		in := make(chan int) // never sends, never closes
		b := pipeline.BatchCtx(ctx, in, 2, time.Hour)
		p := pipeline.PipeNCtx(ctx, in, func(v int) (int, bool) { return v, true }, 2)
		s := pipeline.SemaphoreCtx(ctx, in, func(v int) int { return v }, 2)
		t1, t2 := pipeline.TeeCtx(ctx, in)
		synctest.Wait()
		cancel()
		synctest.Wait()
		if _, ok := <-b; ok {
			t.Fatal("batch output open")
		}
		if _, ok := <-p; ok {
			t.Fatal("pipe output open")
		}
		if _, ok := <-s; ok {
			t.Fatal("semaphore output open")
		}
		if _, ok := <-t1; ok {
			t.Fatal("tee1 open")
		}
		if _, ok := <-t2; ok {
			t.Fatal("tee2 open")
		}
	})
}

func TestCtxVariantsNormalFlow(t *testing.T) {
	ctx := context.Background()
	got := pipeline.Collect(pipeline.PipeCtx(ctx, pipeline.SourceSlice(ctx, []int{1, 2, 3}), func(v int) (int, bool) { return v * 2, v != 2 }))
	if len(got) != 2 || got[0] != 2 || got[1] != 6 {
		t.Fatalf("got %v", got)
	}
	a, b := pipeline.TeeCtx(ctx, pipeline.SourceSlice(ctx, []int{1, 2}))
	done := make(chan []int)
	go func() { done <- pipeline.Collect(b) }()
	if g := pipeline.Collect(a); len(g) != 2 {
		t.Fatalf("got %v", g)
	}
	if g := <-done; len(g) != 2 {
		t.Fatalf("got %v", g)
	}
	batches := pipeline.Collect(pipeline.BatchCtx(ctx, pipeline.SourceSlice(ctx, []int{1, 2, 3}), 2, time.Hour))
	if len(batches) != 2 || len(batches[1]) != 1 {
		t.Fatalf("got %v", batches)
	}
	sem := pipeline.Collect(pipeline.SemaphoreCtx(ctx, pipeline.SourceSlice(ctx, []int{1, 2, 3}), func(v int) int { return v }, 2))
	if len(sem) != 3 {
		t.Fatalf("got %v", sem)
	}
}
