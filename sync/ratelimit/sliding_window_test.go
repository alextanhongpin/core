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
