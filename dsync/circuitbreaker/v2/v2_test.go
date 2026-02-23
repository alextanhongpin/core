package v2

import (
	"errors"
	"os"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"

	"github.com/alextanhongpin/core/storage/redis/redistest"
)

func TestMain(m *testing.M) {
	stop := redistest.Init()
	defer stop()

	os.Exit(m.Run())
}

func TestCircuitBreaker(t *testing.T) {
	ctx := t.Context()
	key := t.Name()

	client := newClient(t)
	cb := NewCircuitBreaker(client)
	cb.openTimeout = 100 * time.Millisecond

	debug := func(msg string) {
		res, err := client.HGetAll(ctx, key).Result()
		t.Log(msg, res, err)
	}

	wantErr := errors.New("bad request")
	err := cb.Do(ctx, key, func() error {
		return nil
	})
	debug("CLOSED")
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	for range cb.failureThreshold {
		err := cb.Do(ctx, key, func() error {
			return wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("unknown error: %v", err)
		}
	}
	err = cb.Do(ctx, key, func() error {
		return wantErr
	})
	debug("OPENED")
	if !errors.Is(err, ErrOpened) {
		t.Fatalf("want open, got %v", err)
	}

	time.Sleep(cb.openTimeout)
	err = cb.Do(ctx, key, func() error {
		return wantErr
	})
	debug("OPEN TIMEOUT")

	if !errors.Is(err, wantErr) {
		t.Fatalf("want open, got %v", err)
	}
	err = cb.Do(ctx, key, func() error {
		return wantErr
	})
	debug("HALF_OPEN ERROR")

	if !errors.Is(err, ErrOpened) {
		t.Fatalf("want open, got %v", err)
	}
	time.Sleep(cb.openTimeout)

	for range cb.successThreshold {
		err = cb.Do(ctx, key, func() error {
			return nil
		})

		if err != nil {
			t.Fatalf("got error: %v", err)
		}
	}
	debug("HALF_OPEN SUCCESS")

	err = cb.Do(ctx, key, func() error {
		return nil
	})
	debug("CLOSED")
}

func newClient(t *testing.T) *redis.Client {
	t.Helper()

	client := redistest.Client(t)

	if err := Setup(t.Context(), client); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	return client
}
