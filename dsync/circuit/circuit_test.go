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

	cb := circuit.New(newClient(t), opt)
	key := t.Name()

	is := assert.New(t)
	for i := 0; i < opt.FailureThreshold; i++ {
		err := cb.Do(ctx, key, func() error {
			return wantErr
		})

		is.ErrorIs(err, wantErr)
	}

	t.Run("failure", func(t *testing.T) {
		err := cb.Do(ctx, key, func() error {
			return wantErr
		})

		is := assert.New(t)
		is.ErrorIs(err, circuit.ErrUnavailable)
	})

	t.Run("recover", func(t *testing.T) {
		// Because this is Redis TTL, we need to wait for it to expire.
		time.Sleep(opt.BreakDuration)

		err := cb.Do(ctx, key, func() error {
			return nil
		})

		is := assert.New(t)
		is.ErrorIs(err, nil)
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
