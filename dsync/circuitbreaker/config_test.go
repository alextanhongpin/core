package circuitbreaker_test

import (
	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"testing"
	"time"
)

func TestValueConfigValidationAndOwnership(t *testing.T) {
	client := newClient(t)
	cfg := circuitbreaker.Config{FailureThreshold: 1}
	cb, err := circuitbreaker.New(client, cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.FailureThreshold = 100
	cb.Do(t.Context(), t.Name(), func() error { return assertFailure{} })
	status, err := cb.Status(t.Context(), t.Name())
	if err != nil || status != circuitbreaker.Opened {
		t.Fatalf("%v %v", status, err)
	}
	for _, cfg := range []circuitbreaker.Config{{FailureThreshold: -1}, {OpenTimeout: time.Nanosecond}, {SuccessPeriod: -time.Second}} {
		if cb, err := circuitbreaker.New(client, cfg); err == nil || cb != nil {
			t.Fatal("invalid config accepted")
		}
	}
	if _, err := circuitbreaker.New(nil, circuitbreaker.Config{}); err == nil {
		t.Fatal("nil client accepted")
	}
}

type assertFailure struct{}

func (assertFailure) Error() string { return "failure" }
