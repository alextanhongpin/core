// package request handles the parsing and validation for the request body.
package request

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var ErrInvalidJSON = errors.New("request: invalid json")

type BodyError struct {
	Body []byte
	err  error
}

func (b *BodyError) Unwrap() error {
	return b.err
}

func (b *BodyError) Error() string {
	return b.err.Error()
}

type validatable interface {
	Validate() error
}

// DecodeJSON decodes the json to struct and performs validation.
func DecodeJSON(r *http.Request, v any) error {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	r.Body = io.NopCloser(bytes.NewBuffer(bytes.Clone(b)))

	if !json.Valid(b) {
		return &BodyError{Body: b, err: ErrInvalidJSON}
	}

	if err := json.Unmarshal(b, &v); err != nil {
		return &BodyError{Body: b, err: err}
	}

	switch t := v.(type) {
	case validatable:
		return t.Validate()
	default:
		return nil
	}
}
