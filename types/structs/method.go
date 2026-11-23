package structs

import (
	"errors"
	"reflect"
)

// GetMethodNames returns the exported method names of v's pointer type, in
// sorted order. This is the union of value- and pointer-receiver methods,
// whether v is a T or a *T. It returns an error if v is nil.
func GetMethodNames(v any) ([]string, error) {
	if v == nil {
		return nil, errors.New("structs: value is nil")
	}
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Pointer {
		t = reflect.PointerTo(t)
	}
	names := make([]string, 0, t.NumMethod())
	for m := range t.Methods() {
		names = append(names, m.Name)
	}
	return names, nil
}
