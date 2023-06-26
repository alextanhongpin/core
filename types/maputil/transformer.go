package maputil

import (
	"encoding/json"
	"errors"
	"reflect"
)

var ErrNonStruct = errors.New("maputil: cannot convert non-struct to map")

func StructToMap(v any) (map[string]any, error) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, ErrNonStruct
	}

	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	return m, nil
}
