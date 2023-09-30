package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestFixedWindow(t *testing.T) {
	rl := ratelimit.NewFixedWindow(3, time.Second)

	assert := assert.New(t)

	// Succeed.
	res := rl.Allow()
	assert.True(res.Allow)
	assert.Equal(int64(2), res.Remaining)

	res = rl.Allow()
	assert.True(res.Allow)
	assert.Equal(int64(1), res.Remaining)

	res = rl.Allow()
	assert.True(res.Allow)
	assert.Equal(int64(0), res.Remaining)

	res = rl.Allow()
	assert.False(res.Allow)
	assert.Equal(int64(0), res.Remaining)

	// Call after the next period succeeds.
	rl.Now = func() time.Time {
		return time.Now().Add(time.Second)
	}

	res = rl.Allow()
	assert.True(res.Allow)
	assert.Equal(int64(2), res.Remaining)
}
