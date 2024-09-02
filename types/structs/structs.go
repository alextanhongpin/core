package structs

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// PkgName returns the name of the type of the value.
func PkgName(v any) string {
	// Reflect will panic if there is no
	// type detected, which is usually
	// the case for nil types.
	if v == nil {
		return "nil"
	}

	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		return t.String()
	}

	return t.Kind().String()
}

// Name returns the type name without the package name.
func Name(v any) string {
	parts := strings.Split(PkgName(v), ".")
	return parts[len(parts)-1]
}

// NonZero checks if all fields of a struct are non-zero.
func NonZero(v any) error {
	a, err := toMap(v)
	if err != nil {
		return err
	}
	var validate func(string, any) error
	validate = func(key string, v any) error {
		switch m := v.(type) {
		case map[string]any:
			if len(m) == 0 {
				return newFieldError(key)
			}
			for k, v := range m {
				if err := validate(join(key, k), v); err != nil {
					return err
				}
			}
		case []any:
			if len(m) == 0 {
				return newFieldError(key)
			}
			for i, v := range m {
				if err := validate(join(key, fmt.Sprintf("[%d]", i)), v); err != nil {
					return err
				}
			}
		default:
			if isEmpty(m) {
				return newFieldError(key)
			}
		}

		return nil
	}

	return validate(PkgName(v), a)
}

func join(vs ...string) string {
	var sb strings.Builder
	for _, v := range vs {
		if v == "" {
			continue
		}
		sb.WriteString(v)
		sb.WriteString(".")
	}
	return strings.TrimSuffix(sb.String(), ".")
}

func isEmpty(v any) bool {
	return v == nil || v == 0.0 || v == false || v == ""
}

func toMap(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var a any
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, err
	}
	return a, nil
}

type FieldError struct {
	Path  string
	Field string
}

func newFieldError(path string) error {
	parts := strings.Split(path, ".")
	key := parts[len(parts)-1]
	return &FieldError{Path: path, Field: key}
}

func (k *FieldError) Error() string {
	return fmt.Sprintf("field %q is empty", k.Field)
}
