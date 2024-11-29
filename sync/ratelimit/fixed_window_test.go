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
	is.True(rl.Allow())
	is.Equal(2, rl.Remaining())

	is.True(rl.Allow())
	is.Equal(1, rl.Remaining())

	is.True(rl.Allow())
	is.Equal(0, rl.Remaining())

	is.False(rl.Allow())
	is.Equal(0, rl.Remaining())

	// Call after the next period succeeds.
	rl.Now = func() time.Time {
		return time.Now().Add(time.Second)
	}

	is.True(rl.Allow())
	is.Equal(2, rl.Remaining())
}
