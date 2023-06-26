package maputil

import (
	"fmt"
	"strings"
)

type JSONType interface {
	float64 | bool | string
}

// ReplaceFunc iterates over every key-value pairs that
// matches the type.
// Null values will be skipped.
func ReplaceFunc[T JSONType](m map[string]any, fn func(k string, v T) T) map[string]any {
	return replaceFunc(m, fn)
}

func Mask(m map[string]any, fn func(k string) bool) map[string]any {
	return replaceFunc(m, func(k, v string) string {
		if fn(k) {
			return "*!REDACTED*"
		}

		return v
	})
}

func MaskFields(fields ...string) func(k string) bool {
	return func(k string) bool {
		for _, f := range fields {
			if f == k {
				return true
			}
		}

		return false
	}
}

func replaceFunc[T any](m map[string]any, fn func(k string, v T) T) map[string]any {
	var transformer func(string, any) any
	transformer = func(k string, v any) any {
		switch t := v.(type) {
		case map[string]any:
			res := make(map[string]any)
			for kk, vv := range t {
				key := fmt.Sprintf("%s.%s", k, kk)
				res[kk] = transformer(key, vv)
			}

			return res
		case []any:
			res := make([]any, len(t))
			for i, v := range t {
				// We don't care about the indices, only the keys.
				key := fmt.Sprintf("%s[_]", k)
				res[i] = transformer(key, v)
			}
			return res
		case T:
			if !strings.HasSuffix(k, "[_]") {
				return fn(k, t)
			}

			return v
		default:
			return v
		}
	}

	res := make(map[string]any)
	for k, v := range m {
		res[k] = transformer(k, v)
	}

	return res
}
