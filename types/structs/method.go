package structs

import (
	"fmt"
	"reflect"
	"slices"
)

// GetMethodNames returns all exported method names for a struct or pointer to struct.
func GetMethodNames(v any) ([]string, error) {
	if v == nil {
		return nil, fmt.Errorf("value is nil")
	}
	t := reflect.TypeOf(v)
	// If pointer, get the element type for value methods
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	var names []string
	// Collect methods from both value and pointer receiver
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		if m.IsExported() {
			names = append(names, m.Name)
		}
	}
	// Also check pointer type for pointer receiver methods
	pt := reflect.TypeOf(v)
	if pt.Kind() != reflect.Ptr {
		pt = reflect.PointerTo(t)
	}
	for i := 0; i < pt.NumMethod(); i++ {
		m := pt.Method(i)
		if m.IsExported() && !slices.Contains(names, m.Name) {
			names = append(names, m.Name)
		}
	}
	return names, nil
}
