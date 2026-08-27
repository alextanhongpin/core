package circuitbreaker_test

import (
	"context"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestCircuitBreakerSuite(t *testing.T) {
	suite.Run(t, new(CircuitBreakerSuite))
}

type CircuitBreakerSuite struct {
	suite.Suite

	options *circuitbreaker.Options
	cb      *circuitbreaker.CircuitBreaker
}

func (s *CircuitBreakerSuite) SetupTest() {
	options := circuitbreaker.NewOptions()
	options.FailureThreshold = 10
	options.SuccessThreshold = 10
	options.OpenTimeout = 100 * time.Millisecond

	s.options = options
	s.cb = circuitbreaker.New(options)
}

func (s *CircuitBreakerSuite) statusIs(status circuitbreaker.Status) {
	got := s.cb.Status()
	s.Equal(status, got)
}

func (s *CircuitBreakerSuite) setStatus(status circuitbreaker.Status) {
	s.cb.SetStatus(status)
}

func (s *CircuitBreakerSuite) runErr(want error) {
	got := s.cb.Do(s.T().Context(), func(context.Context) error {
		return assert.AnError
	})
	s.ErrorIs(got, want)
}

func (s *CircuitBreakerSuite) run(want error) {
	t := s.T()

	ctx := t.Context()
	got := s.cb.Do(ctx, func(context.Context) error {
		return nil
	})
	s.ErrorIs(got, want)
}

func (s *CircuitBreakerSuite) triggerOpen() {
	for range s.options.FailureThreshold {
		s.runErr(assert.AnError)
	}
}

func (s *CircuitBreakerSuite) TestClosed() {
	s.run(nil)
	s.statusIs(circuitbreaker.Closed)
}

func (s *CircuitBreakerSuite) TestOpened() {
	s.triggerOpen()
	s.statusIs(circuitbreaker.Opened)
}

func (s *CircuitBreakerSuite) TestHalfOpenError() {
	s.triggerOpen()
	s.statusIs(circuitbreaker.Opened)
	time.Sleep(s.options.OpenTimeout)

	s.runErr(assert.AnError)
	s.statusIs(circuitbreaker.Opened)
	s.run(circuitbreaker.ErrOpened)
}

func (s *CircuitBreakerSuite) TestHalfOpenSuccess() {
	s.triggerOpen()
	s.statusIs(circuitbreaker.Opened)
	time.Sleep(s.options.OpenTimeout)

	s.run(nil)
	s.statusIs(circuitbreaker.HalfOpen)

	for range s.options.SuccessThreshold {
		s.run(nil)
	}
	s.statusIs(circuitbreaker.Closed)
}

func (s *CircuitBreakerSuite) TestForcedOpen() {
	s.setStatus(circuitbreaker.ForcedOpen)
	s.run(circuitbreaker.ErrOpened)
	s.statusIs(circuitbreaker.ForcedOpen)
}

func (s *CircuitBreakerSuite) TestDisabled() {
	s.setStatus(circuitbreaker.Disabled)
	for range s.options.FailureThreshold + 1 {
		s.runErr(assert.AnError)
	}
	s.statusIs(circuitbreaker.Disabled)
}
