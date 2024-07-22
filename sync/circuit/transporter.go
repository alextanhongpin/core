package circuit

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
	t  transporter
	cb circuitbreaker
}

func NewTransporter(t transporter, cb circuitbreaker) *Transporter {
	return &Transporter{
		t:  t,
		cb: cb,
	}
}

func (t *Transporter) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	resp, err = t.t.RoundTrip(r)
	if err != nil {
		return nil, err
	}

	if resp != nil && resp.StatusCode >= http.StatusInternalServerError {
		cbErr := t.cb.Do(func() error {
			return errors.New(resp.Status)
		})
		if errors.Is(cbErr, ErrBrokenCircuit) {
			return nil, cbErr
		}
	}

	return resp, nil
}
