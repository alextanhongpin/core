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
		allow, err := rl.Allow(ctx, key)
		is.Nil(err)
		if allow {
			count++
		}

		remaining, err := rl.Remaining(ctx, key)
		is.Nil(err)

		resetAfter, err := rl.ResetAfter(ctx, key)
		is.Nil(err)

		t.Log(allow, remaining, resetAfter)
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
		is.Nil(err)
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
		is.Nil(err)
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
		allow, err := rl.Allow(ctx, key)
		is := assert.New(t)
		is.Nil(err)
		is.True(allow)

		resetAfter, err := rl.ResetAfter(ctx, key)
		is.Nil(err)
		is.Equal(time.Duration(0), resetAfter)
	})

	t.Run("AllowN", func(t *testing.T) {
		t.Cleanup(func() {
			client.FlushAll(ctx)
		})

		rl := ratelimit.NewFixedWindow(client, 5, 10*time.Second)
		key := t.Name()
		allow, err := rl.AllowN(ctx, key, 5)

		is := assert.New(t)
		is.Nil(err)
		is.True(allow)

		resetAfter, err := rl.ResetAfter(ctx, key)
		is.Nil(err)
		is.LessOrEqual(10*time.Second, resetAfter)
	})
}
