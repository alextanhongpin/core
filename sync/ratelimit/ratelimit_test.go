package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	rl := ratelimit.New("5/s", 5, time.Second)

	now := time.Now().Truncate(rl.Period())

	rl.SetNow(func() time.Time {
		return now
	})

	assert := assert.New(t)

	// Succeed.
	assert.True(rl.Allow())
	assert.Equal(4, rl.Remaining())

	// Call within the same period fails.
	assert.False(rl.Allow())
	assert.Equal(4, rl.Remaining())

	// Call after the next period succeeds.
	rl.SetNow(func() time.Time {
		return now.Add(rl.Period())
	})

	assert.True(rl.Allow())
	assert.Equal(3, rl.Remaining())
}
