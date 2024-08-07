package circuit_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/circuit"
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
	opts := circuit.NewOptions()
	opts.SamplingDuration = 1 * time.Second
	opts.BreakDuration = 1 * time.Second

	cb, stop1 := circuit.New(newClient(t), t.Name(), opts)
	defer stop1()
	cb2, stop2 := circuit.New(newClient(t), t.Name(), opts)
	defer stop2()

	t.Run("open", func(t *testing.T) {
		is := assert.New(t)
		for range opts.FailureThreshold {
			err := cb.Do(ctx, func() error {
				return wantErr
			})

			is.ErrorIs(err, wantErr)
		}

		err := cb.Do(ctx, func() error {
			return wantErr
		})

		is.ErrorIs(err, circuit.ErrUnavailable)
		is.Equal(circuit.Open, cb.Status())

		// Wait for message to be subscribed.
		time.Sleep(100 * time.Millisecond)
		is.Equal(circuit.Open, cb2.Status())
	})

	t.Run("half-open", func(t *testing.T) {
		// Because this is Redis TTL, we need to wait for it to expire.
		time.Sleep(opts.BreakDuration + time.Millisecond)

		err := cb.Do(ctx, func() error {
			return nil
		})

		is := assert.New(t)
		is.ErrorIs(err, nil)
		is.Equal(circuit.HalfOpen, cb.Status())
	})

	t.Run("closed", func(t *testing.T) {
		is := assert.New(t)
		for range opts.SuccessThreshold {
			err := cb.Do(ctx, func() error {
				return nil
			})

			is.Nil(err)
		}

		is.Equal(circuit.Closed, cb.Status())
	})
}

func TestSlowCall(t *testing.T) {
	opts := circuit.NewOptions()
	cb, stop := circuit.New(newClient(t), t.Name(), opts)
	defer stop()

	cb.SlowCallCount = func(time.Duration) int {
		// Ignore duration, just return a constant value.
		return opts.FailureThreshold
	}

	err := cb.Do(ctx, func() error {
		// No error, but failed due to slow call.
		return nil
	})

	is := assert.New(t)
	is.Nil(err)
	is.Equal(circuit.Open, cb.Status())
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
