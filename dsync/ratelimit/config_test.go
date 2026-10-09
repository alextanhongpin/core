package ratelimit_test

import (
	"github.com/alextanhongpin/core/dsync/ratelimit"
	"testing"
	"time"
)

func TestValueConfig(t *testing.T) {
	client := newClient(t)
	cfg := ratelimit.Config{Limit: 1, Period: time.Hour}
	r, err := ratelimit.NewFixedWindow(client, cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Limit = 100
	r.Allow(t.Context(), t.Name())
	if ok, err := r.Allow(t.Context(), t.Name()); err != nil || ok {
		t.Fatal("caller reconfigured limiter", err)
	}
	for _, cfg := range []ratelimit.Config{{Limit: -1}, {Period: time.Nanosecond}, {Burst: -1}} {
		if _, err := ratelimit.NewFixedWindow(client, cfg); err == nil {
			t.Fatal("invalid config accepted")
		}
	}
	if _, err := ratelimit.NewGCRA(client, ratelimit.Config{Limit: 2, Period: time.Millisecond}); err == nil {
		t.Fatal("unsafe interval accepted")
	}
	if _, err := ratelimit.NewGCRA(nil, ratelimit.Config{}); err == nil {
		t.Fatal("nil client accepted")
	}
}
