package circuitbreaker

import (
	"context"
	"errors"
	"io"
	"net/http"
)

type Transporter struct {
	rt http.RoundTripper
	cb circuitbreaker
}

func NewTransporter(rt http.RoundTripper, cb circuitbreaker) *Transporter {
	if rt == nil {
		rt = http.DefaultTransport
	}
	return &Transporter{
		rt: rt,
		cb: cb,
	}
}

func (t *Transporter) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	err = t.cb.Do(func() error {
		resp, err = t.rt.RoundTrip(r)
		if err != nil {
			// The caller giving up says nothing about the dependency.
			if errors.Is(err, context.Canceled) && errors.Is(r.Context().Err(), context.Canceled) {
				return neutralError{err}
			}
			return err
		}
		if resp != nil && resp.StatusCode >= http.StatusInternalServerError {
			return errors.New(resp.Status)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrOpened) && r.Body != nil {
			// RoundTrip must close the request body even when it never sends.
			r.Body.Close()
		}
		if resp != nil && resp.Body != nil {
			// Drain a bounded amount so the connection can be reused.
			io.CopyN(io.Discard, resp.Body, 4<<10)
			resp.Body.Close()
		}
		return nil, err
	}

	return
}
