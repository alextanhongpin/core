package circuitbreaker

import (
	"errors"
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
			return err
		}
		if resp != nil && resp.StatusCode >= http.StatusInternalServerError {
			return errors.New(resp.Status)
		}
		return nil
	})
	if err != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return nil, err
	}

	return
}
