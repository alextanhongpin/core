package circuitbreaker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
	"github.com/stretchr/testify/assert"
)

var (
	ctx     = context.Background()
	wantErr = errors.New("want error")
)

func TestCircuit(t *testing.T) {
	cb := circuitbreaker.New()
	cb.BreakDuration = 50 * time.Millisecond

	t.Run("initial", func(t *testing.T) {
		is := assert.New(t)
		is.Equal(circuitbreaker.Closed, cb.Status())
	})

	t.Run("opened", func(t *testing.T) {
		is := assert.New(t)

		for range cb.FailureThreshold {
			err := cb.Do(func() error {
				return wantErr
			})
			is.ErrorIs(err, wantErr)
		}
		err := cb.Do(func() error {
			return wantErr
		})
		is.ErrorIs(err, circuitbreaker.ErrBrokenCircuit)
		is.Equal(circuitbreaker.Open, cb.Status())
	})

	t.Run("half-opened", func(t *testing.T) {
		time.Sleep(cb.BreakDuration + 5*time.Millisecond)
		is := assert.New(t)
		is.Equal(circuitbreaker.HalfOpen, cb.Status())
	})

	t.Run("closed", func(t *testing.T) {
		is := assert.New(t)

		for range cb.SuccessThreshold {
			err := cb.Do(func() error {
				return nil
			})
			is.Nil(err)
		}

		is.Equal(circuitbreaker.Closed, cb.Status())
	})
}

func TestHalfOpenFail(t *testing.T) {
	cb := circuitbreaker.New()
	cb.BreakDuration = 50 * time.Millisecond

	is := assert.New(t)
	is.Equal(circuitbreaker.Closed, cb.Status())

	// Shift to closed state.
	for range cb.FailureThreshold {
		err := cb.Do(func() error {
			return wantErr
		})
		is.NotNil(err)
	}
	is.Equal(circuitbreaker.Open, cb.Status())

	time.Sleep(cb.BreakDuration + 5*time.Millisecond)
	is.Equal(circuitbreaker.HalfOpen, cb.Status())

	// Trigger failure in half-opened state.
	err := cb.Do(func() error {
		return wantErr
	})
	is.ErrorIs(err, wantErr)
	is.Equal(circuitbreaker.Open, cb.Status())
}

func TestSlowCount(t *testing.T) {
	cb := circuitbreaker.New()
	cb.SlowCallCount = func(time.Duration) int {
		// A single slow call will trigger the circuitbreaker to
		// open.
		return cb.FailureThreshold
	}
	err := cb.Do(func() error {
		// No error, but the slow call count is incremented.
		return nil
	})
	is := assert.New(t)
	is.Nil(err)
	is.Equal(circuitbreaker.Open, cb.Status())
}
