package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

func TestGCRAFullRange(t *testing.T) {
	rl := ratelimit.NewGCRA(5, time.Second, 0)

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
		want := p.Milliseconds()%2 == 0
		got := rl.Allow()
		if want != got {
			t.Fatalf("doesn't allow: %v", p)
		}

		t.Log(now.Add(p).Sub(now), got, rl.RetryAfter())
	}
}

func TestGCRAPartial(t *testing.T) {
	rl := ratelimit.NewGCRA(5, time.Second, 0)

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

		want := p.Milliseconds()%2 == 0
		got := rl.Allow()
		if want != got {
			t.Fatalf("doesn't allow: %v", p)
		}

		t.Log(now.Add(p).Sub(now), got, rl.RetryAfter())
	}
}

func TestGCRABurst(t *testing.T) {
	rl := ratelimit.NewGCRA(5, time.Second, 1)

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
		1001 * time.Millisecond,
	}

	now := time.Now().Truncate(time.Second)
	for _, p := range periods {
		rl.Now = func() time.Time {
			return now.Add(p)
		}
		want := p.Milliseconds()%2 == 0
		got := rl.Allow()
		if want != got {
			t.Fatalf("want %t, got %t: %v", want, got, p)
		}

		t.Log(now.Add(p).Sub(now), got, rl.RetryAfter())
	}
}

func TestGCRABurstPartial(t *testing.T) {
	rl := ratelimit.NewGCRA(5, time.Second, 1)

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
		1001 * time.Millisecond,
	}

	now := time.Now().Truncate(time.Second)
	var count int
	for _, p := range periods {
		rl.Now = func() time.Time {
			return now.Add(p)
		}
		want := p.Milliseconds()%2 == 0
		got := rl.Allow()
		if want != got {
			t.Fatalf("want %t, got %t, %v", want, got, p)
		}
		if got {
			count++
		}

		t.Log(now.Add(p).Sub(now), got, rl.RetryAfter())
	}
	if 6 != count {
		t.Fatalf("want %d, got %d", 6, count)
	}
}

func TestGCRAMultipleBurst(t *testing.T) {
	rl := ratelimit.NewGCRA(5, time.Second, 5)

	periods := []time.Duration{
		0,
		1 * time.Millisecond,
		3 * time.Millisecond,
		5 * time.Millisecond,
		200 * time.Millisecond,
		201 * time.Millisecond,
		399 * time.Millisecond,
		400 * time.Millisecond,
		600 * time.Millisecond,
		800 * time.Millisecond,
	}

	now := time.Now().Truncate(time.Second)
	for _, p := range periods {
		rl.Now = func() time.Time {
			return now.Add(p)
		}
		got := rl.Allow()
		if !got {
			t.Fatalf("doesn't allow: %v", p)
		}

		t.Log(now.Add(p).Sub(now), got, rl.RetryAfter())
	}
}

func TestGCRAAllowN(t *testing.T) {
	rl := ratelimit.NewGCRA(5, time.Second, 0)

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
		want := p.Milliseconds()%2 == 0
		got := rl.Allow()
		if want != got {
			t.Fatalf("doesn't allow: %v", p)
		}

		t.Log(now.Add(p).Sub(now), got, rl.RetryAfter())
	}
}

func TestGCRABurstTotal(t *testing.T) {
	rl := ratelimit.NewGCRA(5, time.Second, 1)

	now := time.Now().Truncate(time.Second)
	var delay []time.Duration
	var count int
	for i := 0; i < 1000; i++ {
		p := time.Duration(i) * time.Millisecond
		rl.Now = func() time.Time {
			return now.Add(p)
		}

		if rl.Allow() {
			delay = append(delay, p)
			count++
		}
	}
	if want, got := 6, count; want != got {
		t.Fatalf("want %d, got %d", want, got)
	}
	t.Log(delay)
}
