package background_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/background"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

func TestBackground(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		is := assert.New(t)
		bg, stop := background.New(ctx, -1, func(ctx context.Context, n int) {
			is.Equal(42, n)
		})
		defer stop()

		is.Nil(bg.Send(42))
	})

	t.Run("early stop", func(t *testing.T) {
		is := assert.New(t)
		bg, stop := background.New(ctx, -1, func(ctx context.Context, n int) {
			is.Equal(42, n)
		})
		stop()

		is.ErrorIs(bg.Send(1), background.ErrTerminated)
	})
}

func TestBackgroundWithOptions(t *testing.T) {
	t.Run("buffered channel", func(t *testing.T) {
		is := assert.New(t)
		var processed int32

		opts := background.Options{
			WorkerCount: 1,
			BufferSize:  10,
		}

		bg, stop := background.NewWithOptions(ctx, opts, func(ctx context.Context, n int) {
			atomic.AddInt32(&processed, 1)
		})
		defer stop()

		// Send multiple tasks quickly
		for i := 0; i < 5; i++ {
			is.NoError(bg.Send(i))
		}

		// Wait for processing
		time.Sleep(100 * time.Millisecond)
		is.Equal(int32(5), atomic.LoadInt32(&processed))
	})

	t.Run("worker timeout", func(t *testing.T) {
		is := assert.New(t)
		var started, completed int32

		opts := background.Options{
			WorkerCount:   1,
			WorkerTimeout: 50 * time.Millisecond,
		}

		bg, stop := background.NewWithOptions(ctx, opts, func(ctx context.Context, n int) {
			atomic.AddInt32(&started, 1)

			select {
			case <-time.After(100 * time.Millisecond):
				atomic.AddInt32(&completed, 1)
			case <-ctx.Done():
				// Task was cancelled due to timeout
			}
		})
		defer stop()

		is.NoError(bg.Send(1))
		time.Sleep(200 * time.Millisecond)

		// Task should have started but not completed due to timeout
		is.Equal(int32(1), atomic.LoadInt32(&started))
		is.Equal(int32(0), atomic.LoadInt32(&completed))
	})

	t.Run("error handling", func(t *testing.T) {
		is := assert.New(t)
		var errorCount int32

		opts := background.Options{
			WorkerCount: 1,
			OnError: func(task interface{}, recovered interface{}) {
				atomic.AddInt32(&errorCount, 1)
				is.Equal("test panic", recovered)
			},
		}

		bg, stop := background.NewWithOptions(ctx, opts, func(ctx context.Context, n int) {
			panic("test panic")
		})
		defer stop()

		is.NoError(bg.Send(1))
		time.Sleep(100 * time.Millisecond)

		is.Equal(int32(1), atomic.LoadInt32(&errorCount))
	})

	t.Run("task completion callback", func(t *testing.T) {
		is := assert.New(t)
		var completedTasks int32
		var totalDuration time.Duration

		opts := background.Options{
			WorkerCount: 1,
			OnTaskComplete: func(task interface{}, duration time.Duration) {
				atomic.AddInt32(&completedTasks, 1)
				totalDuration += duration
			},
		}

		bg, stop := background.NewWithOptions(ctx, opts, func(ctx context.Context, n int) {
			time.Sleep(10 * time.Millisecond)
		})
		defer stop()

		is.NoError(bg.Send(1))
		is.NoError(bg.Send(2))
		time.Sleep(100 * time.Millisecond)

		is.Equal(int32(2), atomic.LoadInt32(&completedTasks))
		is.True(totalDuration > 20*time.Millisecond)
	})
}

func TestTrySend(t *testing.T) {
	t.Run("context cancelled", func(t *testing.T) {
		is := assert.New(t)

		bg, stop := background.New(ctx, 1, func(ctx context.Context, n int) {})
		stop() // Cancel immediately

		is.False(bg.TrySend(1))
	})

	t.Run("basic functionality", func(t *testing.T) {
		is := assert.New(t)
		var processed int32

		opts := background.Options{
			WorkerCount: 1,
			BufferSize:  10, // Large buffer
		}

		bg, stop := background.NewWithOptions(ctx, opts, func(ctx context.Context, n int) {
			atomic.AddInt32(&processed, 1)
		})
		defer stop()

		// Should succeed with large buffer
		is.True(bg.TrySend(1))

		// Wait for processing
		time.Sleep(50 * time.Millisecond)
		is.Equal(int32(1), atomic.LoadInt32(&processed))
	})
}

func TestMetrics(t *testing.T) {
	t.Run("basic metrics", func(t *testing.T) {
		is := assert.New(t)
		var wg sync.WaitGroup

		opts := background.Options{
			WorkerCount: 2,
			BufferSize:  5,
		}

		bg, stop := background.NewWithOptions(ctx, opts, func(ctx context.Context, n int) {
			wg.Done()
		})
		defer stop()

		// Wait for workers to start
		time.Sleep(50 * time.Millisecond)

		// Send some tasks
		wg.Add(3)
		for i := 0; i < 3; i++ {
			is.NoError(bg.Send(i))
		}

		wg.Wait()

		metrics := bg.Metrics()
		is.Equal(int64(3), metrics.TasksQueued)
		is.Equal(int64(3), metrics.TasksProcessed)
		is.Equal(int64(0), metrics.TasksRejected)
		is.True(metrics.ActiveWorkers >= 1) // At least 1 worker should be active
	})

	t.Run("rejected tasks", func(t *testing.T) {
		is := assert.New(t)

		bg, stop := background.New(ctx, 1, func(ctx context.Context, n int) {})
		stop() // Cancel immediately

		// Try to send after cancellation
		is.Error(bg.Send(1))
		is.False(bg.TrySend(2))

		metrics := bg.Metrics()
		is.Equal(int64(0), metrics.TasksQueued)
		is.Equal(int64(0), metrics.TasksProcessed)
		is.Equal(int64(2), metrics.TasksRejected)
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("high concurrency", func(t *testing.T) {
		is := require.New(t)

		const numTasks = 1000
		var processed int64

		opts := background.Options{
			WorkerCount: 10,
			BufferSize:  100,
		}

		bg, stop := background.NewWithOptions(ctx, opts, func(ctx context.Context, n int) {
			atomic.AddInt64(&processed, 1)
		})
		defer stop()

		// Send tasks concurrently
		var wg sync.WaitGroup
		wg.Add(numTasks)

		for i := 0; i < numTasks; i++ {
			go func(i int) {
				defer wg.Done()
				is.NoError(bg.Send(i))
			}(i)
		}

		wg.Wait()

		// Wait for processing
		time.Sleep(200 * time.Millisecond)

		is.Equal(int64(numTasks), atomic.LoadInt64(&processed))

		metrics := bg.Metrics()
		is.Equal(int64(numTasks), metrics.TasksQueued)
		is.Equal(int64(numTasks), metrics.TasksProcessed)
		is.Equal(int64(0), metrics.TasksRejected)
	})
}
