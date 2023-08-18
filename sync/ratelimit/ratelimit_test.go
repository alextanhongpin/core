package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	rl := ratelimit.New(3, time.Second)

	assert := assert.New(t)

	// Succeed.
	assert.True(rl.Allow())
	assert.Equal(2, rl.Remaining())

	assert.True(rl.Allow())
	assert.Equal(1, rl.Remaining())

	assert.True(rl.Allow())
	assert.Equal(0, rl.Remaining())

	assert.False(rl.Allow())
	assert.Equal(0, rl.Remaining())

	// Call after the next period succeeds.
	rl.SetNow(func() time.Time {
		return time.Now().Add(time.Second)
	})

	assert.True(rl.Allow())
	assert.Equal(2, rl.Remaining())
}

func TestMultiRateLimit(t *testing.T) {
	rl := ratelimit.NewMulti(ratelimit.MultiOption{
		Minute: 5,
		Second: 3,
	})

	assert := assert.New(t)

	// Succeed.
	assert.True(rl.Allow())
	assert.Equal(4, rl.Remaining())

	// Call within the same period fails.
	assert.True(rl.Allow())
	assert.Equal(3, rl.Remaining())

	// Call within the same period fails.
	assert.True(rl.Allow())
	assert.Equal(2, rl.Remaining())

	// Call within the same period fails.
	assert.False(rl.Allow())
	assert.Equal(2, rl.Remaining())

	// Call after the next period succeeds.
	rl.SetNow(func() time.Time {
		return time.Now().Add(time.Second)
	})

	assert.True(rl.Allow())
	assert.Equal(1, rl.Remaining())

	assert.True(rl.Allow())
	assert.Equal(0, rl.Remaining())

	assert.False(rl.Allow())

	rl.SetNow(func() time.Time {
		return time.Now().Add(time.Minute)
	})

	assert.True(rl.Allow())
	assert.Equal(4, rl.Remaining())
}
