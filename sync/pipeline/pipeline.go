package pipeline

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

// Batch collects items into batches when the size or
// timeout exceeds the specied threshold.
func Batch[T any](in <-chan T, size int, timeout time.Duration) <-chan []T {
	if size <= 0 {
		panic("pipeline: invalid n")
	}

	out := make(chan []T)

	go func() {
		defer close(out)

		var batch []T
		timer := time.NewTimer(timeout)
		timer.Stop()

		flush := func() {
			if len(batch) > 0 {
				out <- batch
				batch = nil
			}
		}

		for {
			select {
			case v, ok := <-in:
				if !ok {
					flush()
					return
				}

				batch = append(batch, v)
				if len(batch) == 1 {
					timer.Reset(timeout)
				}

				if len(batch) >= size {
					timer.Stop()
					flush()
				}

			case <-timer.C:
				flush()
			}
		}
	}()

	return out
}

// Buffer creates a buffered channel stage
func Buffer[T any](in <-chan T, cap int) <-chan T {
	if cap <= 0 {
		panic("pipeline: invalid capacity")
	}

	out := make(chan T, cap)
	go func() {
		defer close(out)
		for v := range in {
			out <- v
		}
	}()
	return out
}

// Debounce ensures minimum time between emissions
func Debounce[T any](in <-chan T, duration time.Duration) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		var lastEmit time.Time
		for v := range in {
			if time.Since(lastEmit) >= duration {
				out <- v
				lastEmit = time.Now()
			}
		}
	}()

	return out
}

// Dedup removes duplicate items (requires comparable type)
func Dedup[T comparable](in <-chan T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		seen := make(map[T]struct{})
		for v := range in {
			if _, exists := seen[v]; !exists {
				seen[v] = struct{}{}
				out <- v
			}
		}
	}()

	return out
}

// FanIn merges multiple input channels into one output channel
func FanIn[T any](channels ...<-chan T) <-chan T {
	out := make(chan T)

	var wg sync.WaitGroup
	for _, ch := range channels {
		wg.Go(func() {
			for v := range ch {
				out <- v
			}
		})
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// FanOut distributes input to multiple output channels
func FanOut[T any](in <-chan T, n int) []<-chan T {
	if n <= 0 {
		panic("pipeline: invalid n")
	}

	outputs := make([]chan T, n)
	for i := range outputs {
		outputs[i] = make(chan T)
	}

	// Convert to read-only channels
	readOnlyOutputs := make([]<-chan T, n)
	for i, ch := range outputs {
		readOnlyOutputs[i] = ch
	}

	go func() {
		defer func() {
			for _, ch := range outputs {
				close(ch)
			}
		}()

		i := 0
		for v := range in {
			outputs[i%n] <- v
			i++
		}
	}()

	return readOnlyOutputs
}

// Merge merges multiple channels using a merge function
func Merge[T any](mergeFn func(T, T) T, channels ...<-chan T) <-chan T {
	if len(channels) == 0 {
		out := make(chan T)
		close(out)
		return out
	}

	if len(channels) == 1 {
		return channels[0]
	}

	// Binary merge
	merge := func(left, right <-chan T) <-chan T {
		out := make(chan T)

		go func() {
			defer close(out)

			for {
				select {
				case v1, ok1 := <-left:
					if !ok1 {
						// Left channel closed, drain right
						for v2 := range right {
							out <- v2
						}
						return
					}

					select {
					case v2, ok2 := <-right:
						if !ok2 {
							// Right channel closed, send left value and drain left
							out <- v1
							for v := range left {
								out <- v
							}
							return
						}
						out <- mergeFn(v1, v2)
					case out <- v1:
						// No value from right, send left value
					}

				case v2, ok2 := <-right:
					if !ok2 {
						// Right channel closed, drain left
						for v1 := range left {
							out <- v1
						}
						return
					}
					out <- v2
				}
			}
		}()

		return out
	}

	// Merge all channels in pairs
	result := channels[0]
	for i := 1; i < len(channels); i++ {
		result = merge(result, channels[i])
	}

	return result
}

// Pipe transform the value received by the input channel before passing to the
// output channel.
func Pipe[T, V any](in chan T, fn func(T) (V, bool)) chan V {
	return PipeN(in, fn, 1)
}

// PipeN is like Pipe, but runs multiple workers.
func PipeN[T, V any](in chan T, fn func(T) (V, bool), n int) chan V {
	if n == 0 {
		panic("min 1 running goroutine")
	}

	out := make(chan V)
	var wg sync.WaitGroup

	for range n {
		wg.Go(func() {
			for v := range in {
				res, ok := fn(v)
				if ok {
					out <- res
				}
			}
		})
	}

	go func() {
		defer close(out)
		wg.Wait()
	}()

	return out
}

// RateLimit limits the rate of items passing through
func RateLimit[T any](in <-chan T, every int, interval time.Duration) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		t := time.NewTicker(interval / time.Duration(every))
		defer t.Stop()

		for v := range in {
			<-t.C
			out <- v
		}
	}()

	return out
}

// SafeClose safely closes a channel, handling potential panics
func SafeClose[T any](ch chan T) (closed bool) {
	defer func() {
		if recover() != nil {
			closed = false
		}
	}()
	close(ch)
	return true
}

// Semaphore limits concurrent execution using a semaphore
func Semaphore[T, V any](in <-chan T, fn func(T) V, n int) <-chan V {
	if n <= 0 {
		panic("pipeline: invalid n")
	}

	out := make(chan V)
	ctx := context.Background()
	sem := semaphore.NewWeighted(int64(n))

	go func() {
		defer close(out)

		var wg sync.WaitGroup
		for v := range in {
			wg.Go(func() {
				_ = sem.Acquire(ctx, 1)
				defer sem.Release(1)

				out <- fn(v)
			})
		}
		wg.Wait()
	}()

	return out
}

// Tee splits a channel into two identical channels
func Tee[T any](in <-chan T) (<-chan T, <-chan T) {
	out1, out2 := make(chan T), make(chan T)

	go func() {
		defer close(out1)
		defer close(out2)

		for v := range in {
			// Send to both channels, blocking until both are ready
			var wg sync.WaitGroup
			wg.Go(func() {
				out1 <- v
			})

			wg.Go(func() {
				out2 <- v
			})

			wg.Wait()
		}
	}()

	return out1, out2
}
