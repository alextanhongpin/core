package idempotent

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/alextanhongpin/core/storage/redis/redistest"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	stop := redistest.Init()
	code := m.Run()
	stop()
	os.Exit(code)
}

func TestPrivateLoad(t *testing.T) {
	client := setupRedis(t)

	cleanup := func(t *testing.T) {
		t.Helper()
		t.Cleanup(func() {
			client.FlushAll(ctx).Err()
		})
	}

	idem := New[string, int](client, &Option{
		LockTTL: 100 * time.Millisecond,
		KeepTTL: 200 * time.Millisecond,
	})

	t.Run("when key does not exist", func(t *testing.T) {
		cleanup(t)

		_, err := idem.load(ctx, "hello", "world")
		assert.ErrorIs(t, err, redis.Nil)
	})

	t.Run("when key is uuid", func(t *testing.T) {
		cleanup(t)

		a := assert.New(t)
		a.Nil(client.Set(ctx, "hello", uuid.New().String(), 0).Err())
		_, err := idem.load(ctx, "hello", "world")
		a.ErrorIs(err, ErrRequestInFlight)
	})

	t.Run("when key is data", func(t *testing.T) {
		cleanup(t)

		a := assert.New(t)

		reqHash := hash([]byte(`"world"`))
		a.Nil(client.Set(ctx, "hello", fmt.Sprintf(`{"request": %q, "response": 42}`, reqHash), 0).Err())
		v, err := idem.load(ctx, "hello", "world")
		a.Nil(err)
		a.Equal(42, v)
	})

	t.Run("when request does not match", func(t *testing.T) {
		cleanup(t)

		a := assert.New(t)
		a.Nil(client.Set(ctx, "hello", `{"request": "world", "response": 42}`, 0).Err())
		_, err := idem.load(ctx, "hello", "not-world")
		a.ErrorIs(err, ErrRequestMismatch)
	})
}

func TestPrivateLockUnlock(t *testing.T) {
	client := setupRedis(t)

	cleanup := func(t *testing.T) {
		t.Helper()
		t.Cleanup(func() {
			client.FlushAll(ctx).Err()
		})
	}

	idem := New[string, int](client, &Option{
		LockTTL: 100 * time.Millisecond,
		KeepTTL: 200 * time.Millisecond,
	})

	t.Run("when lock success", func(t *testing.T) {
		cleanup(t)

		ok, err := idem.lock(ctx, "hello", []byte("world"))
		a := assert.New(t)
		a.Nil(err)
		a.True(ok)
		a.True(100*time.Millisecond-client.PTTL(ctx, "hello").Val() < 10*time.Millisecond)

		t.Run("when lock second time", func(t *testing.T) {
			ok, err = idem.lock(ctx, "hello", []byte("world"))
			a.Nil(err)
			a.False(ok, "then it will fail to lock")
		})

		t.Run("when unlock failed", func(t *testing.T) {
			err := idem.unlock(ctx, "hello", []byte("wrong-key"))
			a.Nil(err)

			val, err := client.Get(ctx, "hello").Result()
			a.Nil(err)
			a.Equal("world", val)
		})

		t.Run("when unlock success", func(t *testing.T) {
			err := idem.unlock(ctx, "hello", []byte("world"))
			a.Nil(err)

			_, err = client.Get(ctx, "hello").Result()
			a.ErrorIs(err, redis.Nil)
		})
	})
}

func TestPrivateReplace(t *testing.T) {
	client := setupRedis(t)

	cleanup := func(t *testing.T) {
		t.Helper()
		t.Cleanup(func() {
			client.FlushAll(ctx).Err()
		})
	}

	idem := New[string, int](client, &Option{
		LockTTL: 1 * time.Second,
		KeepTTL: 2 * time.Second,
	})

	t.Run("when replace failed", func(t *testing.T) {
		cleanup(t)

		a := assert.New(t)

		// Set a value to be replaced.
		a.Nil(client.Set(ctx, "hello", "world", 0).Err())
		a.Nil(idem.replace(ctx, "hello", []byte("invalid-old-value"), "new-value"))

		v, err := client.Get(ctx, "hello").Result()
		a.Nil(err)
		a.Equal("world", v, "then the value stays the same")
	})

	t.Run("when replace success", func(t *testing.T) {
		cleanup(t)

		a := assert.New(t)

		// Set a value to be replaced.
		a.Nil(client.Set(ctx, "hello", "world", 0).Err())
		a.Nil(idem.replace(ctx, "hello", []byte("world"), "new-value"))

		v, err := client.Get(ctx, "hello").Result()
		a.Nil(err)
		a.Equal(`"new-value"`, v, "then the value will be replaced")
		a.True(200*time.Millisecond-client.PTTL(ctx, "hello").Val() < 10*time.Millisecond)
	})
}

func setupRedis(t *testing.T) *redis.Client {
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
