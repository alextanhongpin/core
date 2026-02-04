package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestGCRA(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)

	is := assert.New(t)
	// 5 requests per second.
	rl := ratelimit.NewGCRA(client, 5, time.Second, 0)

	key := t.Name()
	var total int
	for range 10 {
		time.Sleep(100 * time.Millisecond)
		ok, err := rl.Allow(ctx, key)
		is.NoError(err)
		if ok {
			total++
		}
	}
	is.Equal(5, total)
}

func TestGCRAAllowN(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)

	is := assert.New(t)
	rl := ratelimit.NewGCRA(client, 5, time.Second, 0)

	key := t.Name()
	var total int
	for range 10 {
		time.Sleep(100 * time.Millisecond)
		ok, err := rl.AllowN(ctx, key, 5)
		is.NoError(err)
		if ok {
			total++
		}
	}

	is.Equal(1, total)
}

func TestGCRABurst(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)

	is := assert.New(t)
	// 5 request per second, each request takes 200ms.
	rl := ratelimit.NewGCRA(client, 5, time.Second, 1)

	var total int
	key := t.Name()
	for range 10 {
		time.Sleep(100 * time.Millisecond)
		ok, err := rl.Allow(ctx, key)
		is.NoError(err)
		if ok {
			total++
		}
	}
	is.Equal(6, total)
}
