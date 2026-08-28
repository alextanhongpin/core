package throttle

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alextanhongpin/evaltest"
)

// ThrottleInput describes a scenario for testing the Throttler.Do method.
type ThrottleInput struct {
	Limit            int `yaml:"limit"`
	BacklogLimit     int `yaml:"backlog_limit"`
	BacklogTimeoutMs int `yaml:"backlog_timeout_ms"`
	Calls            int `yaml:"calls"`
}

// ThrottleOutput captures the error from the last Do call.
type ThrottleOutput struct {
	LastErr string `json:"last_err"`
}

// TestThrottleDo uses evaltest to run data-driven scenarios for Throttler.Do.
// Each case creates a fresh Throttler with the given limits and performs
// `Calls` consecutive Do invocations. The test asserts the error from the
// final call.
func TestThrottleDo(t *testing.T) {
	evaltest.Run(t, func(t *testing.T, ctx context.Context, input ThrottleInput) (ThrottleOutput, error) {
		cfg := Config{
			Limit:          input.Limit,
			BacklogLimit:   input.BacklogLimit,
			BacklogTimeout: time.Duration(input.BacklogTimeoutMs) * time.Millisecond,
		}
		th, err := New(&cfg)
		if err != nil {
			return ThrottleOutput{}, err
		}

		var lastErr error
		for i := 0; i < input.Calls; i++ {
			// Use a background context; the internal timeout is driven by cfg.BacklogTimeout.
			// For deterministic tests we keep BacklogTimeout large enough to avoid spurious timeouts.
			lastErr = th.Do(context.Background(), func(context.Context) error { return nil })
		}

		return ThrottleOutput{
			LastErr: fmt.Sprintf("%v", lastErr),
		}, nil
	})
}
