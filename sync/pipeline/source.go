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
