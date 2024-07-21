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
	opt.FailureThreshold = 5
	opt.BreakDuration = 1 * time.Second

	cb := circuit.New(newClient(t), t.Name(), opt)
	{
		stop := cb.Start()
		defer stop()
	}
	cb2 := circuit.New(newClient(t), t.Name(), opt)
	{
		stop := cb2.Start()
		defer stop()
	}

	is := assert.New(t)
	for i := 0; i < opt.FailureThreshold; i++ {
		err := cb.Do(ctx, func() error {
			return wantErr
		})

		is.ErrorIs(err, wantErr)
	}

	// Wait for the redis subscribe message to be processed.
	time.Sleep(100 * time.Millisecond)

	t.Run("failure", func(t *testing.T) {
		err := cb.Do(ctx, func() error {
			return wantErr
		})

		assert.ErrorIs(t, err, circuit.ErrUnavailable)
	})

	t.Run("pubsub", func(t *testing.T) {
		err := cb2.Do(ctx, func() error {
			return wantErr
		})

		assert.ErrorIs(t, err, circuit.ErrUnavailable)
	})

	t.Run("recover", func(t *testing.T) {
		// Because this is Redis TTL, we need to wait for it to expire.
		time.Sleep(opt.BreakDuration)

		err := cb.Do(ctx, func() error {
			return nil
		})

		assert.ErrorIs(t, err, nil)
	})
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
