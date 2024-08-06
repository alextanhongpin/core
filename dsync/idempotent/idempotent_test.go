package idempotent

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func TestMakeHandler(t *testing.T) {
	store := NewRedisStore(newClient(t), nil)
	h := MakeHandler(store, func(ctx context.Context, req string) (string, error) {
		return "world", nil
	})

	res, shared, err := h.Do(ctx, t.Name(), "hello")
	is := assert.New(t)
	is.Nil(err)
	is.False(shared)
	is.Equal("world", res)

	res, shared, err = h.Do(ctx, t.Name(), "hello")
	is.Nil(err)
	is.True(shared)
	is.Equal("world", res)
}

func TestPrivateLockUnlock(t *testing.T) {
	client := newClient(t)

	cleanup := func(t *testing.T) {
		t.Helper()
		t.Cleanup(func() {
			client.FlushAll(ctx).Err()
		})
	}

	store := NewRedisStore(client, &Options{
		LockTTL: 100 * time.Millisecond,
		KeepTTL: 200 * time.Millisecond,
	})

	t.Run("when lock success", func(t *testing.T) {
		cleanup(t)

		key := t.Name()
		_, loaded, err := store.loadOrStore(ctx, key, "world", store.opts.LockTTL)
		assert.Nil(t, err, "expected error to be nil")
		assert.False(t, loaded, "expected value to be stored")

		// Check the lock TTL.
		lockTTL := client.PTTL(ctx, key).Val()
		assert.True(t, 100*time.Millisecond-lockTTL < 10*time.Millisecond, "expected lock TTL to be close to 100ms")

		t.Run("when lock second time", func(t *testing.T) {
			lockValue, loaded, err := store.loadOrStore(ctx, key, "world", store.opts.LockTTL)
			assert.Nil(t, err, "expected error to be nil")
			assert.True(t, loaded, "then the value is loaded")
			assert.Equal(t, []byte("world"), lockValue, "expected lock value to be 'world'")
		})

		t.Run("when unlock failed with wrong key", func(t *testing.T) {
			err := store.unlock(ctx, key, "wrong-key")
			assert.ErrorIs(t, err, ErrLeaseInvalid, "expected error to be ErrLeaseInvalid")

			val, err := client.Get(ctx, key).Result()
			assert.Nil(t, err, "expected error to be nil")
			assert.Equal(t, "world", val, "expected lock value to remain unchanged")
		})

		t.Run("when unlock success", func(t *testing.T) {
			err := store.unlock(ctx, key, "world")
			assert.Nil(t, err, "expected error to be nil")

			_, err = client.Get(ctx, key).Result()
			assert.ErrorIs(t, err, redis.Nil, "expected lock to be released")
		})
	})
}

func TestPrivateReplace(t *testing.T) {
	client := newClient(t)

	cleanup := func(t *testing.T) {
		t.Helper()
		t.Cleanup(func() {
			client.FlushAll(ctx).Err()
		})
	}

	store := NewRedisStore(client, &Options{
		LockTTL: 1 * time.Second,
		KeepTTL: 2 * time.Second,
	})

	t.Run("when replace failed with invalid old value", func(t *testing.T) {
		cleanup(t)

		a := assert.New(t)
		key := t.Name()

		// Set a value to be replaced.
		a.Nil(client.Set(ctx, key, "world", 0).Err())

		err := store.replace(ctx, key, "invalid-old-value", "new-value", store.opts.KeepTTL)
		// The key should be released.
		a.ErrorIs(err, ErrLeaseInvalid)

		v, err := client.Get(ctx, key).Result()
		a.Nil(err, "expected error to be nil")
		a.Equal("world", v, "then the value stays the same")
	})

	t.Run("when replace success", func(t *testing.T) {
		cleanup(t)

		a := assert.New(t)
		key := t.Name()

		// Set a value to be replaced.
		a.Nil(client.Set(ctx, key, "world", 0).Err())

		err := store.replace(ctx, key, "world", "new-value", store.opts.KeepTTL)
		a.Nil(err, "expected error to be nil")

		v, err := client.Get(ctx, key).Result()
		a.Nil(err, "expected error to be nil")
		a.Equal("new-value", v, "then the value will be replaced")

		// Check the updated TTL.
		updatedTTL := client.PTTL(ctx, key).Val()
		a.True(200*time.Millisecond-updatedTTL < 10*time.Millisecond, "expected updated TTL to be close to 200ms")
	})
}

func TestConcurrent(t *testing.T) {
	type Request struct {
		Msg string
	}
	type Response struct {
		Msg string
	}

	fn := func(ctx context.Context, req Request) (*Response, error) {
		time.Sleep(100 * time.Millisecond)
		return &Response{
			Msg: strings.ToUpper(req.Msg),
		}, nil
	}

	client := newClient(t)
	store := NewRedisStore(client, nil)
	h := MakeHandler(store, fn)
	n := 10

	is := assert.New(t)

	var wg sync.WaitGroup
	wg.Add(n)

	counter := new(atomic.Int64)

	for range n {
		go func() {
			defer wg.Done()

			res, shared, err := h.Do(ctx, t.Name(), Request{Msg: "hello"})
			if err == nil {
				is.Equal("HELLO", res.Msg)
				is.False(shared)
				return
			}

			if errors.Is(err, ErrRequestInFlight) {
				counter.Add(1)
				return
			}

			is.Nil(err)
		}()
	}

	wg.Wait()
	is.Equal(int64(n-1), counter.Load())
}

// TestExtendLock test the scenario where the callback function takes a longer
// time than the lock expiry, and the lock expired before the callback function
// completes.
// We expect the lock to be refresh periodically.
func TestExtendLock(t *testing.T) {
	client := newClient(t)

	fn := func(ctx context.Context, req string) (int, error) {
		// slow function
		time.Sleep(250 * time.Millisecond)
		return 42, nil
	}

	store := NewRedisStore(client, nil)
	opts := []Option{WithKeepTTL(200 * time.Millisecond), WithLockTTL(100 * time.Millisecond)}
	h := MakeHandler(store, fn, opts...)
	_, _, err := h.Do(ctx, t.Name(), "world")
	if err != nil {
		t.Fatal(err)
	}
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
