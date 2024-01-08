package internal

import (
	"reflect"
	"strings"

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

func GetStructTags[T any](v T, name, tag string) map[string][]string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	kv := make(map[string][]string)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := f.Tag.Get(name)
		if name == "" {
			name = f.Name
		}

		v := f.Tag.Get(tag)
		if v == "" {
			continue
		}
		kv[name] = strings.Split(v, ",")
	}
	return kv
}
