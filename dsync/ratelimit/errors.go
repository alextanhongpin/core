package ratelimit

import "errors"

var (
	// ErrNegative is returned when a negative number of requests is passed to
	// AllowN or LimitN. It indicates a programming error.
	ErrNegative = errors.New("ratelimit: requests cannot be negative")

	// ErrEmptyKey is returned when the key is empty. It indicates a
	// programming error.
	ErrEmptyKey = errors.New("ratelimit: empty key")
)
