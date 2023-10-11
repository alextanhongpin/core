package circuitbreaker

import (
	"errors"
	"net/http"
)

type transporter interface {
	RoundTrip(r *http.Request) (*http.Response, error)
}

type breaker interface {
	Do(func() error) error
}

type RoundTripper struct {
	Transport      transporter
	CircuitBreaker breaker
}

func (t *RoundTripper) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	err = t.CircuitBreaker.Do(func() error {
		resp, err = t.Transport.RoundTrip(r)
		if err != nil {
			return err
		}

		// NOTE: Create your own implementation here.
		if resp.StatusCode >= http.StatusInternalServerError {
			return errors.New(resp.Status)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return
}
