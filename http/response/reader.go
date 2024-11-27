package response

import (
	"bytes"
	"io"
	"net/http"
)

func Read(w *http.Response) ([]byte, error) {
	b, err := io.ReadAll(w.Body)
	if err != nil {
		return nil, err
	}

	w.Body = io.NopCloser(bytes.NewReader(b))
	return b, nil
}
