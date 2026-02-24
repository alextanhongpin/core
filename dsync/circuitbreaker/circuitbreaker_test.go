package circuitbreaker_test

import (
	"os"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"github.com/alextanhongpin/core/storage/redis/redistest"
)

func TestMain(m *testing.M) {
	stop := redistest.Init()
	code := m.Run()
	stop()
	os.Exit(code)
}

func newClient(t *testing.T) *redis.Client {
	t.Helper()

	client := redistest.Client(t)

	if err := circuitbreaker.Setup(t.Context(), client); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	return client
}

func TestCircuitBreakerSuite(t *testing.T) {
	suite.Run(t, new(CircuitBreakerSuite))
}

type CircuitBreakerSuite struct {
	suite.Suite

	client  *redis.Client
	options *circuitbreaker.Options
	cb      *circuitbreaker.CircuitBreaker
}

func (s *CircuitBreakerSuite) SetupTest() {
	options := circuitbreaker.NewOptions()
	options.FailureThreshold = 10
	options.SuccessThreshold = 10
	options.OpenTimeout = 100 * time.Millisecond

	s.client = newClient(s.T())
	s.options = options
	s.cb = circuitbreaker.New(s.client, options)
}

func (s *CircuitBreakerSuite) statusIs(status circuitbreaker.Status) {
	t := s.T()

	ctx := t.Context()
	key := t.Name()

	got, err := s.cb.Status(ctx, key)

	s.NoError(err)
	s.Equal(status, got)
}

func (s *CircuitBreakerSuite) setStatus(status circuitbreaker.Status) {
	t := s.T()

	ctx := t.Context()
	key := t.Name()

	err := s.cb.SetStatus(ctx, key, status)
	s.NoError(err)
}

func (s *CircuitBreakerSuite) runErr(want error) {
	t := s.T()

	ctx := t.Context()
	key := t.Name()
	got := s.cb.Do(ctx, key, func() error {
		return assert.AnError
	})
	s.ErrorIs(got, want)
}

func (s *CircuitBreakerSuite) run(want error) {
	t := s.T()

	ctx := t.Context()
	key := t.Name()
	got := s.cb.Do(ctx, key, func() error {
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
