package structs

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// ZeroPath sets each dot-separated field path, such as "Profile.Age", to its
// zero value. Each segment is a struct field name; pointers and interfaces
// along the way are followed, and slices and arrays apply the rest of the path
// to every element. A nil pointer on the path is skipped.
//
// It returns an error if a segment does not exist, a segment is applied to a
// value that is not a struct, or the target cannot be set (unexported, or
// input not passed by pointer). Paths processed before the error stay changed.
func ZeroPath(input any, fieldPaths ...string) error {
	if len(fieldPaths) == 0 {
		return errors.New("structs: field paths cannot be empty")
	}
	for _, fieldPath := range fieldPaths {
		if fieldPath == "" {
			return errors.New("structs: field path cannot be empty")
		}
		if err := walkPathValue(reflect.ValueOf(input), strings.Split(fieldPath, ".")); err != nil {
			return fmt.Errorf("structs: path %q: %w", fieldPath, err)
		}
	}
	return nil
}

func walkPathValue(v reflect.Value, parts []string) error {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
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
		if len(parts) == 1 {
			if !f.CanSet() {
				return fmt.Errorf("field %s cannot be set (must be exported and passed by pointer)", parts[0])
			}
			f.SetZero()
			return nil
		}
		return walkPathValue(f, parts[1:])

	default:
		return fmt.Errorf("field %s: %s is not a struct", parts[0], v.Type())
	}
}
