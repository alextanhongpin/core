package ratelimit_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/redis/go-redis/v9"
)

func TestMain(m *testing.M) {
	stop := redistest.Init()
	code := m.Run()
	stop()
	os.Exit(code)
}

func TestLeakyBucket(t *testing.T) {
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	t.Cleanup(func() {
		client.FlushAll(ctx)
		client.Close()
	})

	rl := ratelimit.NewLeakyBucket(client, &ratelimit.LeakyBucketOption{
		Limit:  5,
		Period: 1 * time.Second,
		Burst:  0,
	})

	key := fmt.Sprintf("%s:key", t.Name())
	now := time.Now()
	periods := []time.Duration{
		0,
		199 * time.Millisecond,
		200 * time.Millisecond,
		201 * time.Millisecond,

		399 * time.Millisecond,
		400 * time.Millisecond,
		401 * time.Millisecond,

		599 * time.Millisecond,
		600 * time.Millisecond,
		601 * time.Millisecond,

		799 * time.Millisecond,
		800 * time.Millisecond,
		801 * time.Millisecond,

		999 * time.Millisecond,
		1000 * time.Millisecond,
		1001 * time.Millisecond,
	}

	for _, p := range periods {
		rl.Now = now.Add(p)

		allow := p.Milliseconds()%2 == 0

		result, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		if want, got := allow, result.Allow; want != got {
			t.Fatalf("allow: want %t, got %t", want, got)
		}

		t.Log(now.Add(p).Sub(now), result.Allow, result.Remaining)
	}
}

func TestLeakyBucketExpired(t *testing.T) {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	t.Cleanup(func() {
		client.FlushAll(ctx)
		client.Close()
	})

	rl := ratelimit.NewLeakyBucket(client, &ratelimit.LeakyBucketOption{
		Limit:  5,
		Period: 1 * time.Second,
		Burst:  0,
	})

	key := fmt.Sprintf("%s:key", t.Name())
	now := time.Now()
	periods := []time.Duration{
		0,
		600 * time.Millisecond,
		601 * time.Millisecond,

		799 * time.Millisecond,
		800 * time.Millisecond,
		801 * time.Millisecond,

		999 * time.Millisecond,
		1000 * time.Millisecond,
		1001 * time.Millisecond,
	}

	for _, p := range periods {
		rl.Now = now.Add(p)

		allow := p.Milliseconds()%2 == 0

		result, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		t.Log(now.Add(p).Sub(now), result.Allow, result.Remaining)
		if want, got := allow, result.Allow; want != got {
			t.Fatalf("allow: want %t, got %t", want, got)
		}
	}
}
