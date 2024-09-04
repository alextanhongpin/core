// package request handles the parsing and validation for the request body.
package request

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type validatable interface {
	Valid() error
}

// DecodeJSON decodes the json to struct and performs validation.
func DecodeJSON[T validatable](r *http.Request, v T) error {
	// Duplicate the request to a buffer.
	var buf bytes.Buffer
	rr := io.TeeReader(r.Body, &buf)
	if err := json.NewDecoder(rr).Decode(&v); err != nil && !errors.Is(err, io.EOF) {
		// Set back to the body as if it was never read before.
		// This allows us to log the request body.
		r.Body = io.NopCloser(&buf)

		return err
	}
	r.Body = io.NopCloser(&buf)

	return v.Valid()
}
