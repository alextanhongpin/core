package request

import (
	"bytes"
	"io"
	"net/http"
)

func Read(r *http.Request) ([]byte, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(bytes.NewReader(b))
	return b, nil
}

func Clone(r *http.Request) (*http.Request, error) {
	b, err := Read(r)
	if err != nil {
		return nil, err
	}

	rc := r.Clone(r.Context())
	rc.Body = io.NopCloser(bytes.NewBuffer(b))

	return rc, nil
}
