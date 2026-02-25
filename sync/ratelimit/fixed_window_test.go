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
		n      = 15
		period = time.Second
	)
	rl, err := ratelimit.NewFixedWindow(limit, period)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	for range n {
		rl.Now = func() time.Time {
			return now
		}
		res := rl.Limit(key)
		t.Log(res.String())
		now = now.Add(period / time.Duration(n))
	}

	is := assert.New(t)
	is.Equal(1, rl.Size())

	rl.Now = func() time.Time {
		return now.Add(time.Second)
	}
	rl.Clear()
	is.Zero(rl.Size())
}
