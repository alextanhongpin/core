package always

import "context"

// Verifier is a function type that represents a verification function.
// It takes a context and a value of type T, and returns an error if the verification fails.
type Verifier[T any] func(ctx context.Context, t T) error

// Verify is a function that verifies a value of type T using a list of verifiers.
// It takes a context.Context, a value t of type T, and a variadic parameter fns of type Verifier[T].
// It iterates through the list of verifiers and calls each verifier function with the context and value.
// If any verifier returns an error, Verify immediately returns that error.
// If all verifiers pass without any errors, Verify returns nil.
func Verify[T any](ctx context.Context, t T, fns ...Verifier[T]) error {
	for i := 0; i < len(fns); i++ {
		if err := fns[i](ctx, t); err != nil {
			return err
		}
	}

	return nil
}
