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
