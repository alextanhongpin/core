package cache

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"reflect"
)

type GobEncoder struct {
}

func NewGobEncoder() *GobEncoder {
	return &GobEncoder{}
}

func (g *GobEncoder) Marshal(v any) ([]byte, error) {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Pointer {
		// Register non pointer.
		gob.Register(v)
		v = val.Addr().Interface()
	} else {
		// Register non-pointer struct.
		gob.Register(val.Elem())
	}

	var b bytes.Buffer
	// v must be pointer.
	err := gob.NewEncoder(&b).Encode(v)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (g *GobEncoder) Unmarshal(b []byte, v any) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Pointer {
		// Register non-pointer struct.
		gob.Register(val.Elem())
	}
	// v must be pointer.
	return gob.NewDecoder(bytes.NewBuffer(b)).Decode(v)
}

type JSONEncoder struct {
}

func NewJSONEncoder() *JSONEncoder {
	return &JSONEncoder{}
}

func (j *JSONEncoder) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (g *JSONEncoder) Unmarshal(b []byte, v any) error {
	return json.Unmarshal(b, v)
}
