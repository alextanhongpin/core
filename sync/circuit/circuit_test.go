package circuit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/circuit"
	"github.com/stretchr/testify/assert"
)

var (
	ctx     = context.Background()
	wantErr = errors.New("want error")
)

func TestCircuit(t *testing.T) {
	opt := circuit.NewOption()
	opt.BreakDuration = 50 * time.Millisecond
	cb := circuit.New(opt)

	t.Run("initial", func(t *testing.T) {
		is := assert.New(t)
		is.Equal(circuit.Closed, cb.Status())
	})

	t.Run("opened", func(t *testing.T) {
		is := assert.New(t)

		for range opt.FailureThreshold {
			err := cb.Do(func() error {
				return wantErr
			})
			is.ErrorIs(err, wantErr)
		}
		err := cb.Do(func() error {
			return wantErr
		})
		is.ErrorIs(err, circuit.ErrBrokenCircuit)
		is.Equal(circuit.Open, cb.Status())
	})

	t.Run("half-opened", func(t *testing.T) {
		time.Sleep(opt.BreakDuration + 5*time.Millisecond)
		is := assert.New(t)
		is.Equal(circuit.HalfOpen, cb.Status())
	})

	t.Run("closed", func(t *testing.T) {
		is := assert.New(t)

		for range opt.SuccessThreshold {
			err := cb.Do(func() error {
				return nil
			})
			is.Nil(err)
		}

		is.Equal(circuit.Closed, cb.Status())
	})
}

func TestHalfOpenFail(t *testing.T) {
	opt := circuit.NewOption()
	opt.BreakDuration = 50 * time.Millisecond
	cb := circuit.New(opt)

	is := assert.New(t)
	is.Equal(circuit.Closed, cb.Status())

	// Shift to closed state.
	for range opt.FailureThreshold {
		err := cb.Do(func() error {
			return wantErr
		})
		is.NotNil(err)
	}
	is.Equal(circuit.Open, cb.Status())

	time.Sleep(opt.BreakDuration + 5*time.Millisecond)
	is.Equal(circuit.HalfOpen, cb.Status())

	// Trigger failure in half-opened state.
	err := cb.Do(func() error {
		return wantErr
	})
	is.ErrorIs(err, wantErr)
	is.Equal(circuit.Open, cb.Status())
}

func TestSlowCount(t *testing.T) {
	opt := circuit.NewOption()
	cb := circuit.New(opt)
	cb.SlowCallCount = func(time.Duration) int {
		// A single slow call will trigger the circuitbreaker to
		// open.
		return opt.FailureThreshold
	}
	err := cb.Do(func() error {
		// No error, but the slow call count is incremented.
		return nil
	})
	is := assert.New(t)
	is.Nil(err)
	is.Equal(circuit.Open, cb.Status())
}
