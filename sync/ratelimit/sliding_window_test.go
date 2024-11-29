package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestSlidingWindow(t *testing.T) {
	rl := ratelimit.NewSlidingWindow(1, time.Second)

	is := assert.New(t)
	is.True(rl.Allow())
	is.False(rl.Allow())
}

func TestSlidingWindow_RateLimited(t *testing.T) {
	rl := ratelimit.NewSlidingWindow(5, time.Second)

	var count int
	for i := 0; i < 10; i++ {
		if rl.Allow() {
			count++
		}
	}

	is := assert.New(t)
	is.Equal(5, count)
}

/*
func TestSlidingWindow_RetryAt(t *testing.T) {
	t.Run("per second", func(t *testing.T) {
		rl := ratelimit.NewSlidingWindow(5, time.Second)
		for range 6 {
			rl.Allow()
		}

		is := assert.New(t)
		is.True(rl.Allow().RetryAfter > 950*time.Millisecond)
	})

	t.Run("per hour", func(t *testing.T) {
		rl := ratelimit.NewSlidingWindow(5, time.Hour)
		for range 6 {
			rl.Allow()
		}

		is := assert.New(t)
		is.True(rl.Allow().RetryAfter > 59*time.Minute+50*time.Second)
	})
}
*/
