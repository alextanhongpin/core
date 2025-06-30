package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	rl := ratelimit.New(
		ratelimit.MustNewFixedWindow(3, time.Second), // Max 3 request in one second.
		ratelimit.MustNewGCRA(10, time.Second, 0),    // 10 request per second, means 1 req every 100ms.
	)

	var count int
	for range 10 {
		if rl.Allow() {
			count++
		}
		time.Sleep(100 * time.Millisecond)
	}

	is := assert.New(t)
	is.Equal(3, count)
}
