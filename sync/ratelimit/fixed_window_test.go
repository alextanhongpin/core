package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestFixedWindow(t *testing.T) {
	rl := ratelimit.MustNewFixedWindow(3, time.Second)

	is := assert.New(t)

	// Succeed.
	got := rl.Limit()
	is.True(got.Allow)
	is.Equal(int64(2), got.Remaining)

	got = rl.Limit()
	is.True(got.Allow)
	is.Equal(int64(1), got.Remaining)

	got = rl.Limit()
	is.True(got.Allow)
	is.Equal(int64(0), got.Remaining)

	got = rl.Limit()
	is.False(got.Allow)
	is.Equal(int64(0), got.Remaining)

	// Call after the next period succeeds.
	rl.Now = func() time.Time {
		return time.Now().Add(time.Second)
	}

	got = rl.Limit()
	is.True(got.Allow)
	is.Equal(int64(2), got.Remaining)
}
