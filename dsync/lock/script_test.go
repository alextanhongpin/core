package lock_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestScript_LoadOrStore_Unlock(t *testing.T) {
	client := redistest.Client(t)

	store := lock.New(client)
	lockTTL := 100 * time.Millisecond

	t.Run("stored", func(t *testing.T) {
		var (
			is  = assert.New(t)
			key = t.Name()
		)
		_, loaded, err := store.LoadOrStore(ctx, key, "world", lockTTL)
		is.Nil(err, "expected error to be nil")
		is.False(loaded, "expected value to be stored")

		// Check the lock TTL.
		ttl := client.PTTL(ctx, key).Val()
		is.True(ttl <= lockTTL, "expected lock TTL to be close to 100ms")

		t.Run("loaded", func(t *testing.T) {
			val, loaded, err := store.LoadOrStore(ctx, key, "world", lockTTL)
			is := assert.New(t)
			is.Nil(err, "expected error to be nil")
			is.True(loaded, "then the value is loaded")
			is.Equal("world", val, "expected lock value to be 'world'")
		})

		t.Run("unlock failed with wrong key", func(t *testing.T) {
			is := assert.New(t)
			err := store.Unlock(ctx, key, "wrong-key")
			is.ErrorIs(err, lock.ErrConflict, "expected error to be ErrConflict")

			val, err := client.Get(ctx, key).Result()
			is.Nil(err, "expected error to be nil")
			is.Equal("world", val, "expected lock value to remain unchanged")
		})

		t.Run("unlock succeed", func(t *testing.T) {
			err := store.Unlock(ctx, key, "world")
			is := assert.New(t)
			is.Nil(err, "expected error to be nil")

			_, err = client.Get(ctx, key).Result()
			is.ErrorIs(err, redis.Nil, "expected lock to be released")
		})
	})
}

func TestScript_Replace(t *testing.T) {
	client := redistest.Client(t)
	store := lock.New(client)

	t.Run("when replace failed with invalid old value", func(t *testing.T) {
		var (
			is  = assert.New(t)
			key = t.Name()
		)

		// Set is value to be replaced.
		is.Nil(client.Set(ctx, key, "old-value", 0).Err())
		err := store.Replace(ctx, key, "invalid-old-value", "new-value", 2*time.Second)
		// The key should be released.
		is.ErrorIs(err, lock.ErrConflict)

		v, err := client.Get(ctx, key).Result()
		is.Nil(err, "expected error to be nil")
		is.Equal("old-value", v, "then the value stays the same")
	})

	t.Run("when replace success", func(t *testing.T) {
		var (
			is  = assert.New(t)
			key = t.Name()
		)

		// Set is value to be replaced.
		is.Nil(client.Set(ctx, key, "old-value", 0).Err())

		err := store.Replace(ctx, key, "old-value", "new-value", 200*time.Millisecond)
		is.Nil(err, "expected error to be nil")

		v, err := client.Get(ctx, key).Result()
		is.Nil(err, "expected error to be nil")
		is.Equal("new-value", v, "then the value will be replaced")

		// Check the updated TTL.
		ttl := client.PTTL(ctx, key).Val()
		is.True(ttl <= 200*time.Millisecond, "expected updated TTL to be close to 200ms")
	})
}
