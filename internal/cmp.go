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

func IgnoreMapEntriesFromTags[T any](v T) (cmp.Option, bool) {
	ignore := ignoreMapEntriesFromTags(v)
	return cmpopts.IgnoreMapEntries(func(k string, v any) bool {
		return ignore[k]
	}), len(ignore) > 0
}

func ignoreMapEntriesFromTags[T any](v T) map[string]bool {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	ignore := make(map[string]bool)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := f.Tag.Get("json")
		if name == "" {
			name = f.Name
		}

		if strings.HasSuffix(f.Tag.Get("cmp"), ",ignore") {
			ignore[name] = true
		}
	}
	return ignore
}
