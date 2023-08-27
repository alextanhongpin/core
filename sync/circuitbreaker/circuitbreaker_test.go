package circuitbreaker_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
	"github.com/stretchr/testify/assert"
)

var svc = &service{}

type service struct {
	err error
}

func (s *service) SetError(err error) {
	s.err = err
}

func (s *service) Exec(ctx context.Context) error {
	return s.err
}

func TestCircuitBreaker(t *testing.T) {

	var wantErr = errors.New("want error")
	ctx := context.Background()

	assert := assert.New(t)

	opt := circuitbreaker.NewOption()
	opt.Timeout = 100 * time.Millisecond
	opt.Handler = svc
	cb := circuitbreaker.New(opt)

	assert.Equal(circuitbreaker.Closed, cb.Status())

	// Hit the failure threshold first.
	assert.ErrorIs(fire(ctx, cb, 10, wantErr), wantErr)
	assert.Equal(circuitbreaker.Closed, cb.Status())

	// Above failure threshold, circuitbreaker becomes open.
	assert.ErrorIs(fire(ctx, cb, 1, wantErr), wantErr)
	assert.ErrorIs(fire(ctx, cb, 1, wantErr), circuitbreaker.Unavailable)
	assert.Equal(circuitbreaker.Open, cb.Status())
	assert.True(cb.ResetIn() > 0)

	// After timeout, it becomes half-open. But we need to trigger it once to
	// update the status first.
	time.Sleep(105 * time.Millisecond)
	assert.Nil(fire(ctx, cb, 1, nil))
	assert.Equal(circuitbreaker.HalfOpen, cb.Status())
	assert.Equal(time.Duration(0), cb.ResetIn())

	// Hit the success threshold first.
	assert.Nil(fire(ctx, cb, 4, nil))
	assert.Equal(circuitbreaker.HalfOpen, cb.Status())

	// After success threshold, it becomes closed again.
	assert.Nil(fire(ctx, cb, 1, nil))
	assert.Equal(circuitbreaker.Closed, cb.Status())
}

type circuit interface {
	Exec(ctx context.Context) error
}

func fire(ctx context.Context, cb circuit, n int, err error) error {
	var wg sync.WaitGroup
	wg.Add(n - 1)
	svc.SetError(err)

	for i := 0; i < n-1; i++ {
		go func() {
			defer wg.Done()

			_ = cb.Exec(ctx)
		}()
	}
	wg.Wait()

	return cb.Exec(ctx)
}
