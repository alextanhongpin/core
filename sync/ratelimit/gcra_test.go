package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestGCRA(t *testing.T) {
	var (
		limit  = 5
		period = time.Second
		burst  = 1
		n      = 20
	)
	rl, err := ratelimit.NewGCRA(limit, period, burst)
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
		now = now.Add(period / time.Duration(n))
		t.Log(res.String())
	}

	assert.Equal(t, 11, count)

	is := assert.New(t)
	is.Equal(1, rl.Size())

	rl.Now = func() time.Time {
		return now.Add(2 * time.Second)
	}
	rl.Clear()
	is.Zero(rl.Size())
}
