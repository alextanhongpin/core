package retry

import (
	"context"
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
	Transport transporter
	Backoffs  Backoffs
}

func (t *RoundTripper) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	err = t.Backoffs.Exec(r.Context(), func(ctx context.Context) error {
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

	return
}
