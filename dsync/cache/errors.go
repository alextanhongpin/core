package cache

import (
	"errors"

	redis "github.com/redis/go-redis/v9"
)

// ErrNotExist is returned when a key does not exist in the cache.
// ErrExists is returned when trying to store a key that already exists (StoreOnce).
var (
	ErrNotExist = redis.Nil
	ErrExists   = errors.New("key already exists")
)
