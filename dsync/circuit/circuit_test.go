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
	opt := circuit.NewOption()
	opt.SamplingDuration = 1 * time.Second
	opt.BreakDuration = 1 * time.Second

	cb, stop1 := circuit.New(newClient(t), t.Name(), opt)
	defer stop1()
	cb2, stop2 := circuit.New(newClient(t), t.Name(), opt)
	defer stop2()

	t.Run("open", func(t *testing.T) {
		is := assert.New(t)
		for range opt.FailureThreshold {
			err := cb.Do(ctx, func() error {
				return wantErr
			})

			is.ErrorIs(err, wantErr)
		}

		// Wait for the redis subscribe message to be processed.
		time.Sleep(100 * time.Millisecond)

		err := cb.Do(ctx, func() error {
			return wantErr
		})

		is.ErrorIs(err, circuit.ErrUnavailable)
		is.Equal(circuit.Open, cb.Status())
		is.Equal(circuit.Open, cb2.Status())
	})

	t.Run("half-open", func(t *testing.T) {
		// Because this is Redis TTL, we need to wait for it to expire.
		time.Sleep(opt.BreakDuration + time.Millisecond)

		err := cb.Do(ctx, func() error {
			return nil
		})

		is := assert.New(t)
		is.ErrorIs(err, nil)
		is.Equal(circuit.HalfOpen, cb.Status())
	})

	t.Run("closed", func(t *testing.T) {
		is := assert.New(t)
		for range opt.SuccessThreshold {
			err := cb.Do(ctx, func() error {
				return nil
			})

			is.Nil(err)
		}

		is.Equal(circuit.Closed, cb.Status())
	})
}

func TestSlowCall(t *testing.T) {
	opt := circuit.NewOption()
	cb, stop := circuit.New(newClient(t), t.Name(), opt)
	defer stop()

	cb.SlowCallCount = func(time.Duration) int {
		// Ignore duration, just return a constant value.
		return opt.FailureThreshold
	}

	err := cb.Do(ctx, func() error {
		// No error, but failed due to slow call.
		return nil
	})

	// Wait for the message to be subscribed.
	time.Sleep(100 * time.Millisecond)
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
