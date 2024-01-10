package internal

import (
	"reflect"
)

func IsStruct(v any) bool {
	t := reflect.TypeOf(v)
	// When v is nil.
	if t == nil {
		return false
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t.Kind() == reflect.Struct
}
