package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestCircuitBreaker(t *testing.T) {
	var wantErr = errors.New("want error")
	ctx := context.Background()

	assert := assert.New(t)

	cb := newCircuitBreaker()

	assert.Equal(Closed, cb.Status())

	// Hit the failure threshold first.
	assert.ErrorIs(fire(ctx, cb, 10, wantErr), wantErr)
	assert.Equal(Closed, cb.Status())

	// Above failure threshold, circuitbreaker becomes open.
	assert.ErrorIs(fire(ctx, cb, 1, wantErr), wantErr)
	assert.ErrorIs(fire(ctx, cb, 1, wantErr), Unavailable)
	assert.Equal(Open, cb.Status())
	assert.True(cb.ResetIn() > 0)

	// After timeout, it becomes half-open. But we need to trigger it once to
	// update the status first.
	time.Sleep(105 * time.Millisecond)
	assert.Nil(fire(ctx, cb, 1, nil))
	assert.Equal(HalfOpen, cb.Status())
	assert.Equal(time.Duration(0), cb.ResetIn())

	// Hit the success threshold first.
	assert.Nil(fire(ctx, cb, 5, nil))
	assert.Equal(HalfOpen, cb.Status())

	// After success threshold, it becomes closed again.
	assert.Nil(fire(ctx, cb, 1, nil))
	assert.Equal(Closed, cb.Status())
}

func TestCircuitBreakerResetBeforeOpen(t *testing.T) {
	var wantErr = errors.New("want error")
	ctx := context.Background()

	assert := assert.New(t)

	cb := newCircuitBreaker()

	assert.Equal(Closed, cb.Status())

	// Hit the failure threshold first.
	assert.ErrorIs(fire(ctx, cb, 10, wantErr), wantErr)
	assert.Equal(Closed, cb.Status())

	// Sleep until resets.
	time.Sleep(105 * time.Millisecond)
	assert.ErrorIs(fire(ctx, cb, 1, wantErr), wantErr)
	assert.Equal(Closed, cb.Status())
	assert.Equal(time.Duration(0), cb.ResetIn())
}

func TestCircuitBreakerInsufficientErrorRate(t *testing.T) {
	var wantErr = errors.New("want error")
	ctx := context.Background()

	assert := assert.New(t)

	cb := newCircuitBreaker()

	assert.Equal(Closed, cb.Status())

	// Hit the failure threshold first.
	assert.Nil(fire(ctx, cb, 5, nil))
	assert.ErrorIs(fire(ctx, cb, 10, wantErr), wantErr)
	assert.Equal(Closed, cb.Status())

	// Above failure threshold, but below error rate, so circuitbreaker remains
	// closed.
	assert.ErrorIs(fire(ctx, cb, 1, wantErr), wantErr)
	assert.Equal(Closed, cb.Status())
	assert.Equal(time.Duration(0), cb.ResetIn())
}

func TestStore(t *testing.T) {
	assert := assert.New(t)

	cb := newCircuitBreaker()
	assert.Equal(Closed, cb.Status())

	cb.opt.Store = &mockStore{status: Open}
	err := cb.Do(func() error {
		return nil
	})
	assert.ErrorIs(err, Unavailable)
	assert.Equal(Open, cb.Status())
}

func TestClosedState(t *testing.T) {
	t.Run("resets counter on entry", func(t *testing.T) {
		opt := NewOption()
		opt.count = 999
		state := NewClosedState(opt)
		state.Entry()
		assert.Equal(t, int64(0), state.opt.count)
	})

	t.Run("cannot transition to Open when success", func(t *testing.T) {
		opt := NewOption()
		opt.count = 10
		state := NewClosedState(opt)
		status, ok := state.Next()
		assert := assert.New(t)
		assert.False(ok)
		assert.Equal(Open, status)
	})

	t.Run("transitions to Open when above error rate", func(t *testing.T) {
		opt := NewOption()

		state := NewClosedState(opt)
		state.Entry()

		opt.count = opt.FailureThreshold + 1
		opt.total = opt.FailureThreshold + 1

		status, ok := state.Next()

		assert := assert.New(t)
		assert.True(ok)
		assert.Equal(Open, status)
	})

	t.Run("cannot transition to Open when below error rate", func(t *testing.T) {
		opt := NewOption()
		opt.count = (opt.FailureThreshold + 1)
		opt.total = (opt.FailureThreshold + 1) * 2

		state := NewClosedState(opt)
		status, ok := state.Next()

		assert := assert.New(t)
		assert.False(ok)
		assert.Equal(Open, status)
	})

	t.Run("resets counter after interval ends", func(t *testing.T) {
		assert := assert.New(t)

		wantErr := errors.New("want error")
		opt := NewOption()
		opt.Sometimes = rate.Sometimes{
			Interval: 100 * time.Millisecond,
		}

		state := NewClosedState(opt)
		state.Entry()
		err := state.Do(func() error {
			return wantErr
		})

		assert.ErrorIs(err, wantErr)
		assert.Equal(int64(1), state.opt.count)
		assert.Equal(int64(1), state.opt.total)

		state.opt.count += opt.FailureThreshold
		state.opt.total = state.opt.count

		// To reset the timer.
		time.Sleep(105 * time.Millisecond)

		assert.False(state.isFailureThresholdReached())

		err = state.Do(func() error {
			return wantErr
		})

		assert.False(state.isFailureThresholdReached())
		assert.ErrorIs(err, wantErr)
		assert.Equal(int64(1), state.opt.count)
		assert.Equal(int64(1), state.opt.total)
	})
}

