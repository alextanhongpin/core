// package request handles the parsing and validation for the request body.
package request

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

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
	Valid() error
}

type JSONDecoder struct {
	r      *http.Request
	Logger *slog.Logger
}

func NewJSONDecoder(r *http.Request) *JSONDecoder {
	return &JSONDecoder{
		r:      r,
		Logger: slog.Default(),
	}
}

func (dec *JSONDecoder) Decode(v validatable) error {
	r := dec.r

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return &BodyError{Body: bytes.Clone(b), err: err}
	}

	if !json.Valid(b) {
		return &BodyError{Body: bytes.Clone(b), err: errors.New("non-json payload")}
	}

	buf := bytes.NewBuffer(b)
	r.Body = io.NopCloser(buf)

	if err := json.Unmarshal(b, &v); err != nil {
		return &BodyError{Body: bytes.Clone(b), err: err}
	}

	return v.Valid()

}

// DecodeJSON decodes the json to struct and performs validation.
func DecodeJSON(r *http.Request, v validatable) error {
	return NewJSONDecoder(r).Decode(v)
}
