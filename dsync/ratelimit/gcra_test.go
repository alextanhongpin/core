package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestGCRA_Limit(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)

	is := assert.New(t)
	// 5 requests per second.
	rl := ratelimit.NewGCRA(client, 5, time.Second, 0)

	key := t.Name()
	var total int
	for range 10 {
		time.Sleep(100 * time.Millisecond)
		r, err := rl.Limit(ctx, key)
		is.NoError(err)
		if r.Allow {
			total++
		}
		t.Logf("allow=%t remaining=%d reset_after=%s retry_after=%s\n", r.Allow, r.Remaining, r.ResetAfter, r.RetryAfter)
	}
	is.Equal(5, total)
}

func TestGCRA_LimitN(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)

	is := assert.New(t)
	rl := ratelimit.NewGCRA(client, 5, time.Second, 3)

	key := t.Name()
	var total int
	for range 10 {
		time.Sleep(100 * time.Millisecond)
		r, err := rl.LimitN(ctx, key, 2)
		is.NoError(err)
		if r.Allow {
			total++
		}
		t.Logf("allow=%t remaining=%d reset_after=%s retry_after=%s\n", r.Allow, r.Remaining, r.ResetAfter, r.RetryAfter)
	}

	is.NotZero(total)
}

func TestGCRA_withBurst(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)

	is := assert.New(t)
	// 5 request per second, each request takes 200ms.
	rl := ratelimit.NewGCRA(client, 5, time.Second, 1)

	var total int
	key := t.Name()
	for range 10 {
		time.Sleep(100 * time.Millisecond)
		r, err := rl.Limit(ctx, key)
		is.NoError(err)
		if r.Allow {
			total++
		}
		t.Logf("allow=%t remaining=%d reset_after=%s retry_after=%s\n", r.Allow, r.Remaining, r.ResetAfter, r.RetryAfter)
	}
	is.Equal(6, total)
}
