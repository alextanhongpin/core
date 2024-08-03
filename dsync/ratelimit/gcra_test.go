package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/ratelimit"
)

func TestGCRA(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)

	rl := ratelimit.NewGCRA(client, &ratelimit.GCRAOption{
		Limit:  5,
		Period: 1 * time.Second,
	})

	key := t.Name()
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

	now := time.Now().Truncate(time.Second)
	for _, p := range periods {
		rl.Now = func() time.Time {
			return now.Add(p)
		}

		allow := p.Milliseconds()%2 == 0
		result, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%6s, allow: %t (%d/%d), retry_in: %s, reset_in: %s\n", now.Add(p).Sub(now), result.Allow, result.Remaining, result.Limit, result.RetryIn(), result.ResetIn())
		if want, got := allow, result.Allow; want != got {
			t.Fatalf("allow: want %t, got %t", want, got)
		}
	}
}

func TestGCRAAllowN(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)

	rl := ratelimit.NewGCRA(client, &ratelimit.GCRAOption{
		Limit:  5,
		Period: 1 * time.Second,
	})

	key := t.Name()
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

	now := time.Now().Truncate(time.Second)
	for _, p := range periods {
		rl.Now = func() time.Time {
			return now.Add(p)
		}

		allow := p == 0 || p == time.Second
		result, err := rl.AllowN(ctx, key, 5)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%6s, allow: %t (%d/%d), retry_in: %s, reset_in: %s\n", now.Add(p).Sub(now), result.Allow, result.Remaining, result.Limit, result.RetryIn(), result.ResetIn())
		if want, got := allow, result.Allow; want != got {
			t.Fatalf("allow: want %t, got %t", want, got)
		}
	}
}

func TestGCRAPartial(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)

	rl := ratelimit.NewGCRA(client, &ratelimit.GCRAOption{
		Limit:  5,
		Period: 1 * time.Second,
	})

	key := t.Name()
	periods := []time.Duration{
		0,
		// Bursts at the end
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

		allow := p.Milliseconds()%2 == 0
		result, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%6s, allow: %t (%d/%d), retry_in: %s, reset_in: %s\n", now.Add(p).Sub(now), result.Allow, result.Remaining, result.Limit, result.RetryIn(), result.ResetIn())
		if want, got := allow, result.Allow; want != got {
			t.Fatalf("allow: want %t, got %t", want, got)
		}
	}
}

func TestGCRABurst(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)
	rl := ratelimit.NewGCRA(client, &ratelimit.GCRAOption{
		Limit:  5,
		Period: 1 * time.Second,
		Burst:  1,
	})

	key := t.Name()
	periods := []time.Duration{
		0,
		10 * time.Millisecond,
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
		1010 * time.Millisecond,
	}

	now := time.Now().Truncate(time.Second)
	for _, p := range periods {
		rl.Now = func() time.Time {
			return now.Add(p)
		}

		if p > time.Second {
			if err := client.Del(ctx, key).Err(); err != nil {
				t.Fatal(err)
			}
			t.Log("deleted key")
		}

		allow := p.Milliseconds()%2 == 0
		result, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%6s, allow: %t (%d/%d), retry_in: %s, reset_in: %s\n", now.Add(p).Sub(now), result.Allow, result.Remaining, result.Limit, result.RetryIn(), result.ResetIn())
		if want, got := allow, result.Allow; want != got {
			t.Fatalf("allow: want %t, got %t", want, got)
		}
	}
}

func TestGCRABurstPartial(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)
	rl := ratelimit.NewGCRA(client, &ratelimit.GCRAOption{
		Limit:  5,
		Period: 1 * time.Second,
		Burst:  1,
	})

	key := t.Name()
	periods := []time.Duration{
		0,
		10 * time.Millisecond,

		// Bursts at the end
		600 * time.Millisecond,
		602 * time.Millisecond,

		799 * time.Millisecond,
		800 * time.Millisecond,
		801 * time.Millisecond,

		999 * time.Millisecond,
		1000 * time.Millisecond,
		1010 * time.Millisecond,
	}

	now := time.Now().Truncate(time.Second)
	for _, p := range periods {
		rl.Now = func() time.Time {
			return now.Add(p)
		}
		if p > time.Second {
			if err := client.Del(ctx, key).Err(); err != nil {
				t.Fatal(err)
			}
			t.Log("deleted key")
		}

		allow := p.Milliseconds()%2 == 0
		result, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%6s, allow: %t (%d/%d), retry_in: %s, reset_in: %s\n", now.Add(p).Sub(now), result.Allow, result.Remaining, result.Limit, result.RetryIn(), result.ResetIn())
		if want, got := allow, result.Allow; want != got {
			t.Fatalf("allow: want %t, got %t", want, got)
		}
	}
}

func TestGCRABurstTotal(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)
	rl := ratelimit.NewGCRA(client, &ratelimit.GCRAOption{
		Limit:  5,
		Period: 1 * time.Second,
		Burst:  1,
	})

	key := t.Name()

	now := time.Now().Truncate(time.Second)
	var count int
	for i := 0; i < 1000; i++ {
		p := time.Duration(i) * time.Millisecond
		rl.Now = func() time.Time {
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
	if want, got := 6, count; want != got {
		t.Fatalf("want %d, got %d", want, got)
	}
}
