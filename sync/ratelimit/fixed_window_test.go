package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestFixedWindow(t *testing.T) {
	var (
		key    = t.Name()
		limit  = 5
		n      = 20
		period = time.Second
	)
	rl, err := ratelimit.NewFixedWindow(limit, period)
	if err != nil {
		t.Fatal(err)
	}

	var count int
	now := time.Now()
	for range 2 * n {
		rl.Now = func() time.Time {
			return now
		}
		res := rl.Limit(key)
		if res.Allow {
			count++
		}
		now = now.Add(period / time.Duration(n))
		t.Log(res.String())
	}

	is := assert.New(t)
	is.Equal(limit*2, count)
	is.Equal(1, rl.Size())

	rl.Now = func() time.Time {
		return now.Add(time.Second)
	}
	rl.Clear()
	is.Zero(rl.Size())
}
