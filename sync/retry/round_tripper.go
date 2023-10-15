package retry

import (
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
	r.ShouldHandle = func(w *http.Response, err error) bool {
		// Retry when status code is 5XX.
		if w != nil && w.StatusCode >= http.StatusInternalServerError {
			return true
		}

		return err != nil
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
