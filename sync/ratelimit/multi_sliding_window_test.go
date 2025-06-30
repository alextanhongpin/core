package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestMultiSlidingWindow_Clear(t *testing.T) {
	r := ratelimit.MustNewMultiSlidingWindow(5, time.Second)

	var foo, bar int
	for range 10 {
		if r.Allow("foo") {
			foo++
		}

		if r.Allow("bar") {
			bar++
		}
	}

	is := assert.New(t)
	is.Equal(5, foo)
	is.Equal(5, bar)
	is.Equal(2, r.Size())

	r.Now = func() time.Time {
		// There's an additional 400ms, because
		// we incremented the timestamp by the
		// interval twice.
		return time.Now().Add(time.Second)
	}
	r.Clear()
	is.Equal(0, r.Size())
}
