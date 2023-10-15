package retry

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
	Transport transporter
	Retrier   *Retry[*http.Response]
}

func NewRoundTripper(t transporter) *RoundTripper {
	r := New[*http.Response](NewOption())
	r.ShouldHandle = func(resp *http.Response, err error) (bool, error) {
		// Retry when status code is 5XX.
		if resp != nil && resp.StatusCode >= http.StatusInternalServerError {
			return true, errors.New(resp.Status)
		}

		return err != nil, err
	}

	return &RoundTripper{
		Transport: t,
		Retrier:   r,
	}
}

func (t *RoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, _, err := t.Retrier.Do(func() (*http.Response, error) {
		return t.Transport.RoundTrip(r)
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}
