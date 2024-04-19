package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

func TestTokenBucketFullRange(t *testing.T) {
	rl := ratelimit.NewTokenBucket(5, time.Second, 0)

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
		dryRun := rl.AllowAt(now.Add(p), 1)
		allow := p.Milliseconds()%2 == 0

		result := rl.Allow()
		if result.Allow != allow || dryRun != allow {
			t.Fatalf("doesn't allow: %v", p)
		}

		t.Log(now.Add(p).Sub(now), result.Allow, result.Remaining, result.RetryAt.Sub(now), result.ResetAt.Sub(now))
	}
}

func TestTokenBucketPartial(t *testing.T) {
	rl := ratelimit.NewTokenBucket(5, time.Second, 0)

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

		dryRun := rl.AllowAt(now.Add(p), 1)
		allow := p.Milliseconds()%2 == 0

		result := rl.Allow()
		if result.Allow != allow || dryRun != allow {
			t.Fatalf("doesn't allow: %v", p)
		}

		t.Log(now.Add(p).Sub(now), result.Allow, result.Remaining, result.RetryAt.Sub(now), result.ResetAt.Sub(now))
	}
}

func TestTokenBucketBurst(t *testing.T) {
	rl := ratelimit.NewTokenBucket(5, time.Second, 1)

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
		dryRun := rl.AllowAt(now.Add(p), 1)
		allow := p.Milliseconds()%2 == 0

		result := rl.Allow()
		if result.Allow != allow || dryRun != allow {
			t.Fatalf("doesn't allow: %v", p)
		}

		t.Log(now.Add(p).Sub(now), result.Allow, result.Remaining, result.RetryAt.Sub(now), result.ResetAt.Sub(now))
	}
}

func TestTokenBucketBurstPartial(t *testing.T) {
	rl := ratelimit.NewTokenBucket(5, time.Second, 1)

	periods := []time.Duration{
		0,
		10 * time.Millisecond,

		// Bursts at the end
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
		dryRun := rl.AllowAt(now.Add(p), 1)
		allow := p.Milliseconds()%2 == 0

		result := rl.Allow()
		if result.Allow != allow || dryRun != allow {
			t.Fatalf("doesn't allow: %v", p)
		}

		t.Log(now.Add(p).Sub(now), result.Allow, result.Remaining, result.RetryAt.Sub(now), result.ResetAt.Sub(now))
	}
}

func TestTokenBucketMultipleBurst(t *testing.T) {
	rl := ratelimit.NewTokenBucket(5, time.Second, 5)

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
		dryRun := rl.AllowAt(now.Add(p), 1)
		allow := true

		result := rl.Allow()
		if result.Allow != allow || dryRun != allow {
			t.Fatalf("doesn't allow: %v", p)
		}

		t.Log(now.Add(p).Sub(now), result.Allow, result.Remaining, result.RetryAt.Sub(now), result.ResetAt.Sub(now))
	}
}

func TestTokenBucketAllowN(t *testing.T) {
	rl := ratelimit.NewTokenBucket(5, time.Second, 0)

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

		dryRun := rl.AllowAt(now.Add(p), 5)
		allow := p == 0 || p == time.Second
		result := rl.AllowN(5)
		if result.Allow != allow || dryRun != allow {
			t.Fatalf("doesn't allow: %v", p)
		}

		t.Log(now.Add(p).Sub(now), result.Allow, result.Remaining, result.RetryAt.Sub(now), result.ResetAt.Sub(now))
	}
}
