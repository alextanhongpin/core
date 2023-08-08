package retry

import (
	"math/rand"
	"time"
)

var (
	Linear      = linear()
	Exponential = exponential()
)

func Exec(fn func() error, ts []time.Duration) (err error) {
	timeouts := append([]time.Duration{0}, ts...)

	for _, t := range timeouts {
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

func Query[T any](fn func() (T, error), ts ...time.Duration) (v T, err error) {
	timeouts := append([]time.Duration{0}, ts...)

	for _, t := range timeouts {
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

func linear() []time.Duration {
	n := 10

	res := make([]time.Duration, n)
	for i := 0; i < n; i++ {
		res[i] = 1 * time.Second
	}

	return res
}

func exponential() []time.Duration {
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
