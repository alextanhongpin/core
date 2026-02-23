package v2

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/alextanhongpin/core/storage/redis/redistest"
	redis "github.com/redis/go-redis/v9"
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
	for range cb.failureThreshold {
		err := cb.Do(ctx, key, func() error {
			return wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("unknown error: %v", err)
		}
	}
	err := cb.Do(ctx, key, func() error {
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
	//t.Log(client.HGetAll(ctx, key).Result())
	err = cb.Do(ctx, key, func() error {
		return nil
	})

	debug("HALF_OPEN SUCCESS")
	if err != nil {
		t.Fatalf("got error: %v", err)
	}
}

func newClient(t *testing.T) *redis.Client {
	t.Helper()

	client := redistest.Client(t)

	if err := Setup(t.Context(), client); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	return client
}
