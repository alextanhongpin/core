package internal

import "reflect"

func IsStruct(v any) bool {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t.Kind() == reflect.Struct
}
