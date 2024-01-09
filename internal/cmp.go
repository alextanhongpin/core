package internal

import (
	"reflect"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func IgnoreMapEntries(keys ...string) cmp.Option {
	return cmpopts.IgnoreMapEntries(func(k string, v any) bool {
		for _, key := range keys {
			if key == k {
				return true
			}
		}

		return false
	})
}

func GetStructTags(a any, fn func(tag reflect.StructField) error) error {
	c := make(map[any]bool)

	var get func(a any) error
	get = func(a any) error {
		if a == nil {
			return nil
		}

		t := reflect.TypeOf(a)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			t = t.Elem()
		}
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		if t.Kind() == reflect.Map {
			iter := reflect.New(t).Elem().MapRange()
			for iter.Next() {
				get(iter.Value())
			}

			return nil
		}

		if t.Kind() != reflect.Struct {
			return nil
		}

		// Prevent recursive type.
		if c[t] {
			return nil
		}

		c[t] = true

		for _, f := range reflect.VisibleFields(t) {
			if err := fn(f); err != nil {
				return err
			}

			nt := reflect.New(f.Type).Elem().Interface()
			if err := get(nt); err != nil {
				return err
			}
		}

		return nil
	}

	return get(a)
}
