package structs

import (
	"fmt"
	"reflect"
	"slices"
)

// ZeroField sets the fields to its zero value.
// It accepts a struct pointer, a slice of structs, or a slice of struct pointers.
func ZeroField(input any, fields ...string) error {
	if len(fields) == 0 {
		return fmt.Errorf("fields cannot be empty")
	}

	v := reflect.ValueOf(input)
	return walkValue(v, fields)
}

func walkValue(v reflect.Value, fields []string) error {
	// Automatically unwrap pointers and interfaces
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil // Cannot traverse a nil pointer
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice:
		// If it's a slice, apply the logic to every item in the slice
		for i := range v.Len() {
			if err := walkValue(v.Index(i), fields); err != nil {
				return err
			}
		}
		return nil

	case reflect.Struct:
		for f, val := range v.Fields() {
			if err := walkValue(val, fields); err != nil {
				return err
			}

			if slices.Contains(fields, f.Name) {
				if !val.CanSet() {
					return fmt.Errorf("field %s cannot be set (must be exported and passed by pointer)", f.Name)
				}
				val.Set(reflect.Zero(val.Type()))
			}
		}
		return nil
	default:
		return nil
	}
}
