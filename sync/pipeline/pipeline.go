package pipeline

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

// Batch collects items into batches when the size or
// timeout exceeds the specified threshold. The consumer must drain the output
// until it is closed. Use BatchCtx to be able to cancel the stage.
func Batch[T any](in <-chan T, size int, timeout time.Duration) <-chan []T {
	return BatchCtx(context.Background(), in, size, timeout)
}

// BatchCtx is like Batch, but stops when ctx is canceled: it stops reading in,
// stops sending (a partially filled batch is discarded), closes the output and
// exits, even if nobody is receiving from the output.
func BatchCtx[T any](ctx context.Context, in <-chan T, size int, timeout time.Duration) <-chan []T {
	if size <= 0 {
		panic("pipeline: invalid n")
	}

	out := make(chan []T)

	go func() {
		defer close(out)

		var batch []T
		var timer *time.Timer
		var tick <-chan time.Time
		defer func() {
			if timer != nil {
				timer.Stop()
			}
		}()
		// flush reports false if ctx was canceled before the batch was sent.
		flush := func() bool {
			if len(batch) > 0 {
				if timer != nil {
					timer.Stop()
				}
				tick = nil
				select {
				case out <- batch:
				case <-ctx.Done():
					return false
				}
				batch = nil
			}
			return true
		}

		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					flush()
					return
				}
				if len(batch) == 0 {
					if timer == nil {
						timer = time.NewTimer(timeout)
					} else {
						timer.Reset(timeout)
					}
					tick = timer.C
				}
				batch = append(batch, v)
				if len(batch) >= size && !flush() {
					return
				}
			case <-tick:
				if !flush() {
					return
				}
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

// Debounce is a leading-edge throttle, not a trailing-edge debounce: it emits
// an item, then drops every item that arrives within duration of that
// emission. Dropped items are lost. The consumer must drain the output.
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

// Dedup removes duplicate items (requires comparable type). It remembers every
// value it has seen for as long as the stage runs, so memory grows with the
// number of distinct values; do not use it on unbounded streams of distinct
// values.
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

// DedupFunc is like Dedup, but compares items by the key fn returns and emits
// the first item seen for each key. Like Dedup, it keeps an unbounded set of
// seen keys for the lifetime of the stage.
func DedupFunc[T any, V comparable](in <-chan T, fn func(T) V) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		seen := make(map[V]struct{})
		for v := range in {
			k := fn(v)
			if _, exists := seen[k]; !exists {
				seen[k] = struct{}{}
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

// FanOut distributes input round-robin to n output channels. It has
// head-of-line blocking: it sends to one output at a time, so a slow or
// undrained output stalls delivery to all the others. Consumers must drain
// every output until it is closed.
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

// Merge merges multiple channels using a merge function. mergeFn is applied
// only opportunistically: when a value from one channel is received and a value
// from the other is ready at that moment they are combined, otherwise values
// are forwarded unchanged. Which values get combined, and how many, depends on
// scheduling and is nondeterministic. Consumers must drain the output.
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
// output channel. The consumer must drain the output until it is closed. Use
// PipeCtx to be able to cancel the stage.
func Pipe[T, V any](in <-chan T, fn func(T) (V, bool)) <-chan V {
	return PipeNCtx(context.Background(), in, fn, 1)
}

// PipeCtx is like Pipe, but stops when ctx is canceled. See PipeNCtx.
func PipeCtx[T, V any](ctx context.Context, in <-chan T, fn func(T) (V, bool)) <-chan V {
	return PipeNCtx(ctx, in, fn, 1)
}

// PipeN is like Pipe, but runs multiple workers; results may arrive out of
// order. The consumer must drain the output until it is closed.
func PipeN[T, V any](in <-chan T, fn func(T) (V, bool), n int) <-chan V {
	return PipeNCtx(context.Background(), in, fn, n)
}

// PipeNCtx is like PipeN, but stops when ctx is canceled: workers stop reading
// in and stop sending, the output is closed once every worker has exited (a
// callback already running is waited for), even if nobody is receiving.
func PipeNCtx[T, V any](ctx context.Context, in <-chan T, fn func(T) (V, bool), n int) <-chan V {
	if n <= 0 {
		panic("min 1 running goroutine")
	}

	out := make(chan V)
	var wg sync.WaitGroup

	for range n {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case v, ok := <-in:
					if !ok {
						return
					}
					res, ok := fn(v)
					if !ok {
						continue
					}
					select {
					case out <- res:
					case <-ctx.Done():
						return
					}
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
	if every <= 0 || interval <= 0 || interval/time.Duration(every) <= 0 {
		panic("pipeline: invalid rate limit")
	}
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

// Semaphore limits concurrent execution using a semaphore. The consumer must
// drain the output until it is closed. Use SemaphoreCtx to be able to cancel
// the stage.
func Semaphore[T, V any](in <-chan T, fn func(T) V, n int) <-chan V {
	return SemaphoreCtx(context.Background(), in, fn, n)
}

// SemaphoreCtx is like Semaphore, but stops when ctx is canceled: it stops
// reading in and starting callbacks, pending sends are abandoned, and the output
// is closed after running callbacks return, even if nobody is receiving.
func SemaphoreCtx[T, V any](ctx context.Context, in <-chan T, fn func(T) V, n int) <-chan V {
	if n <= 0 {
		panic("pipeline: invalid n")
	}

	out := make(chan V)
	sem := semaphore.NewWeighted(int64(n))

	go func() {
		defer close(out)

		var wg sync.WaitGroup
		defer wg.Wait()
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				if sem.Acquire(ctx, 1) != nil {
					return
				}
				wg.Go(func() {
					defer sem.Release(1)

					res := fn(v)
					select {
					case out <- res:
					case <-ctx.Done():
					}
				})
			}
		}
	}()

	return out
}

// Tee splits a channel into two identical channels. It has head-of-line
// blocking: each value is sent to both outputs and the next value is not read
// until both have received it, so a slow or undrained output stalls the other.
// Consumers must drain both outputs until they are closed. Use TeeCtx to be
// able to cancel the stage.
func Tee[T any](in <-chan T) (<-chan T, <-chan T) {
	return TeeCtx(context.Background(), in)
}

// TeeCtx is like Tee, but stops when ctx is canceled: it stops reading in,
// abandons pending sends, and closes both outputs, even if nobody is receiving.
func TeeCtx[T any](ctx context.Context, in <-chan T) (<-chan T, <-chan T) {
	out1, out2 := make(chan T), make(chan T)

	go func() {
		defer close(out1)
		defer close(out2)

		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				// Send to both channels, blocking until both are ready.
				var wg sync.WaitGroup
				wg.Go(func() {
					select {
					case out1 <- v:
					case <-ctx.Done():
					}
				})
				wg.Go(func() {
					select {
					case out2 <- v:
					case <-ctx.Done():
					}
				})
				wg.Wait()
			}
		}
	}()

	return out1, out2
}
