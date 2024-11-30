package lock_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestPrivateLockUnlock(t *testing.T) {
	client := redistest.Client(t)

	cleanup := func(t *testing.T) {
		t.Helper()
		t.Cleanup(func() {
			client.FlushAll(ctx).Err()
		})
	}

	store := lock.New(client)
	store.LockTTL = 100 * time.Millisecond
	store.WaitTTL = 0

	t.Run("when lock success", func(t *testing.T) {
		cleanup(t)

		key := t.Name()
		_, loaded, err := store.LoadOrStore(ctx, key, "world", store.LockTTL)
		assert.Nil(t, err, "expected error to be nil")
		assert.False(t, loaded, "expected value to be stored")

		// Check the lock TTL.
		lockTTL := client.PTTL(ctx, key).Val()
		assert.True(t, 100*time.Millisecond-lockTTL < 10*time.Millisecond, "expected lock TTL to be close to 100ms")

		t.Run("when lock second time", func(t *testing.T) {
			lockValue, loaded, err := store.LoadOrStore(ctx, key, "world", store.LockTTL)
			assert.Nil(t, err, "expected error to be nil")
			assert.True(t, loaded, "then the value is loaded")
			assert.Equal(t, "world", lockValue, "expected lock value to be 'world'")
		})

		t.Run("when unlock failed with wrong key", func(t *testing.T) {
			err := store.Unlock(ctx, key, "wrong-key")
			assert.ErrorIs(t, err, lock.ErrConflict, "expected error to be ErrConflict")

			val, err := client.Get(ctx, key).Result()
			assert.Nil(t, err, "expected error to be nil")
			assert.Equal(t, "world", val, "expected lock value to remain unchanged")
		})

		t.Run("when unlock success", func(t *testing.T) {
			err := store.Unlock(ctx, key, "world")
			assert.Nil(t, err, "expected error to be nil")

			_, err = client.Get(ctx, key).Result()
			assert.ErrorIs(t, err, redis.Nil, "expected lock to be released")
		})
	})
}

func TestPrivateReplace(t *testing.T) {
	client := redistest.Client(t)

	cleanup := func(t *testing.T) {
		t.Helper()
		t.Cleanup(func() {
			client.FlushAll(ctx).Err()
		})
	}

	store := lock.New(client)
	store.LockTTL = time.Second

	t.Run("when replace failed with invalid old value", func(t *testing.T) {
		cleanup(t)

		a := assert.New(t)
		key := t.Name()

		// Set a value to be replaced.
		a.Nil(client.Set(ctx, key, "world", 0).Err())

		err := store.Replace(ctx, key, "invalid-old-value", "new-value", 2*time.Second)
		// The key should be released.
		a.ErrorIs(err, lock.ErrConflict)

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

		err := store.Replace(ctx, key, "world", "new-value", 2*time.Second)
		a.Nil(err, "expected error to be nil")

		v, err := client.Get(ctx, key).Result()
		a.Nil(err, "expected error to be nil")
		a.Equal("new-value", v, "then the value will be replaced")

		// Check the updated TTL.
		updatedTTL := client.PTTL(ctx, key).Val()
		a.True(200*time.Millisecond-updatedTTL < 10*time.Millisecond, "expected updated TTL to be close to 200ms")
	})
}
