package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestMultiGCRA_Clear(t *testing.T) {
	r := ratelimit.MustNewMultiGCRA(5, time.Second, 1)

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
	is.Equal(2, foo)
	is.Equal(2, bar)
	is.Equal(2, r.Size())

	r.Now = func() time.Time {
		// There's an additional 400ms, because
		// we incremented the timestamp by the
		// interval twice.
		return time.Now().Add(time.Second).Add(2 * 200 * time.Millisecond)
	}
	r.Clear()
	is.Equal(0, r.Size())
}
