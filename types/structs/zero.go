package structs

import "reflect"

func IsNilOrZero(x any) bool {
	if x == nil {
		return true
	}

	v := reflect.ValueOf(x)
	switch v.Kind() {
	// Check types that can legally be nil
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		if v.IsNil() {
			return true
		}
	}

	// Check if it's the default zero value for its type (works for structs, ints, etc.)
	return v.IsZero()
}
