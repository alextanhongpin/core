package httputil

import (
	"bytes"
	"io"
	"net/http"
)

func ReadRequest(r *http.Request) ([]byte, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(bytes.NewReader(b))
	return b, nil
}

func ReadResponse(w *http.Response) ([]byte, error) {
	b, err := io.ReadAll(w.Body)
	if err != nil {
		return nil, err
	}

	w.Body = io.NopCloser(bytes.NewReader(b))
	return b, nil
}

func CloneRequest(r *http.Request) (*http.Request, error) {
	b, err := ReadRequest(r)
	if err != nil {
		return nil, err
	}

	rc := r.Clone(r.Context())
	rc.Body = io.NopCloser(bytes.NewBuffer(b))

	return rc, nil
}