func TestOpenState(t *testing.T) {
	t.Run("starts half-open timeout timer on entry", func(t *testing.T) {
		now := time.Now()
		opt := NewOption()
		opt.Now = func() time.Time {
			return now
		}

		state := NewOpenState(opt)
		state.Entry()
		assert.Equal(t, now.Add(opt.HalfOpenTimeout), state.opt.openedAt)
	})

	t.Run("cannot transition to HalfOpen timeout timer is running", func(t *testing.T) {
		now := time.Now()
		opt := NewOption()
		opt.Now = func() time.Time {
			return now
		}

		state := NewOpenState(opt)
		state.Entry()
		status, ok := state.Next()
		assert := assert.New(t)
		assert.False(ok)
		assert.Equal(HalfOpen, status)
	})

	t.Run("transitions to HalfOpen on timer expired", func(t *testing.T) {
		now := time.Now()
		opt := NewOption()
		opt.Now = func() time.Time {
			return now
		}

		state := NewOpenState(opt)
		state.Entry()

		opt.Now = func() time.Time {
			return now.Add(opt.HalfOpenTimeout).Add(1 * time.Nanosecond)
		}

		status, ok := state.Next()
		assert := assert.New(t)
		assert.True(ok)
		assert.Equal(HalfOpen, status)
	})
}

func TestHalfOpenState(t *testing.T) {
	t.Run("resets success counter on entry", func(t *testing.T) {
		opt := NewOption()
		state := NewHalfOpenState(opt)
		state.Entry()
		assert.Equal(t, int64(0), state.opt.count)
	})

	t.Run("transitions to Open on error", func(t *testing.T) {
		wantErr := errors.New("want error")
		opt := NewOption()
		state := NewHalfOpenState(opt)

		err := state.Do(func() error {
			return wantErr
		})

		assert := assert.New(t)
		assert.ErrorIs(err, wantErr)

		status, ok := state.Next()
		assert.True(ok)
		assert.Equal(Open, status)
	})

	t.Run("cannot transition to Closed until success", func(t *testing.T) {
		opt := NewOption()
		state := NewHalfOpenState(opt)

		err := state.Do(func() error {
			return nil
		})

		assert := assert.New(t)
		assert.Nil(err)

		status, ok := state.Next()
		assert.False(ok)
		assert.Equal(Closed, status)
	})

	t.Run("transitions to Closed on success", func(t *testing.T) {
		opt := NewOption()
		opt.count = opt.SuccessThreshold
		state := NewHalfOpenState(opt)

		err := state.Do(func() error {
			return nil
		})

		assert := assert.New(t)
		assert.Nil(err)

		status, ok := state.Next()
		assert.True(ok)
		assert.Equal(Closed, status)
	})
}

type circuit interface {
	Do(func() error) error
}

func newCircuitBreaker() *CircuitBreaker {
	opt := NewOption()
	opt.HalfOpenTimeout = 100 * time.Millisecond
	opt.Sometimes = rate.Sometimes{
		Interval: 100 * time.Millisecond,
	}
	return New(opt)
}

func fire(ctx context.Context, cb circuit, n int, err error) error {
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

type mockStore struct {
	status Status
}

func (s *mockStore) Get() (Status, bool) {
	return s.status, true
}

func (s *mockStore) Set(status Status) {
	s.status = status
}
