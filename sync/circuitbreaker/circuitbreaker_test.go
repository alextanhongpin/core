package circuitbreaker_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker(t *testing.T) {
	var wantErr = errors.New("want error")

	assert := assert.New(t)

	cb := circuitbreaker.New()
	cb.SetTimeout(100 * time.Millisecond)

	assert.Equal(circuitbreaker.Closed, cb.Status())

	// Hit the failure threshold first.
	assert.ErrorIs(fire(10, wantErr, cb), wantErr)
	assert.Equal(circuitbreaker.Closed, cb.Status())

	// Above failure threshold, circuitbreaker becomes open.
	assert.ErrorIs(fire(1, wantErr, cb), wantErr)
	assert.ErrorIs(fire(1, wantErr, cb), circuitbreaker.Unavailable)
	assert.Equal(circuitbreaker.Open, cb.Status())
	assert.True(cb.ResetIn() > 0)

	// After timeout, it becomes half-open. But we need to trigger it once to
	// update the status first.
	time.Sleep(105 * time.Millisecond)
	assert.Nil(fire(1, nil, cb))
	assert.Equal(circuitbreaker.HalfOpen, cb.Status())
	assert.Equal(time.Duration(0), cb.ResetIn())

	// Hit the success threshold first.
	assert.Nil(fire(4, nil, cb))
	assert.Equal(circuitbreaker.HalfOpen, cb.Status())

	// After success threshold, it becomes closed again.
	assert.Nil(fire(1, nil, cb))
	assert.Equal(circuitbreaker.Closed, cb.Status())
}

type circuit interface {
	Do(func() error) error
}

func fire(n int, err error, cb circuit) error {
	var wg sync.WaitGroup
	wg.Add(n - 1)

	for i := 0; i < n-1; i++ {
		go func() {
			defer wg.Done()

			_ = cb.Do(func() error {
				return err
			})
		}()
	}
	wg.Wait()

	return cb.Do(func() error {
		return err
	})
}
