package retry_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/backoff"
	"github.com/alextanhongpin/core/sync/retry"
)

type customBackoff struct{}

func (customBackoff) At(int) time.Duration { return 0 }

func TestBackoffCompatibility(t *testing.T) {
	var constant *backoff.Constant = retry.NewConstantBackoff(time.Second)
	var linear *backoff.Linear = retry.NewLinearBackoff(time.Second)
	var exponential *backoff.Exponential = retry.NewExponentialBackoff(time.Second, time.Minute)
	policies := []retry.Backoff{
		constant, linear, exponential,
		&retry.ConstantBackoff{Period: time.Second},
		&retry.LinearBackoff{Period: time.Second},
		&retry.ExponentialBackoff{Base: time.Second, Cap: time.Minute},
		customBackoff{}, backoff.NewConstant(0),
	}
	for _, policy := range policies {
		var shared backoff.Backoff = policy
		if _, err := retry.New(retry.Config{Backoff: shared}); err != nil {
			t.Fatal(err)
		}
	}
	if constant.At(1) != time.Second || linear.At(2) != 2*time.Second {
		t.Fatal("legacy constructors changed")
	}
}
