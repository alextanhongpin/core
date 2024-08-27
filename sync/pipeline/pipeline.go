package pipeline

import (
	"context"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
)

// Queue acts as an intermediary stage that queues results to a buffered channel.
func Queue[T any](n int, in <-chan T) <-chan T {
	out := make(chan T, n)

	go func() {
		defer close(out)

		for v := range in {
			out <- v
		}
	}()

	return out
}

// Context acts as an intermediary stage that can be canceled.
func Context[T any](ctx context.Context, in <-chan T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					return
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

func OrDone[T any](done chan struct{}, in <-chan T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for {
			select {
			case <-done:
				return
			case v, ok := <-in:
				if !ok {
					return
				}

				select {
				case <-done:
					return
				case out <- v:
				}
			}
		}
	}()

	return out
}

func PassThrough[T any](in chan T, fn func(T)) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for v := range in {
			fn(v)
			out <- v
		}
	}()

	return out
}

func Progress[T any](period time.Duration, in <-chan T, fn func(total int, rate int64)) <-chan T {
	out := make(chan T)

	var i int
	r := rate.NewRate(period)

	go func() {
		defer close(out)

		for v := range in {
			i++
			fn(i, r.Inc(1))
			out <- v
		}
	}()

	return out
}

// Count reports the number of items passing through.
func Count[T any](in <-chan T, fn func(int)) <-chan T {
	out := make(chan T)

	var i int
	go func() {
		defer close(out)

		for v := range in {
			i++
			fn(i)
			out <- v
		}
	}()

	return out
}

func Rate[T any](period time.Duration, in <-chan T, fn func(int64)) <-chan T {
	out := make(chan T)
	r := rate.NewRate(period)

	go func() {
		defer close(out)

		for v := range in {
			fn(r.Inc(1))
			out <- v
		}
	}()

	return out
}

func ErrorCount[T any](in chan Result[T], fn func(failures, total int)) <-chan Result[T] {
	out := make(chan Result[T])

	var successes int
	var failures int
	go func() {
		defer close(out)

		for v := range in {
			_, err := v.Unwrap()
			if err != nil {
				failures++
			} else {
				successes++
			}
			fn(failures, failures+successes)
			out <- v
		}
	}()

	return out
}

func ErrorRate[T any](period time.Duration, in chan Result[T], fn func(failures, total float64)) <-chan Result[T] {
	out := make(chan Result[T])
	r := rate.NewErrors(period)

	go func() {
		defer close(out)

		for v := range in {
			var successes, failures float64
			_, err := v.Unwrap()
			if err != nil {
				successes, failures = r.Inc(-1)
			} else {
				successes, failures = r.Inc(1)
			}
			fn(failures, successes+failures)
			out <- v
		}
	}()

	return out
}

func Pipe[T any](in chan T, fn func(T) T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for v := range in {
			out <- fn(v)
		}
	}()

	return out
}

func Map[T, V any](in <-chan T, fn func(T) V) <-chan V {
	out := make(chan V)

	go func() {
		defer close(out)

		for v := range in {
			out <- fn(v)
		}
	}()

	return out
}

func Filter[T any](in chan T, fn func(T) bool) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for v := range in {
			if !fn(v) {
				continue
			}

			out <- v
		}
	}()

	return out
}

// Pool runs max n processes in-flight to process the incoming channel.
// This is not the same as fan-out.
func Pool[T, V any](n int, in <-chan T, fn func(T) V) <-chan V {
	out := make(chan V)

	var wg sync.WaitGroup
	wg.Add(n)

	for range n {
		go func() {
			defer wg.Done()

			for v := range in {
				out <- fn(v)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func FanOut[T any](n int, in <-chan T) []<-chan T {
	out := make([]<-chan T, n)

	for i := range n {
		out[i] = in
	}

	return out
}

func FanIn[T any](cs ...chan T) <-chan T {
	out := make(chan T)

	var wg sync.WaitGroup
	wg.Add(len(cs))

	for _, c := range cs {
		go func() {
			defer wg.Done()

			for v := range c {
				out <- v
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out

}

// Transform.
func Deduplicate() {}
func Cache()       {} // ReadThrough
func Idempotent()  {}
func Dataloader()  {}

// func Singleflight() {}
func Semaphore[T, V any](n int, in <-chan T, fn func(T) V) <-chan V {
	out := make(chan V)

	sem := make(chan struct{}, n)

	var wg sync.WaitGroup
	wg.Add(n)

	go func() {
		for v := range in {
			go func() {
				defer wg.Done()

				sem <- struct{}{}
				defer func() {
					<-sem
				}()

				out <- fn(v)
			}()
		}
	}()

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// Resilience.
func RateLimit[T any](request int, period time.Duration, in <-chan T) <-chan T {
	ch := make(chan T)
	t := time.NewTicker(period / time.Duration(request))

	go func() {
		defer t.Stop()
		defer close(ch)

		for v := range in {
			<-t.C
			ch <- v
		}
	}()

	return ch
}

//func Throttle() {}
//func Retry()    {}
//func CircuitBreaker() {}
//func Debounce()       {}

// Pipe.
func Bridge() {}
func Take()   {}

func Tee[T any](in chan T) (out1, out2 chan T) {
	out1, out2 = make(chan T), make(chan T)

	go func() {
		defer close(out1)
		defer close(out2)

		for v := range in {
			a, b := out1, out2

			for range 2 {
				select {
				case a <- v:
					a = nil
				case b <- v:
					b = nil
				}
			}
		}
	}()

	return
}

type Result[T any] struct {
	Data T
	Err  error
}

func (r Result[T]) Unwrap() (T, error) {
	return r.Data, r.Err
}

func MakeResult[T any](data T, err error) Result[T] {
	return Result[T]{Data: data, Err: err}
}

func FlatMap[T any](in <-chan Result[T]) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for res := range in {
			if res.Err != nil {
				continue
			}

			out <- res.Data
		}
	}()

	return out
}

func ErrorHandler[T any](in <-chan Result[T], fn func(error)) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		for res := range in {
			if res.Err != nil {
				fn(res.Err)
				continue
			}

			out <- res.Data
		}
	}()

	return out
}
