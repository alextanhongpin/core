package pipeline

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
	"golang.org/x/sync/semaphore"
)

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

type ThroughputInfo struct {
	Total         int
	TotalFailures int
	Rate          int64
	ErrorRate     float64
}

func (t ThroughputInfo) String() string {
	return fmt.Sprintf("total: %d, errors: %d (%.0f%%), rate: %d req/s", t.Total, t.TotalFailures, t.ErrorRate*100, t.Rate)
}

func Throughput[T any](in <-chan Result[T], fn func(ThroughputInfo)) <-chan Result[T] {
	out := make(chan Result[T])

	var total, totalFailures int
	r := rate.NewRate(time.Second)
	er := rate.NewErrors(time.Second)

	go func() {
		defer close(out)

		for v := range in {
			_, err := v.Unwrap()
			if err != nil {
				totalFailures++
				er.Inc(-1)
			} else {
				er.Inc(1)
			}
			total++
			r.Inc(1)
			fn(ThroughputInfo{
				Rate:          int64(math.Ceil(r.Throughput())),
				Total:         total,
				TotalFailures: totalFailures,
				ErrorRate:     er.Rate(),
			})
			out <- v
		}
	}()

	return out
}

type RateInfo struct {
	Total int
	Rate  int64
}

func (r RateInfo) String() string {
	return fmt.Sprintf("total: %d, throughput: %d req/s", r.Total, r.Rate)
}

func Rate[T any](in <-chan T, fn func(RateInfo)) <-chan T {
	out := make(chan T)

	var total int
	r := rate.NewRate(time.Second)

	go func() {
		defer close(out)

		for v := range in {
			total++
			r.Inc(1)
			fn(RateInfo{
				Total: total,
				Rate:  int64(math.Ceil(r.Throughput())),
			})
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

func Semaphore[T, V any](n int, in <-chan T, fn func(T) V) <-chan V {
	out := make(chan V)

	ctx := context.Background()
	sem := semaphore.NewWeighted(int64(n))

	go func() {
		defer close(out)

		for v := range in {
			sem.Acquire(ctx, 1)

			go func() {
				defer sem.Release(1)

				out <- fn(v)
			}()
		}
		sem.Acquire(ctx, int64(n))
	}()

	return out
}

// Resilience.
func RateLimit[T any](request int, period time.Duration, in <-chan T) <-chan T {
	ch := make(chan T)

	go func() {
		t := time.NewTicker(period / time.Duration(request))
		defer t.Stop()

		defer close(ch)

		for v := range in {
			<-t.C
			ch <- v
		}
	}()

	return ch
}

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

func Error[T any](in <-chan Result[T], fn func(error)) <-chan T {
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

func Batch[T comparable](limit int, period time.Duration, in <-chan T) <-chan []T {
	out := make(chan []T)

	cache := make(map[T]struct{})
	batch := func() {
		if len(cache) == 0 {
			return
		}

		keys := make([]T, 0, len(cache))
		for k := range cache {
			keys = append(keys, k)
		}
		clear(cache)

		out <- keys
	}

	go func() {
		defer close(out)
		defer batch()

		t := time.NewTicker(period)
		defer t.Stop()

		for {
			select {
			case k, ok := <-in:
				if !ok {
					return
				}

				cache[k] = struct{}{}
				if len(cache) >= limit {
					batch()
				}
			case <-t.C:
				batch()
			}
		}
	}()

	return out
}
