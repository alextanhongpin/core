package circuitbreaker_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var (
	ctx     = context.Background()
	wantErr = errors.New("want error")
)

func TestMain(m *testing.M) {
	stop := redistest.Init()
	code := m.Run()
	stop()
	os.Exit(code)
}

func TestCircuit(t *testing.T) {

	cb, stop1 := circuitbreaker.New(newClient(t), t.Name())
	cb.SamplingDuration = 1 * time.Second
	cb.BreakDuration = 1 * time.Second
	defer stop1()
	cb2, stop2 := circuitbreaker.New(newClient(t), t.Name())
	cb2.SamplingDuration = 1 * time.Second
	cb2.BreakDuration = 1 * time.Second
	defer stop2()

	t.Run("open", func(t *testing.T) {
		is := assert.New(t)
		for range cb.FailureThreshold {
			err := cb.Do(ctx, func() error {
				return wantErr
			})

			is.ErrorIs(err, wantErr)
		}

		err := cb.Do(ctx, func() error {
			return wantErr
		})

		is.ErrorIs(err, circuitbreaker.ErrUnavailable)
		is.Equal(circuitbreaker.Open, cb.Status())

		// Wait for message to be subscribed.
		time.Sleep(100 * time.Millisecond)
		is.Equal(circuitbreaker.Open, cb2.Status())
	})

	t.Run("half-open", func(t *testing.T) {
		// Because this is Redis TTL, we need to wait for it to expire.
		time.Sleep(cb.BreakDuration + time.Millisecond)

		err := cb.Do(ctx, func() error {
			return nil
		})

		is := assert.New(t)
		is.ErrorIs(err, nil)
		is.Equal(circuitbreaker.HalfOpen, cb.Status())
	})

	t.Run("closed", func(t *testing.T) {
		is := assert.New(t)
		for range cb.SuccessThreshold {
			err := cb.Do(ctx, func() error {
				return nil
			})

			is.Nil(err)
		}

		is.Equal(circuitbreaker.Closed, cb.Status())
	})
}

func TestSlowCall(t *testing.T) {
	cb, stop := circuitbreaker.New(newClient(t), t.Name())
	defer stop()

	cb.SlowCallCount = func(time.Duration) int {
		// Ignore duration, just return a constant value.
		return cb.FailureThreshold
	}

	err := cb.Do(ctx, func() error {
		// No error, but failed due to slow call.
		return nil
	})

	is := assert.New(t)
	is.Nil(err)
	is.Equal(circuitbreaker.Open, cb.Status())
}

func newClient(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})

	t.Helper()
	t.Cleanup(func() {
		client.FlushAll(ctx).Err()
		client.Close()
	})

	return client
}
