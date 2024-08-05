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

		_, err := idem.load(ctx, t.Name(), "world")
		// Assert that the error is redis.Nil, indicating that the key does not
		// exist.
		assert.ErrorIs(t, err, redis.Nil)
	})

	t.Run("when key is uuid", func(t *testing.T) {
		cleanup(t)

		a := assert.New(t)
		key := t.Name()
		a.Nil(client.Set(ctx, key, uuid.New().String(), 0).Err())
		_, err := idem.load(ctx, key, "world")
		// Assert that the error is ErrRequestInFlight, indicating that a request
		// is already in flight for the key.
		a.ErrorIs(err, ErrRequestInFlight)
	})

	t.Run("when key is data", func(t *testing.T) {
		cleanup(t)

		a := assert.New(t)

		key := t.Name()
		reqHash := hash([]byte(`"world"`))
		a.Nil(client.Set(ctx, key, fmt.Sprintf(`{"request": %q, "response": 42}`, reqHash), 0).Err())
		v, err := idem.load(ctx, key, "world")
		a.Nil(err)
		a.Equal(42, v)
	})

	t.Run("when request does not match", func(t *testing.T) {
		cleanup(t)

		a := assert.New(t)
		key := t.Name()
		a.Nil(client.Set(ctx, key, `{"request": "world", "response": 42}`, 0).Err())
		_, err := idem.load(ctx, key, "not-world")
		// Assert that the error is ErrRequestMismatch, indicating that the request
		// does not match the stored request for the key.
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

		key := t.Name()
		ok, err := idem.lock(ctx, key, []byte("world"), idem.lockTTL)
		assert.Nil(t, err, "expected error to be nil")
		assert.True(t, ok, "expected lock to succeed")

		// Check the lock TTL.
		lockTTL := client.PTTL(ctx, key).Val()
		assert.True(t, 100*time.Millisecond-lockTTL < 10*time.Millisecond, "expected lock TTL to be close to 100ms")

		t.Run("when lock second time", func(t *testing.T) {
			ok, err = idem.lock(ctx, key, []byte("world"), idem.lockTTL)
			assert.Nil(t, err, "expected error to be nil")
			assert.False(t, ok, "then it will fail to lock")
			// Verify that the lock is still held by the first request
			lockValue, err := client.Get(ctx, key).Bytes()
			assert.Nil(t, err, "expected error to be nil")
			assert.Equal(t, []byte("world"), lockValue, "expected lock value to be 'world'")
		})

		t.Run("when unlock failed with wrong key", func(t *testing.T) {
			err := idem.unlock(ctx, key, []byte("wrong-key"))
			assert.ErrorIs(t, err, ErrLeaseInvalid, "expected error to be ErrLeaseInvalid")

			val, err := client.Get(ctx, key).Result()
			assert.Nil(t, err, "expected error to be nil")
			assert.Equal(t, "world", val, "expected lock value to remain unchanged")
		})

		t.Run("when unlock success", func(t *testing.T) {
			err := idem.unlock(ctx, key, []byte("world"))
			assert.Nil(t, err, "expected error to be nil")

			_, err = client.Get(ctx, key).Result()
			assert.ErrorIs(t, err, redis.Nil, "expected lock to be released")
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

	t.Run("when replace failed with invalid old value", func(t *testing.T) {
		cleanup(t)

		a := assert.New(t)
		key := t.Name()

		// Set a value to be replaced.
		a.Nil(client.Set(ctx, key, "world", 0).Err())

		err := idem.replace(ctx, key, []byte("invalid-old-value"), []byte("new-value"), idem.keepTTL)
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

		err := idem.replace(ctx, key, []byte("world"), []byte("new-value"), idem.keepTTL)
		a.Nil(err, "expected error to be nil")

		v, err := client.Get(ctx, key).Result()
		a.Nil(err, "expected error to be nil")
		a.Equal("new-value", v, "then the value will be replaced")

		// Check the updated TTL.
		updatedTTL := client.PTTL(ctx, key).Val()
		a.True(200*time.Millisecond-updatedTTL < 10*time.Millisecond, "expected updated TTL to be close to 200ms")
	})
}

// TestSlow test the scenario where the callback function takes a longer time
// than the lock expiry, and the lock expired before the callback function
// completes.
// We expect the lock to be refresh periodically.
func TestSlow(t *testing.T) {
	client := setupRedis(t)

	idem := New[string, int](client, &Option{
		LockTTL: 100 * time.Millisecond,
		KeepTTL: 200 * time.Millisecond,
	})
	fn := func(ctx context.Context, req string) (int, error) {
		// slow function
		time.Sleep(250 * time.Millisecond)
		return 42, nil
	}
	_, err := idem.Do(ctx, t.Name(), fn, "world")
	if err != nil {
		t.Fatal(err)
	}
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
