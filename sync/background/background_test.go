package background_test

import (
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/background"
	"github.com/stretchr/testify/assert"
)

type numberTask struct {
	sleep time.Duration
	nums  []int
	mu    sync.RWMutex
}

func (t *numberTask) Exec(n int) {
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

func TestBackground(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		task := &numberTask{}
		bg, stop := background.New[int](task)
		defer stop()

		assert := assert.New(t)
		assert.Nil(bg.Send(1))

		stop()

		assert.Equal([]int{1}, task.Numbers())
	})

	t.Run("early stop", func(t *testing.T) {
		task := &numberTask{}
		bg, stop := background.New[int](task)

		assert := assert.New(t)
		assert.Nil(bg.Send(1))
		stop()

		assert.Equal(background.Stopped, bg.Send(2))
		assert.Equal([]int{1}, task.Numbers())
	})

	t.Run("concurrent send", func(t *testing.T) {
		assert := assert.New(t)

		task := &numberTask{}
		bg, stop := background.New[int](task)

		race := make(chan bool)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()

			<-race
			assert.Nil(bg.Send(1))
		}()

		go func() {
			defer wg.Done()

			<-race
			assert.Nil(bg.Send(2))
		}()

		// Signal sending.
		close(race)
		wg.Wait()

		stop()
		assert.ElementsMatch([]int{1, 2}, task.Numbers())
	})

	t.Run("buffer with early stop", func(t *testing.T) {
		task := &numberTask{sleep: 50 * time.Millisecond}
		bg, stop := background.New[int](task)

		assert := assert.New(t)
		assert.Nil(bg.Send(1))
		assert.Nil(bg.Send(2))
		assert.Nil(bg.Send(3))
		stop()
		assert.Equal(background.Stopped, bg.Send(4))

		assert.ElementsMatch([]int{1, 2, 3}, task.Numbers())
	})
}
