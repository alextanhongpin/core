package circuitbreaker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestCircuitBreaker_Do(t *testing.T) {
	// Create a new circuit breaker with default options.
	opt := circuitbreaker.NewOption()

	now := time.Now()
	opt.Now = func() time.Time {
		return now
	}

	// Create a new circuit breaker.
	cb := circuitbreaker.New(opt)

	var statuses []circuitbreaker.Status
	cb.OnStateChanged = func(ctx context.Context, from, to circuitbreaker.Status) {
		statuses = append(statuses, from, to)
	}

	key := t.Name()

	run := func(t *testing.T, n int, wantErr, gotErr error) {
		t.Helper()

		is := assert.New(t)
		for i := 0; i < n; i++ {
			err := cb.Do(ctx, key, func() error {
				return wantErr
			})
			is.ErrorIs(err, gotErr)
		}
	}

	testStatus := func(t *testing.T, want circuitbreaker.Status) {
		t.Helper()

		is := assert.New(t)
		status, err := cb.Status(ctx, key)
		is.Nil(err)
		is.Equal(want, status)
	}

	testStatusChanged := func(t *testing.T, from, to circuitbreaker.Status) {
		t.Helper()

		is := assert.New(t)
		i := len(statuses) - 2
		is.Equal(from, statuses[i])
		is.Equal(to, statuses[i+1])
	}

	t.Run("initial status is closed", func(t *testing.T) {
		testStatus(t, circuitbreaker.Closed)
	})

	t.Run("status changed to open", func(t *testing.T) {
		var wantErr = errors.New("wantErr")
		run(t, opt.FailureThreshold, wantErr, wantErr)
		run(t, 1, wantErr, circuitbreaker.ErrBrokenCircuit)
		testStatus(t, circuitbreaker.Open)
		testStatusChanged(t, circuitbreaker.Closed, circuitbreaker.Open)
	})

	t.Run("status changed to half-open", func(t *testing.T) {
		now = now.Add(opt.BreakDuration)

		run(t, 1, nil, nil)
		testStatus(t, circuitbreaker.HalfOpen)
		testStatusChanged(t, circuitbreaker.Open, circuitbreaker.HalfOpen)
	})

	t.Run("status changed to closed", func(t *testing.T) {
		run(t, opt.SuccessThreshold, nil, nil)
		testStatus(t, circuitbreaker.Closed)
		testStatusChanged(t, circuitbreaker.HalfOpen, circuitbreaker.Closed)
	})
}
