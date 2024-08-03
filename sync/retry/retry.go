// Package retry implements functions for DoFunc mechanism.
package retry

import (
	"errors"
	"math"
	"math/rand"
	"time"
)

var ErrRetryLimitExceeded = errors.New("retry: limit exceeded")

type PolicyFunc func(i int) time.Duration

type Retry struct {
	Attempts int
	Policy   PolicyFunc
}

func New(n int) *Retry {
	return &Retry{
		Attempts: n,
		Policy:   ExponentialBackoff(100*time.Millisecond, 1*time.Minute, true),
	}
}

func (r *Retry) Do(fn func() error) error {
	return DoFunc(r.Attempts, r.Policy, fn)
}

func DoFunc(n int, p PolicyFunc, fn func() error) (err error) {
	for i := range n {
		time.Sleep(p(i))

		err = fn()
		if err == nil {
			return nil
		}

		var abortErr *AbortError
		if errors.As(err, &abortErr) {
			return abortErr.Unwrap()
		}
	}

	return errors.Join(ErrRetryLimitExceeded, err)
}

func DoFunc2[T any](n int, p PolicyFunc, fn func() (T, error)) (res T, err error) {
	for i := range n {
		time.Sleep(p(i))

		res, err = fn()
		if err == nil {
			return res, nil
		}

		var abortErr *AbortError
		if errors.As(err, &abortErr) {
			return res, abortErr.Unwrap()
		}
	}

	return res, errors.Join(ErrRetryLimitExceeded, err)
}

func Abort(err error) *AbortError {
	return &AbortError{err}
}

type AbortError struct {
	err error
}

func (e *AbortError) Error() string {
	return e.err.Error()
}

func (e *AbortError) Unwrap() error {
	return e.err
}

func ExponentialBackoff(base, cap time.Duration, jitter bool) func(attempts int) time.Duration {
	b := float64(base)
	c := float64(cap)

	return func(attempts int) time.Duration {
		if attempts <= 0 {
			return 0
		}

		a := float64(attempts)
		j := 1.0
		if jitter {
			j += rand.Float64()
		}
		e := math.Pow(2, a)

		return time.Duration(min(c, j*b*e))
	}
}
