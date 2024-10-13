package circuitbreaker

import (
	"errors"
	"net/http"
)

type transporter interface {
	RoundTrip(*http.Request) (*http.Response, error)
}

type circuitbreaker interface {
	Do(fn func() error) error
}

type Transporter struct {
	Transport      transporter
	CircuitBreaker circuitbreaker
}

func NewTransporter(t transporter, cb circuitbreaker) *Transporter {
	return &Transporter{
		Transport:      t,
		CircuitBreaker: cb,
	}
}

func (t *Transporter) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	err = t.CircuitBreaker.Do(func() error {
		resp, err = t.Transport.RoundTrip(r)
		if err != nil {
			return err
		}

		if resp != nil && resp.StatusCode >= http.StatusInternalServerError {
			return errors.New(resp.Status)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}
