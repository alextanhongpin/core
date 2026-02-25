package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestSlidingWindow(t *testing.T) {
	var (
		limit  = 5
		period = time.Second
		n      = 15
	)
	rl, err := ratelimit.NewSlidingWindow(limit, period)
	if err != nil {
		t.Fatal(err)
	}

	var count int

	now := time.Now()
	for range 2 * n {
		rl.Now = func() time.Time {
			return now
		}
		res := rl.Limit(t.Name())
		if res.Allow {
			count++
		}
		t.Log(res.String())
		now = now.Add(period / time.Duration(n))
	}

	assert.Equal(t, 11, count)

	is := assert.New(t)
	is.Equal(1, rl.Size())

	rl.Now = func() time.Time {
		return now.Add(time.Second)
	}
	rl.Clear()
	is.Zero(rl.Size())
}
