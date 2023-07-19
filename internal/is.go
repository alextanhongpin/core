package internal

import (
	"fmt"
	"reflect"
	"time"
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

func IsSliceOfStruct(v any) bool {
	if IsStruct(v) {
		return true
	}

	t := reflect.TypeOf(v)

	switch t.Kind() {
	case reflect.Slice:
		return IsStruct(reflect.ValueOf(t.Elem()))
	default:
		return false
	}
}

func IsNonZeroTime(m map[string]any, keys ...string) error {
	for _, key := range keys {
		v, ok := m[key]
		if !ok {
			return fmt.Errorf("key %q not found", key)
		}

		var t time.Time
		if err := t.UnmarshalText([]byte(v.(string))); err != nil {
			return fmt.Errorf("key %q: %w", key, err)
		}

		if t.IsZero() {
			return fmt.Errorf("key %q is zero time", key)
		}
	}

	return nil
}
