package ratelimit

import (
	"testing"
	"time"
)

func TestGCRARejectsBurstThatOverflowsTimestamp(t *testing.T) {
	cfg := Config{Limit: 1, Period: time.Hour, Burst: 2_500_000}
	r, err := NewGCRA(cfg)
	if err != nil {
		return
	}
	// Accepted configuration must behave: a full batch then another must be denied.
	if !r.AllowN("k", cfg.Burst+1) {
		t.Fatal("full batch denied")
	}
	if r.Allow("k") {
		t.Fatal("timestamp overflow let an exhausted key through")
	}
}
