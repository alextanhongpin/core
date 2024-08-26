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

func Pipe[T any](done chan struct{}, in chan T, fn func(T) T) <-chan T {
	ch := make(chan T)

	go func() {
		defer close(ch)

		for v := range OrDone(done, in) {
			ch <- fn(v)
		}
	}()

	return ch
}

func Map[K, V any](done chan struct{}, in chan K, fn func(K) V) <-chan V {
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

func FanOut[T any](done chan struct{}, n int, in <-chan T, fn func(T) T) []chan T {
	res := make([]chan T, n)

	for i := range n {
		res[i] = make(chan T)

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

func Deduplicate()    {}
func RateLimit()      {}
func Throttle()       {}
func Retry()          {}
func CircuitBreaker() {}
func Debounce()       {}
func Tee()            {}
func Bridge()         {}
