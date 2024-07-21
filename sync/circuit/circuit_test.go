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
	opt.BreakDuration = 100 * time.Millisecond
	cb := circuit.New(opt)

	is := assert.New(t)
	is.Equal(circuit.Closed, cb.Status())

	for range opt.FailureThreshold {
		err := cb.Do(func() error {
			return wantErr
		})
		is.ErrorIs(err, wantErr)
	}

	t.Run("opened", func(t *testing.T) {
		err := cb.Do(func() error {
			return wantErr
		})
		is := assert.New(t)
		is.ErrorIs(err, circuit.ErrBrokenCircuit)
		is.Equal(circuit.Open, cb.Status())
	})

	t.Run("half-opened", func(t *testing.T) {
		time.Sleep(opt.BreakDuration + time.Millisecond)
		err := cb.Do(func() error {
			return nil
		})
		is := assert.New(t)
		is.Nil(err)
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
