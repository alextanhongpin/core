package background_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/background"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestBackground(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		task := &numberTask{}

		opt := background.Option[int]{
			Handler: task,
		}
		bg, stop := background.New(opt)
		defer stop()

		assert := assert.New(t)
		bg.Exec(ctx, 1)

		stop()

		assert.Equal([]int{1}, task.Numbers())
	})

	t.Run("early stop", func(t *testing.T) {
		task := &numberTask{}

		opt := background.Option[int]{
			Handler: task,
		}
		bg, stop := background.New(opt)

		assert := assert.New(t)
		bg.Exec(ctx, 1)
		stop()

		assert.Equal([]int{1}, task.Numbers())

		bg.Exec(ctx, 2)
		stop()

		assert.Equal([]int{1, 2}, task.Numbers())
	})

	t.Run("send wait", func(t *testing.T) {
		task := &numberTask{sleep: 50 * time.Millisecond}

		opt := background.Option[int]{
			Handler: task,
		}

		bg, stop := background.New(opt)
		defer stop()
		assert := assert.New(t)
		bg.ExecWait(ctx, 1, 2, 3)

		assert.ElementsMatch([]int{1, 2, 3}, task.Numbers())
	})
}

type numberTask struct {
	sleep time.Duration
	nums  []int
	mu    sync.RWMutex
}

func (t *numberTask) Exec(ctx context.Context, n int) {
	// Pretend to do some work.
	time.Sleep(t.sleep)

	t.mu.Lock()
	t.nums = append(t.nums, n)
	t.mu.Unlock()
}

func (t *numberTask) Numbers() []int {
	t.mu.RLock()

	nums := make([]int, len(t.nums))
	copy(nums, t.nums)

	t.mu.RUnlock()
	return nums
}
