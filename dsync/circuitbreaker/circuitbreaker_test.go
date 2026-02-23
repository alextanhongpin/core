package circuitbreaker_test

import (
	"os"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

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

func TestCircuitBreaker(t *testing.T) {
	s := newSuite(t)
	s.runErr(t, nil, nil)

	// Initial state.
	s.statusIs(t, circuitbreaker.Closed)

	// Trigger open.
	for range s.options.FailureThreshold {
		s.runErr(t, assert.AnError, assert.AnError)
	}

	s.statusIs(t, circuitbreaker.Opened)
	time.Sleep(s.options.OpenTimeout)

	// Half-open, but trigger an error.
	s.runErr(t, assert.AnError, assert.AnError)
	s.statusIs(t, circuitbreaker.Opened)
	s.runErr(t, assert.AnError, circuitbreaker.ErrOpened)

	time.Sleep(s.options.OpenTimeout)
	s.runErr(t, nil, nil)
	s.statusIs(t, circuitbreaker.HalfOpen)

	// Half-open, but make it successful.
	for range s.options.SuccessThreshold {
		s.runErr(t, nil, nil)
	}

	s.statusIs(t, circuitbreaker.Closed)
}

func TestCircuitBreaker_SetStatus(t *testing.T) {
	t.Run("forced open", func(t *testing.T) {
		s := newSuite(t)
		s.setStatus(t, circuitbreaker.ForcedOpen)
		s.runErr(t, nil, circuitbreaker.ErrOpened)
	})

	t.Run("disabled", func(t *testing.T) {
		s := newSuite(t)
		s.setStatus(t, circuitbreaker.Disabled)
		for range s.options.FailureThreshold + 1 {
			s.runErr(t, assert.AnError, assert.AnError)
		}
	})
}

type suite struct {
	client  *redis.Client
	options *circuitbreaker.Options
	cb      *circuitbreaker.CircuitBreaker
}

func newSuite(t *testing.T) *suite {
	client := newClient(t)

	options := circuitbreaker.NewOptions()
	options.FailureThreshold = 10
	options.SuccessThreshold = 10
	options.OpenTimeout = 100 * time.Millisecond
	cb := circuitbreaker.New(client, options)

	return &suite{
		client:  client,
		options: options,
		cb:      cb,
	}
}

func (s *suite) statusIs(t *testing.T, status circuitbreaker.Status) {
	t.Helper()

	ctx := t.Context()
	key := t.Name()

	got, err := s.cb.Status(ctx, key)

	is := assert.New(t)
	is.NoError(err)
	is.Equal(status, got)
}

func (s *suite) setStatus(t *testing.T, status circuitbreaker.Status) {
	t.Helper()

	ctx := t.Context()
	key := t.Name()
	is := assert.New(t)

	err := s.cb.SetStatus(ctx, key, status)
	is.NoError(err)

	s.statusIs(t, status)
}

func (s *suite) runErr(t *testing.T, with error, want error) {
	t.Helper()

	ctx := t.Context()
	key := t.Name()
	got := s.cb.Do(ctx, key, func() error {
		return with
	})
	assert.ErrorIs(t, got, want)
}
