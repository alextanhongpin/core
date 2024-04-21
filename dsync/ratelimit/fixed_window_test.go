package ratelimit_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
)

func TestFixedWindow(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)

	rl := ratelimit.NewFixedWindow(client, &ratelimit.FixedWindowOption{
		Limit:  5,
		Period: 1 * time.Second,
	})

	key := t.Name()
	periods := []time.Duration{
		0,
		1 * time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		4 * time.Millisecond,

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

	now := time.Now().Truncate(time.Second)
	for _, p := range periods {
		rl.Now = func() time.Time {
			return now.Add(p)
		}

		allow := p < 5*time.Millisecond || p >= 1*time.Second
		result, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		if want, got := allow, result.Allow; want != got {
			t.Fatalf("allow: want %t, got %t", want, got)
		}

		t.Log(now.Add(p).Sub(now), result.Allow, result.Remaining, result.RetryIn(), result.ResetIn())
	}
}

func TestFixedWindow_Interval(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)
	rl := ratelimit.NewFixedWindow(client, &ratelimit.FixedWindowOption{
		Limit:  5,
		Period: 1 * time.Second,
	})

	key := t.Name()

	now := time.Now().Truncate(time.Second).Add(time.Second)
	time.Sleep(now.Sub(time.Now()))

	var count int
	for i := 0; i < 1000; i++ {
		rl.Now = func() time.Time {
			p := time.Duration(i) * time.Millisecond
			return now.Add(p)
		}

		result, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		if result.Allow {
			count++
		}
	}
	if want, got := 5, count; want != got {
		t.Fatalf("want %d, got %d", want, got)
	}
}

func TestFixedWindow_AllowN(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)
	rl := ratelimit.NewFixedWindow(client, &ratelimit.FixedWindowOption{
		Limit:  5,
		Period: 1 * time.Second,
	})

	key := t.Name()

	now := time.Now().Truncate(time.Second).Add(time.Second)
	time.Sleep(now.Sub(time.Now()))

	var count int
	for i := 0; i < 1000; i++ {
		rl.Now = func() time.Time {
			p := time.Duration(i) * time.Millisecond
			return now.Add(p)
		}

		result, err := rl.AllowN(ctx, key, 5)
		if err != nil {
			t.Fatal(err)
		}
		if result.Allow {
			count++
		}
	}
	if want, got := 1, count; want != got {
		t.Fatalf("want %d, got %d", want, got)
	}
}

func TestFixedWindow_Expiry(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)

	t.Run("Allow", func(t *testing.T) {
		t.Cleanup(func() {
			client.FlushAll(ctx)
		})

		rl := ratelimit.NewFixedWindow(client, &ratelimit.FixedWindowOption{
			Limit:  5,
			Period: 10 * time.Second,
		})

		key := t.Name()
		ts := time.Now().Truncate(10 * time.Second).Format(time.RFC3339Nano)
		redisKey := fmt.Sprintf("%s:ratelimit:fixed_window:%s", key, ts)

		_, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		ttl := client.TTL(ctx, redisKey).Val()
		if ttl < 9*time.Second || ttl > 10*time.Second {
			t.Fatalf("ttl: want ~10s, got %s", ttl)
		}
	})

	t.Run("AllowN", func(t *testing.T) {
		t.Cleanup(func() {
			client.FlushAll(ctx)
		})

		rl := ratelimit.NewFixedWindow(client, &ratelimit.FixedWindowOption{
			Limit:  5,
			Period: 10 * time.Second,
		})

		key := t.Name()
		ts := time.Now().Truncate(10 * time.Second).Format(time.RFC3339Nano)
		redisKey := fmt.Sprintf("%s:ratelimit:fixed_window:%s", key, ts)

		_, err := rl.AllowN(ctx, key, 5)
		if err != nil {
			t.Fatal(err)
		}

		ttl := client.TTL(ctx, redisKey).Val()
		if ttl < 9*time.Second || ttl > 10*time.Second {
			t.Fatalf("ttl: want ~10s, got %s", ttl)
		}
	})
}
