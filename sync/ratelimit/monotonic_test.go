package ratelimit

import (
	"testing"
	"time"
)

func TestZeroQuantityDoesNotCreateEntry(t *testing.T) {
	g := MustNewGCRA(Config{Limit: 10, Period: time.Second})
	f := MustNewFixedWindow(Config{Limit: 10, Period: time.Second})
	for name, r := range map[string]interface {
		AllowN(string, int) bool
		Size() int
	}{"gcra": g, "fixed": f} {
		if !r.AllowN("unseen", 0) {
			t.Fatalf("%s: zero-quantity inspection denied", name)
		}
		if r.Size() != 0 {
			t.Fatalf("%s: zero quantity created an entry", name)
		}
		r.AllowN("seen", 1)
		if r.Size() != 1 {
			t.Fatalf("%s: size = %d", name, r.Size())
		}
	}
}

func TestFixedWindowFollowsClockAndSurvivesWallJump(t *testing.T) {
	r := MustNewFixedWindow(Config{Limit: 1, Period: time.Second})
	now := r.base // fake clock starting exactly at base: elapsed == 0
	r.nowFunc = func() time.Time { return now }
	if !r.Allow("k") {
		t.Fatal("first denied")
	}
	if r.Allow("k") {
		t.Fatal("second admitted at elapsed 0 (zero-value state mistaken for fresh window)")
	}
	now = now.Add(time.Second)
	if !r.Allow("k") {
		t.Fatal("new window denied")
	}
}

func TestGCRAUsesInjectedClock(t *testing.T) {
	r := MustNewGCRA(Config{Limit: 1, Period: time.Second})
	now := r.base
	r.nowFunc = func() time.Time { return now }
	if !r.Allow("k") || r.Allow("k") {
		t.Fatal("unexpected admission at t=0")
	}
	now = now.Add(time.Second)
	if !r.Allow("k") {
		t.Fatal("denied after interval")
	}
}
