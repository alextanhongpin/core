package circuitbreaker

import (
	"context"
	"errors"
	"net/http"
)

type transporter interface {
	RoundTrip(*http.Request) (*http.Response, error)
}

type transporterFunc = handler[*http.Request, *http.Response]

type Transporter struct {
	fn transporterFunc
}

func NewTransporter(t transporter, cb func(transporterFunc) transporterFunc) *Transporter {
	return &Transporter{
		fn: cb(func(ctx context.Context, r *http.Request) (*http.Response, error) {
			resp, err := t.RoundTrip(r)
			if err != nil {
				return nil, err
			}

			if resp != nil && resp.StatusCode >= http.StatusInternalServerError {
				return nil, errors.New(resp.Status)
			}

			return resp, nil
		}),
	}
}

func (t *Transporter) RoundTrip(r *http.Request) (*http.Response, error) {
	return t.fn(r.Context(), r)
}
