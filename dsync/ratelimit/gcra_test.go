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

	rl := ratelimit.NewGCRA(client, 5, time.Second, 0)

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

		valid := p.Milliseconds()%2 == 0
		allow, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		if valid != allow {
			t.Fatalf("doesn't allow: %v", p)
		}
	}
}

func TestGCRAAllowN(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)

	rl := ratelimit.NewGCRA(client, 5, time.Second, 0)

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

		valid := p == 0 || p == time.Second
		allow, err := rl.AllowN(ctx, key, 5)
		if err != nil {
			t.Fatal(err)
		}
		if valid != allow {
			t.Fatalf("doesn't allow: %v", p)
		}
	}
}

func TestGCRAPartial(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)

	rl := ratelimit.NewGCRA(client, 5, time.Second, 0)

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

		valid := p.Milliseconds()%2 == 0
		allow, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		if valid != allow {
			t.Fatalf("doesn't allow: %v", p)
		}
	}
}

func TestGCRABurst(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)
	rl := ratelimit.NewGCRA(client, 5, time.Second, 1)

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

		valid := p.Milliseconds()%2 == 0
		allow, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		if valid != allow {
			t.Fatalf("doesn't allow: %v", p)
		}
	}
}

func TestGCRABurstPartial(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)
	rl := ratelimit.NewGCRA(client, 5, time.Second, 1)

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

		valid := p.Milliseconds()%2 == 0
		allow, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		if valid != allow {
			t.Fatalf("doesn't allow: %v", p)
		}
	}
}

func TestGCRABurstTotal(t *testing.T) {
	ctx := context.Background()

	client := newClient(t)
	rl := ratelimit.NewGCRA(client, 5, time.Second, 1)
	key := t.Name()

	now := time.Now().Truncate(time.Second)
	var count int
	for i := 0; i < 1000; i++ {
		p := time.Duration(i) * time.Millisecond
		rl.Now = func() time.Time {
			return now.Add(p)
		}

		allow, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		if allow {
			count++
		}
	}
	if want, got := 6, count; want != got {
		t.Fatalf("want %d, got %d", want, got)
	}
}
