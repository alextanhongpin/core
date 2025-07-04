package background_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/alextanhongpin/core/sync/background"
)

func BenchmarkWorkerPool(b *testing.B) {
	ctx := context.Background()

	b.Run("unbuffered", func(b *testing.B) {
		var processed int64

		opts := background.Options{
			WorkerCount: 4,
			BufferSize:  0,
		}

		worker, stop := background.NewWithOptions(ctx, opts, func(ctx context.Context, n int) {
			atomic.AddInt64(&processed, 1)
		})
		defer stop()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				worker.Send(1)
			}
		})
	})

	b.Run("buffered", func(b *testing.B) {
		var processed int64

		opts := background.Options{
			WorkerCount: 4,
			BufferSize:  100,
		}

		worker, stop := background.NewWithOptions(ctx, opts, func(ctx context.Context, n int) {
			atomic.AddInt64(&processed, 1)
		})
		defer stop()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				worker.Send(1)
			}
		})
	})

	b.Run("try_send", func(b *testing.B) {
		var processed int64

		opts := background.Options{
			WorkerCount: 4,
			BufferSize:  100,
		}

		worker, stop := background.NewWithOptions(ctx, opts, func(ctx context.Context, n int) {
			atomic.AddInt64(&processed, 1)
		})
		defer stop()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				worker.TrySend(1)
			}
		})
	})
}

func BenchmarkMetrics(b *testing.B) {
	ctx := context.Background()

	opts := background.Options{
		WorkerCount: 4,
		BufferSize:  100,
	}

	worker, stop := background.NewWithOptions(ctx, opts, func(ctx context.Context, n int) {
		// Do nothing
	})
	defer stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			worker.Metrics()
		}
	})
}
