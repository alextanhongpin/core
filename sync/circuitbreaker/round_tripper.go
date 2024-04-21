package circuitbreaker

import (
	"context"
	"errors"
	"net/http"
)

type transporter interface {
	RoundTrip(r *http.Request) (*http.Response, error)
}

type breaker interface {
	Do(ctx context.Context, key string, fn func() error) error
}

type RoundTripper struct {
	t              transporter
	cb             breaker
	KeyFromRequest func(*http.Request) string
}

func NewRoundTripper(t transporter, cb breaker) *RoundTripper {
	return &RoundTripper{
		t:  t,
		cb: cb,
		KeyFromRequest: func(r *http.Request) string {
			rc := r.Clone(r.Context())
			rc.URL.RawQuery = ""
			rc.URL.Fragment = ""
			return rc.URL.String()
		},
	}
}

func (t *RoundTripper) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	cbErr := t.cb.Do(r.Context(), t.KeyFromRequest(r), func() error {
		resp, err = t.t.RoundTrip(r)
		if err != nil {
			return err
		}

		// Ignore context cancellation.
		if errors.Is(err, context.Canceled) {
			return nil
		}

		// Ignore non-5xx errors.
		if resp != nil && resp.StatusCode >= http.StatusInternalServerError {
			return errors.New(resp.Status)
		}

		return err
	})

	allErr := errors.Join(err, cbErr)
	if allErr != nil {
		return nil, allErr
	}

	return resp, nil
}
