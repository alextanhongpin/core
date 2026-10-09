package ratelimit_test

import (
	"github.com/alextanhongpin/core/sync/ratelimit"
	"math"
	"testing"
	"time"
)

func TestValueConfigDefaultsValidationAndOwnership(t *testing.T) {
	cfg := ratelimit.Config{Limit: 1, Period: time.Hour}
	fw, err := ratelimit.NewFixedWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}
	gcra, err := ratelimit.NewGCRA(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Limit = 100
	if !fw.Allow("k") || fw.Allow("k") || !gcra.Allow("k") || gcra.Allow("k") {
		t.Fatal("configuration mutation changed limits")
	}
	for _, cfg := range []ratelimit.Config{{Limit: -1}, {Period: -time.Second}, {Burst: -1}} {
		if r, err := ratelimit.NewFixedWindow(cfg); err == nil || r != nil {
			t.Fatal("invalid fixed window config accepted")
		}
		if r, err := ratelimit.NewGCRA(cfg); err == nil || r != nil {
			t.Fatal("invalid GCRA config accepted")
		}
	}
	for _, cfg := range []ratelimit.Config{{Limit: 2, Period: time.Nanosecond}, {Limit: 1, Period: time.Second, Burst: math.MaxInt}} {
		if r, err := ratelimit.NewGCRA(cfg); err == nil || r != nil {
			t.Fatal("unsafe GCRA arithmetic accepted")
		}
	}
	if _, err := ratelimit.NewFixedWindow(ratelimit.Config{}); err != nil {
		t.Fatal(err)
	}
	if _, err := ratelimit.NewGCRA(ratelimit.Config{}); err != nil {
		t.Fatal(err)
	}
}
