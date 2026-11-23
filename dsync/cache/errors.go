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

	// ErrClosed is returned by [File] and [FS] after Close.
	ErrClosed = errors.New("cache: closed")
)
