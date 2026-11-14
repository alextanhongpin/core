package cache

import (
	"encoding/gob"
	"encoding/json"
	"io"
)

// Codec turns values into bytes and back. [JSONCodec] is the default;
// [GobCodec] is also provided. Implementations must be safe for concurrent
// use.
type Codec interface {
	NewEncoder(w io.Writer) Encoder
	NewDecoder(r io.Reader) Decoder
}

// Encoder writes an encoded value. *json.Encoder and *gob.Encoder satisfy it.
type Encoder interface {
	Encode(v any) error
}

// Decoder reads an encoded value into v. *json.Decoder and *gob.Decoder
// satisfy it.
type Decoder interface {
	Decode(v any) error
}

// JSONCodec encodes values with encoding/json. It is the default Codec.
type JSONCodec struct{}

// NewJSONCodec returns a JSONCodec.
func NewJSONCodec() *JSONCodec {
	return new(JSONCodec)
}

// NewEncoder returns a json.Encoder writing to w.
func (j JSONCodec) NewEncoder(w io.Writer) Encoder {
	return json.NewEncoder(w)
}

// NewDecoder returns a json.Decoder reading from r.
func (j JSONCodec) NewDecoder(r io.Reader) Decoder {
	return json.NewDecoder(r)
}

// GobCodec encodes values with encoding/gob. Unlike JSON it supports
// non-string map keys, but the bytes are only readable from Go.
type GobCodec struct{}

// NewGobCodec returns a GobCodec.
func NewGobCodec() *GobCodec {
	return new(GobCodec)
}

// NewEncoder returns a gob.Encoder writing to w.
func (g GobCodec) NewEncoder(w io.Writer) Encoder {
	return gob.NewEncoder(w)
}

// NewDecoder returns a gob.Decoder reading from r.
func (g GobCodec) NewDecoder(r io.Reader) Decoder {
	return gob.NewDecoder(r)
}
