package throttle_test

import (
	"context"
	"github.com/alextanhongpin/core/sync/throttle"
	"math"
	"testing"
	"time"
)

func TestConfigDefaultsValidationAndOwnership(t *testing.T) {
	cfg := throttle.Config{Limit: 1}
	th, err := throttle.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.BacklogTimeout = time.Hour
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		th.Do(context.Background(), func(context.Context) error { close(started); <-release; return nil })
	}()
	<-started
	err = th.Do(t.Context(), func(context.Context) error { t.Error("exceeded concurrency"); return nil })
	close(release)
	<-done
	if err != throttle.ErrCapacityExceeded {
		t.Fatal(err)
	}
	for _, cfg := range []throttle.Config{{Limit: -1}, {BacklogLimit: -1}, {BacklogTimeout: -time.Second}, {Limit: 1, BacklogLimit: math.MaxInt}} {
		if th, err := throttle.New(cfg); err == nil || th != nil {
			t.Fatalf("accepted invalid configuration: %+v", cfg)
		}
	}
	if cfg := (throttle.Config{}).WithDefaults(); cfg.Limit != 1000 || cfg.BacklogLimit != 0 || cfg.BacklogTimeout != 0 {
		t.Fatal(cfg)
	}
}
