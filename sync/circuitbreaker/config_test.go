package circuitbreaker_test

import (
	"errors"
	"github.com/alextanhongpin/core/sync/circuitbreaker"
	"testing"
	"time"
)

func TestConfigDefaultsValidationAndOwnership(t *testing.T) {
	cfg := circuitbreaker.Config{FailureThreshold: 2}
	cb, err := circuitbreaker.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.FailureThreshold = 100
	failure := errors.New("failure")
	cb.Do(func() error { return failure })
	cb.Do(func() error { return failure })
	if cb.Status() != circuitbreaker.Opened {
		t.Fatal("configuration mutation changed threshold")
	}
	for _, cfg := range []circuitbreaker.Config{{FailureThreshold: -1}, {SuccessThreshold: -1}, {FailurePeriod: -time.Second}, {SuccessPeriod: -time.Second}, {OpenTimeout: -time.Second}} {
		if cb, err := circuitbreaker.New(cfg); err == nil || cb != nil {
			t.Fatalf("accepted invalid configuration: %+v", cfg)
		}
	}
	if cfg := (circuitbreaker.Config{}).WithDefaults(); cfg.FailureThreshold != 100 || cfg.OpenTimeout != time.Minute {
		t.Fatal(cfg)
	}
}
