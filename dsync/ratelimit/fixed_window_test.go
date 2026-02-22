package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestFixedWindow(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)
	rl := ratelimit.NewFixedWindow(client, 5, time.Second)
	key := t.Name()
	is := assert.New(t)
	var count int

	for range 10 {
		r, err := rl.Limit(ctx, key)
		is.NoError(err)
		if r.Allow {
			count++
		}
		t.Logf("allow=%t remaining=%d reset_after=%s retry_after=%s\n", r.Allow, r.Remaining, r.ResetAfter, r.RetryAfter)
	}
	is.Equal(5, count)
}

func TestFixedWindow_Interval(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)
	rl := ratelimit.NewFixedWindow(client, 5, time.Second)

	is := assert.New(t)
	key := t.Name()
	var count int
	for i := 0; i < 1000; i++ {
		allow, err := rl.Allow(ctx, key)
		is.NoError(err)
		if allow {
			count++
		}
	}
	is.Equal(5, count)
}

func TestFixedWindow_AllowN(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)
	rl := ratelimit.NewFixedWindow(client, 5, time.Second)

	key := t.Name()

	is := assert.New(t)
	var count int
	for i := 0; i < 1000; i++ {
		allow, err := rl.AllowN(ctx, key, 5)
		is.NoError(err)
		if allow {
			count++
		}
	}
	is.Equal(1, count)
}

func TestFixedWindow_Expiry(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)

	t.Run("Allow", func(t *testing.T) {
		t.Cleanup(func() {
			client.FlushAll(ctx)
		})

		rl := ratelimit.NewFixedWindow(client, 5, 10*time.Second)

		key := t.Name()
		result, err := rl.Limit(ctx, key)
		is := assert.New(t)
		is.NoError(err)
		is.True(result.Allow)

		is.InDelta((10 * time.Second).Seconds(), result.ResetAfter.Seconds(), (100 * time.Millisecond).Seconds())
		is.Equal(time.Duration(0), result.RetryAfter)
	})

	t.Run("AllowN", func(t *testing.T) {
		t.Cleanup(func() {
			client.FlushAll(ctx)
		})

		rl := ratelimit.NewFixedWindow(client, 5, 10*time.Second)
		key := t.Name()
		result, err := rl.LimitN(ctx, key, 5)

		is := assert.New(t)
		is.NoError(err)
		is.True(result.Allow)
		is.LessOrEqual(result.ResetAfter, 10*time.Second)
	})
}
