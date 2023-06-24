package background_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/background"
	"github.com/stretchr/testify/assert"
)

type numberTask struct {
	numbers []int
	delay   time.Duration
	mu      sync.RWMutex
}

func (t *numberTask) sleep() {
	time.Sleep(t.delay)
}

func (t *numberTask) Exec(n int) {
	t.sleep()
	fmt.Println("exec", n)
	t.mu.Lock()
	t.numbers = append(t.numbers, n)
	t.mu.Unlock()
}

func (t *numberTask) Numbers() []int {
	t.mu.RLock()
	numbers := make([]int, len(t.numbers))
	copy(numbers, t.numbers)
	t.mu.RUnlock()
	return numbers
}

func TestBackground(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		task := &numberTask{}
		bg, stop := background.New[int](task)
		defer stop()
		bg.Send(1)
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, []int{1}, task.Numbers())
	})

	t.Run("early stop", func(t *testing.T) {
		task := &numberTask{delay: 100 * time.Millisecond}
		bg, stop := background.New[int](task)
		bg.Send(1)
		stop()
		bg.Send(2)
		task.sleep()
		assert.Equal(t, []int{1, 2}, task.Numbers())
	})

	t.Run("buffer with early stop", func(t *testing.T) {
		task := &numberTask{delay: 100 * time.Millisecond}
		bg, stop := background.New[int](task, background.Buffer(2))
		bg.Send(1)
		bg.Send(2)
		stop()
		bg.Send(3)
		task.sleep()
		task.sleep()

		assert.Equal(t, []int{1, 2, 3}, task.Numbers())
	})
}
