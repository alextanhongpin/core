package cache

import (
	"encoding/gob"
	"encoding/json"
	"io"
)

// Codec defines a unified interface for any serialization format
type Codec interface {
	NewEncoder(w io.Writer) Encoder
	NewDecoder(r io.Reader) Decoder
}

type Encoder interface {
	Encode(v any) error
}

type Decoder interface {
	Decode(v any) error
}

type JSONCodec struct{}

func NewJSONCodec() *JSONCodec {
	return new(JSONCodec)
}

func (j JSONCodec) NewEncoder(w io.Writer) Encoder {
	return json.NewEncoder(w)
}
func (j JSONCodec) NewDecoder(r io.Reader) Decoder {
	return json.NewDecoder(r)
}

type GobCodec struct{}

func NewGobCodec() *GobCodec {
	return new(GobCodec)
}

func (g GobCodec) NewEncoder(w io.Writer) Encoder {
	return gob.NewEncoder(w)
}
func (g GobCodec) NewDecoder(r io.Reader) Decoder {
	return gob.NewDecoder(r)
}
