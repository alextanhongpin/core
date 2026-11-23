package must

import (
	"fmt"
)

// Value returns v, or panics with an error wrapping err when err is non-nil.
func Value[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Errorf("must: %w", err))
	}
	return v
}

// NoError panics with an error wrapping err when err is non-nil.
func NoError(err error) {
	if err != nil {
		panic(fmt.Errorf("must: %w", err))
	}
}
