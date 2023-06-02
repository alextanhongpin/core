// package request handles the parsing and validation for the request body.
package request

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// Validator returns the instance of the global validator.
// Custom validation function can be registered using this instance.
// There is no way to override the validator, to avoid concurrent replacement
// of the global validator.
func Validator() *validator.Validate {
	return validate
}

// Body decodes the json to struct and performs validation.
func Body[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	// Duplicate the request to a buffer.
	var t T
	var buf bytes.Buffer
	rr := io.TeeReader(r.Body, &buf)

	if err := json.NewDecoder(rr).Decode(&t); err != nil && !errors.Is(err, io.EOF) {
		// Set back to the body as if it was never read before.
		// This allows us to log the request body.
		r.Body = io.NopCloser(&buf)

		return t, err
	}

	r.Body = io.NopCloser(&buf)

	if err := validate.Struct(&t); err != nil {
		return t, err
	}

	return t, nil
}
