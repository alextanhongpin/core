package circuitbreaker

import (
	"context"
	"errors"
	"net/http"
)

type transporter interface {
	RoundTrip(r *http.Request) (*http.Response, error)
}

type breaker[T any] interface {
	Do(func() (T, error)) (T, error)
}

type RoundTripper struct {
	Transport      transporter
	CircuitBreaker breaker[*http.Response]
}

func NewRoundTripper(t transporter) *RoundTripper {
	opt := NewOption()
	cb := New[*http.Response](opt)
	cb.ShouldHandle = func(resp *http.Response, err error) (bool, error) {
		// Skip if cancelled by caller.
		if errors.Is(err, context.Canceled) {
			return false, err
		}

		if resp != nil && resp.StatusCode >= http.StatusInternalServerError {
			return true, errors.New(resp.Status)
		}

		return err != nil, err
	}

	return &RoundTripper{
		Transport:      t,
		CircuitBreaker: cb,
	}
}

func (t *RoundTripper) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	resp, err = t.CircuitBreaker.Do(func() (*http.Response, error) {
		return t.Transport.RoundTrip(r)
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}
