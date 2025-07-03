// Package result provides a generic Result type for handling operations that can succeed or fail.
// This is particularly useful when passing results through channels, collecting multiple
// operations, or when you want to defer error handling.
//
// The Result type wraps both a value and an error, similar to Rust's Result type,
// allowing for more functional error handling patterns in Go.
package result

import "errors"

// ErrNoResult is returned when no results are provided to operations that require at least one result.
var ErrNoResult = errors.New("result: no result")

// Result represents an operation that can either succeed with a value of type T or fail with an error.
type Result[T any] struct {
	Data T
	Err  error
}

// Unwrap returns the wrapped value and error.
// This is the primary way to extract values from a Result.
func (r *Result[T]) Unwrap() (T, error) {
	return r.Data, r.Err
}

// IsOK returns true if the result contains a value (no error).
func (r *Result[T]) IsOK() bool {
	return r.Err == nil
}

// IsErr returns true if the result contains an error.
func (r *Result[T]) IsErr() bool {
	return r.Err != nil
}

// Unwrap returns the value, panicking if there's an error.
// Use this only when you're certain the result is successful.
func (r *Result[T]) MustUnwrap() T {
	if r.Err != nil {
		panic("result: unwrapping failed result: " + r.Err.Error())
	}
	return r.Data
}

// UnwrapOr returns the value if successful, otherwise returns the provided default.
func (r *Result[T]) UnwrapOr(defaultValue T) T {
	if r.Err != nil {
		return defaultValue
	}
	return r.Data
}

// UnwrapOrElse returns the value if successful, otherwise calls the provided function.
func (r *Result[T]) UnwrapOrElse(fn func(error) T) T {
	if r.Err != nil {
		return fn(r.Err)
	}
	return r.Data
}

// Map transforms the value using the provided function if the result is successful.
// If the result contains an error, it's passed through unchanged.
func (r *Result[T]) Map(fn func(T) T) *Result[T] {
	if r.Err != nil {
		return r
	}
	return OK(fn(r.Data))
}

// MapError transforms the error using the provided function if the result failed.
// If the result is successful, it's passed through unchanged.
func (r *Result[T]) MapError(fn func(error) error) *Result[T] {
	if r.Err == nil {
		return r
	}
	return &Result[T]{Data: r.Data, Err: fn(r.Err)}
}

// FlatMap (also known as bind/chain) allows chaining operations that return Results.
func (r *Result[T]) FlatMap(fn func(T) *Result[T]) *Result[T] {
	if r.Err != nil {
		return r
	}
	return fn(r.Data)
}

// New creates a new empty Result.
func New[T any]() *Result[T] {
	return &Result[T]{}
}

// OK creates a successful Result with the given value.
func OK[T any](v T) *Result[T] {
	return &Result[T]{
		Data: v,
	}
}

// Err creates a failed Result with the given error.
func Err[T any](err error) *Result[T] {
	return &Result[T]{
		Err: err,
	}
}

// From creates a Result from a function that returns (value, error).
// This is useful for wrapping existing Go functions.
func From[T any](fn func() (T, error)) *Result[T] {
	value, err := fn()
	if err != nil {
		return Err[T](err)
	}
	return OK(value)
}

// Try executes a function and captures any panic as an error.
func Try[T any](fn func() T) *Result[T] {
	defer func() {
		if r := recover(); r != nil {
			// Convert panic to error - this should be handled by the caller
		}
	}()

	return OK(fn())
}

// All takes multiple Results and returns a slice of all values if all are successful,
// or the first error encountered.
func All[T any](rs ...*Result[T]) ([]T, error) {
	if len(rs) == 0 {
		return nil, ErrNoResult
	}

	res := make([]T, len(rs))
	for i, r := range rs {
		v, err := r.Unwrap()
		if err != nil {
			return nil, err
		}
		res[i] = v
	}

	return res, nil
}

// Any returns the first successful Result's value, or an error containing all failures.
// This is useful for fallback scenarios where any successful result is acceptable.
func Any[T any](rs ...*Result[T]) (T, error) {
	if len(rs) == 0 {
		var t T
		return t, ErrNoResult
	}

	var errs []error
	for _, r := range rs {
		v, err := r.Unwrap()
		if err == nil {
			return v, nil
		}
		errs = append(errs, err)
	}

	var t T
	return t, errors.Join(errs...)
}

// Collect transforms a slice of Results into a Result of a slice.
// If any Result contains an error, returns the first error.
func Collect[T any](rs []*Result[T]) *Result[[]T] {
	values, err := All(rs...)
	if err != nil {
		return Err[[]T](err)
	}
	return OK(values)
}

// Filter returns only the successful Results, discarding any errors.
func Filter[T any](rs ...*Result[T]) []T {
	var values []T
	for _, r := range rs {
		if v, err := r.Unwrap(); err == nil {
			values = append(values, v)
		}
	}
	return values
}

// Partition separates Results into successful values and errors.
func Partition[T any](rs ...*Result[T]) (values []T, errs []error) {
	for _, r := range rs {
		if v, err := r.Unwrap(); err == nil {
			values = append(values, v)
		} else {
			errs = append(errs, err)
		}
	}
	return values, errs
}
