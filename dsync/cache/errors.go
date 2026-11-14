package cache

import (
	"errors"
)

var (
	// ErrNotExist is returned when a key is missing or expired. Compare
	// operations (CompareAndSwap, CompareAndDelete) also return it when the
	// current value does not match. Treat it as a cache miss.
	ErrNotExist = errors.New("not exists")

	// ErrExists is returned by StoreOnce when the key already exists.
	ErrExists = errors.New("already exists")

	// ErrLocked is returned by [Lock] when another process holds the lease
	// for the key (Wait is zero, or Wait elapsed), or when this process lost
	// its lease before finishing. Retry later or serve a fallback.
	ErrLocked = errors.New("locked")

	// ErrClosed is returned by [File] and [FS] after Close.
	ErrClosed = errors.New("cache: closed")

	// ErrConflict is returned by an [Idempotent] function when the key was
	// already used with a different request.
	ErrConflict = errors.New("conflict")
)
