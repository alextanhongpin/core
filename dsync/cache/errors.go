package cache

import (
	"errors"
)

// ErrNotExist is returned when a key does not exist in the cache.
// ErrExists is returned when trying to store a key that already exists (StoreOnce).
var (
	ErrNotExist = errors.New("not exists")
	ErrExists   = errors.New("already exists")
	ErrLocked   = errors.New("locked")
	ErrConflict = errors.New("conflict")
)
