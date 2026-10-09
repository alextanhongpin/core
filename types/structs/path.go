package structs

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// ZeroPath sets a nested field path (e.g., "User.Profile.Age") to its zero value.
// It accepts a struct pointer, a slice of structs, or a slice of struct pointers.
func ZeroPath(input any, fieldPaths ...string) error {
	if len(fieldPaths) == 0 {
		return errors.New("field paths cannot be empty")
	}
	for _, fieldPath := range fieldPaths {
		parts := strings.Split(fieldPath, ".")
		if len(parts) == 0 || fieldPath == "" {
			return fmt.Errorf("field path cannot be empty")
		}

		v := reflect.ValueOf(input)
		if err := walkPathValue(v, parts); err != nil {
			return err
		}
	}
	return nil
}

func walkPathValue(v reflect.Value, parts []string) error {
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
			if err := walkPathValue(v.Index(i), parts); err != nil {
				return err
			}
		}
		return nil

	case reflect.Struct:
		f := v.FieldByName(parts[0])
		if !f.IsValid() {
			return fmt.Errorf("field %s does not exist", parts[0])
		}

		// If this is the final target field, zero it out
		if len(parts) == 1 {
			if !f.CanSet() {
				return fmt.Errorf("field %s cannot be set (must be exported and passed by pointer)", parts[0])
			}
			f.Set(reflect.Zero(f.Type()))
			return nil
		}

		// If there are more parts, drill deeper into the nested structure
		return walkPathValue(f, parts[1:])

	default:
		return nil
	}
}
