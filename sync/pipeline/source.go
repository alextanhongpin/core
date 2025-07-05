// source must be cancellable. It takes the context as the first argument.
package pipeline

import "context"

func Generator(ctx context.Context, n int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		for i := range n {
			select {
			case <-ctx.Done():
				return
			case out <- i:
			}
		}
	}()

	return out
}

func GeneratorFunc[T any](ctx context.Context, fn func() T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case out <- fn():
			}
		}
	}()

	return out
}

func Repeat[T any](ctx context.Context, vs ...T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for {
			for _, v := range vs {
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

func RepeatFunc[T any](ctx context.Context, fn func() []T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for {
			for _, v := range fn() {
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

// From creates a channel from a slice of values
func From[T any](ctx context.Context, values ...T) <-chan T {
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

// FromSlice creates a channel from a slice
func FromSlice[T any](ctx context.Context, values []T) <-chan T {
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

// Range creates a channel that emits values from start to end (exclusive)
func Range(ctx context.Context, start, end int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		for i := start; i < end; i++ {
			select {
			case <-ctx.Done():
				return
			case out <- i:
			}
		}
	}()

	return out
}

// RangeStep creates a channel that emits values from start to end with step
func RangeStep(ctx context.Context, start, end, step int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		for i := start; i < end; i += step {
			select {
			case <-ctx.Done():
				return
			case out <- i:
			}
		}
	}()

	return out
}
