package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

func TestRateLimiter(t *testing.T) {
	rl := ratelimit.New(
		&throttler{ratelimit.NewFixedWindow(3, time.Second)}, // Max 3 request in one second.
		ratelimit.NewGCRA(10, time.Second, 0),                // 10 request per second, means 1 req every 100ms.
	)
	var count int
	for range 10 {
		time.Sleep(100 * time.Millisecond)
		if rl.Allow() {
			count++
		}
	}
	if want := 3; count != want {
		t.Fatalf("want %v, got %v", want, count)
	}
}

type throttler struct {
	rl *ratelimit.FixedWindow
}

func (t *throttler) Allow() bool {
	return t.rl.Allow()
}

func (t *throttler) AllowN(n int) bool {
	return t.rl.AllowN(n)
}
