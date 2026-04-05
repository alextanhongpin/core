// source must be cancellable. It takes the context as the first argument.
package pipeline

import (
	"context"
	"iter"
)

// SourceChan creates a new source channel from a channel.
func SourceChan[T any](ctx context.Context, in chan T) chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					break
				}

				select {
				case <-ctx.Done():
					return
				case out <- v:
				}
			}
		}
	}()

	return out
}

// SourceIter creates a new source channel from an iterator.
func SourceIter[T any](ctx context.Context, seq iter.Seq[T]) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for v := range seq {
			select {
			case <-ctx.Done():
				return
			case out <- v:
			}
		}
	}()

	return out
}

// SourceSlice creates a source channel from a slice.
func SourceSlice[T any](ctx context.Context, values []T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for _, v := range values {
			select {
			case <-ctx.Done():
				return
			case out <- v:
			}
		}
	}()

	return out
}
