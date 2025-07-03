// Package contextkey provides type-safe context key management for HTTP request handling.
//
// This package addresses the common need for storing and retrieving typed values
// in Go's context.Context while maintaining type safety and avoiding the usual
// pitfalls of context value management (like type assertions and key collisions).
//
// The Key type is generic, ensuring that values stored and retrieved are of the
// expected type, eliminating runtime type assertion errors and providing
// compile-time type safety.
//
// Example usage:
//
//	// Define typed context keys
//	var UserIDKey contextkey.Key[int] = "user_id"
//	var TokenKey contextkey.Key[string] = "auth_token"
//
//	// Store values in context
//	ctx = UserIDKey.WithValue(ctx, 123)
//	ctx = TokenKey.WithValue(ctx, "abc123")
//
//	// Retrieve values with type safety
//	userID, ok := UserIDKey.Value(ctx)        // userID is int, ok is bool
//	token := TokenKey.MustValue(ctx)          // token is string, panics if not found
//
// This approach eliminates the need for type assertions and provides clear
// documentation of what types are expected for each context key.
package contextkey

import (
	"context"
	"errors"
	"fmt"
)

// ErrNotFound is returned when a context key is not found in the context.
// This error is used by MustValue when a required key is missing.
var ErrNotFound = errors.New("contextkey: key not found")

// Key represents a typed context key that can store and retrieve values of type T.
//
// The generic type parameter T ensures type safety when storing and retrieving
// values from the context, eliminating the need for type assertions and
// providing compile-time type checking.
//
// Keys should typically be defined as package-level variables with descriptive
// names and specific types:
//
//	var UserIDKey contextkey.Key[int] = "user_id"
//	var SessionKey contextkey.Key[*Session] = "session"
type Key[T any] string

// WithValue returns a new context with the given value stored under this key.
//
// This method wraps context.WithValue but ensures type safety by only allowing
// values of the correct type T to be stored.
//
// Parameters:
//   - ctx: The parent context to derive from
//   - t: The value to store (must be of type T)
//
// Returns:
//   - A new context containing the stored value
//
// Example:
//
//	var UserIDKey contextkey.Key[int] = "user_id"
//	ctx = UserIDKey.WithValue(ctx, 123)
func (k Key[T]) WithValue(ctx context.Context, t T) context.Context {
	return context.WithValue(ctx, k, t)
}

// Value retrieves the value associated with this key from the context.
//
// This method provides type-safe retrieval of context values, returning
// the value with its correct type T and a boolean indicating whether
// the key was found in the context.
//
// Parameters:
//   - ctx: The context to search for the value
//
// Returns:
//   - The stored value of type T (zero value if not found)
//   - A boolean indicating whether the key was found
//
// Example:
//
//	userID, ok := UserIDKey.Value(ctx)
//	if !ok {
//		// Handle missing user ID
//		return errors.New("user not authenticated")
//	}
//	// Use userID (guaranteed to be int)
func (k Key[T]) Value(ctx context.Context) (T, bool) {
	t, ok := ctx.Value(k).(T)
	return t, ok
}

// MustValue retrieves the value associated with this key from the context.
//
// This method is similar to Value but panics if the key is not found in the context.
// It should only be used when the presence of the key is guaranteed or when
// a missing key represents a programming error that should halt execution.
//
// Parameters:
//   - ctx: The context to search for the value
//
// Returns:
//   - The stored value of type T
//
// Panics:
//   - If the key is not found in the context
//
// Example:
//
//	// Use only when key presence is guaranteed
//	userID := UserIDKey.MustValue(ctx)  // Will panic if user_id not in context
func (k Key[T]) MustValue(ctx context.Context) T {
	t, ok := k.Value(ctx)
	if !ok {
		panic(fmt.Errorf("%w: %s", ErrNotFound, k))
	}

	return t
}
