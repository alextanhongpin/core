package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestFixedWindow(t *testing.T) {
	rl := ratelimit.NewFixedWindow(3, time.Second)

	is := assert.New(t)

	// Succeed.
	res := rl.Allow()
	is.True(res.Allow)
	is.Equal(int64(2), res.Remaining)

	res = rl.Allow()
	is.True(res.Allow)
	is.Equal(int64(1), res.Remaining)

	res = rl.Allow()
	is.True(res.Allow)
	is.Equal(int64(0), res.Remaining)

	res = rl.Allow()
	is.False(res.Allow)
	is.Equal(int64(0), res.Remaining)

	// Call after the next period succeeds.
	rl.Now = func() time.Time {
		return time.Now().Add(time.Second)
	}

	res = rl.Allow()
	is.True(res.Allow)
	is.Equal(int64(2), res.Remaining)
}
