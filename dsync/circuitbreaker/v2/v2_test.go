package v2

import (
	"os"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

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

	if err := Setup(t.Context(), client); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	return client
}

func TestCircuitBreaker(t *testing.T) {
	s := newSuite(t)
	s.runErr(t, nil, nil)

	// Initial state.
	s.statusIs(t, Closed)

	// Trigger open.
	for range s.cb.failureThreshold {
		s.runErr(t, assert.AnError, assert.AnError)
	}

	s.statusIs(t, Opened)
	time.Sleep(s.cb.openTimeout)

	// Half-open, but trigger an error.
	s.runErr(t, assert.AnError, assert.AnError)
	s.statusIs(t, Opened)
	s.runErr(t, assert.AnError, ErrOpened)

	time.Sleep(s.cb.openTimeout)
	s.runErr(t, nil, nil)
	s.statusIs(t, HalfOpen)

	// Half-open, but make it successful.
	for range s.cb.successThreshold {
		s.runErr(t, nil, nil)
	}

	s.statusIs(t, Closed)
}

func TestCircuitBreaker_SetStatus(t *testing.T) {
	t.Run("forced open", func(t *testing.T) {
		s := newSuite(t)
		s.setStatus(t, ForcedOpen)
		s.runErr(t, nil, ErrOpened)
	})

	t.Run("disabled", func(t *testing.T) {
		s := newSuite(t)
		s.setStatus(t, Disabled)
		for range s.cb.failureThreshold + 1 {
			s.runErr(t, assert.AnError, assert.AnError)
		}
	})
}

type suite struct {
	client *redis.Client
	cb     *CircuitBreaker
}

func newSuite(t *testing.T) *suite {
	client := newClient(t)

	cb := NewCircuitBreaker(client)
	cb.openTimeout = 100 * time.Millisecond

	return &suite{
		client: client,
		cb:     cb,
	}
}

func (s *suite) statusIs(t *testing.T, status Status) {
	t.Helper()

	ctx := t.Context()
	key := t.Name()

	got, err := s.cb.Status(ctx, key)

	is := assert.New(t)
	is.NoError(err)
	is.Equal(status, got)
}

func (s *suite) setStatus(t *testing.T, status Status) {
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
