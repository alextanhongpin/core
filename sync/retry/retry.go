package retry

import (
	"math/rand"
	"time"
)

func Linear[T any]() Timeouts[T] {
	n := 10

	res := make([]time.Duration, n)
	for i := 0; i < n; i++ {
		res[i] = 1 * time.Second
	}

	return res
}

func Exponential[T any]() Timeouts[T] {
	n := 10

	res := make([]time.Duration, n)
	res[0] = 50 * time.Millisecond
	for i := 1; i < n; i++ {
		res[i] = jitter(res[i-1] * 2)
	}

	return res
}

func jitter(d time.Duration) time.Duration {
	return (time.Duration(rand.Intn(int(d/2))) + d).Round(5 * time.Millisecond)
}

type Timeouts[T any] []time.Duration

func (ts Timeouts[T]) timeouts() []time.Duration {
	return append([]time.Duration{0}, ts...)
}

func (ts Timeouts[T]) Exec(fn func() error) (err error) {
	for _, t := range ts.timeouts() {
		if t != 0 {
			time.Sleep(jitter(t))
		}

		err = fn()
		if err == nil {
			return
		}
	}

	return
}

func (ts Timeouts[T]) Query(fn func() (T, error)) (v T, err error) {
	for _, t := range ts.timeouts() {
		if t != 0 {
			time.Sleep(jitter(t))
		}

		v, err = fn()
		if err == nil {
			return
		}
	}

	return
}
