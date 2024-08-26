package pipeline

import "sync"

func OrDone[T any](done chan struct{}, in <-chan T) <-chan T {
	ch := make(chan T)

	go func() {
		defer close(ch)

		for {
			select {
			case <-done:
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				ch <- v
			}
		}
	}()

	return ch
}

func Pipe[T, V any](done chan struct{}, in chan T, fn func(T) V) <-chan V {
	// TODO: How to add buffer?
	ch := make(chan V)

	go func() {
		defer close(ch)

		for v := range OrDone(done, in) {
			ch <- fn(v)
		}
	}()

	return ch
}

func Map[T, V any](done chan struct{}, in chan T, fn func(T) V) <-chan V {
	ch := make(chan V)

	go func() {
		defer close(ch)

		for v := range OrDone(done, in) {
			ch <- fn(v)
		}
	}()

	return ch
}

func Batch() {}

func FanOut[T, V any](done chan struct{}, n int, in <-chan T, fn func(T) V) []chan V {
	res := make([]chan V, n)

	for i := range n {
		res[i] = make(chan V)

		go func() {
			defer close(res[i])

			for v := range OrDone(done, in) {
				res[i] <- fn(v)
			}
		}()
	}

	return res
}

func FanIn[T any](done chan struct{}, cs ...chan T) <-chan T {
	ch := make(chan T)

	var wg sync.WaitGroup
	wg.Add(len(cs))

	for _, c := range cs {
		go func(c chan T) {
			defer wg.Done()

			for v := range OrDone(done, c) {
				ch <- v
			}
		}(c)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch

}

// Transform.
func Deduplicate()  {}
func Cache()        {} // ReadThrough
func Idempotent()   {}
func Dataloader()   {}
func Singleflight() {}
func MaxInFlight()  {}
func Semaphore()    {}

// Resilience.
func RateLimit()      {}
func Throttle()       {}
func Retry()          {}
func CircuitBreaker() {}
func Debounce()       {}

// Pipe.
func Tee()      {}
func Bridge()   {}
func Take()     {}
func Repeat()   {}
func Generate() {}

type Result[T any] struct {
	Data T
	Err  error
}

func makeResult[T any](data T, err error) Result[T] {
	return Result[T]{Data: data, Err: err}
}

func ErrorHandler[T any](done chan struct{}, in <-chan Result[T], fn func(error)) <-chan T {
	ch := make(chan T)

	go func() {
		defer close(ch)

		for res := range OrDone(done, in) {
			if res.Err != nil {
				fn(res.Err)
				continue
			}

			ch <- res.Data
		}
	}()

	return ch
}
