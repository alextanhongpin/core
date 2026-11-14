package ratelimit

import "errors"

var (
	// ErrNegative is returned when a negative number of requests is passed to
	// AllowN or LimitN. It indicates a programming error.
	ErrNegative = errors.New("ratelimit: requests cannot be negative")

	// ErrEmptyKey is returned when the key is empty. It indicates a
	// programming error.
	ErrEmptyKey = errors.New("ratelimit: empty key")

	// ErrNotSetup is returned (wrapped together with the Redis error) when the
	// Lua functions are not loaded on the Redis server. Call Setup at startup.
	// Detect it with errors.Is(err, ErrNotSetup).
	ErrNotSetup = errors.New("ratelimit: Redis function missing, call ratelimit.Setup first")
)
