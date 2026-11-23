package structs

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
)

// ZeroField sets every struct field named in fields to its zero value, at any
// depth: it descends through pointers, interfaces, struct fields, slices, and
// arrays, so ZeroField(&users, "Password") clears Password in every element
// and in nested structs. Map values are not visited. Use ZeroPath to target a
// single nested field instead.
//
// input must be a pointer (or a slice of structs or pointers) for fields to
// be settable. A matching field that cannot be set, such as an unexported
// one, is an error; changes made before the error are kept.
func ZeroField(input any, fields ...string) error {
	if len(fields) == 0 {
		return errors.New("structs: fields cannot be empty")
	}

	return walkValue(reflect.ValueOf(input), fields, make(map[uintptr]bool))
}

func walkValue(v reflect.Value, fields []string, seen map[uintptr]bool) error {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		if v.Kind() == reflect.Pointer {
			// Guard against cycles such as self-referencing structs.
			if seen[v.Pointer()] {
				return nil
			}
			seen[v.Pointer()] = true
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if err := walkValue(v.Index(i), fields, seen); err != nil {
				return err
			}
		}
		return nil

	case reflect.Struct:
		for f, val := range v.Fields() {
			if slices.Contains(fields, f.Name) {
				if !val.CanSet() {
					return fmt.Errorf("structs: field %s cannot be set (must be exported and passed by pointer)", f.Name)
				}
				val.SetZero()
				continue
			}
			if err := walkValue(val, fields, seen); err != nil {
				return err
			}
		}
		return nil

	default:
		return nil
	}
}
