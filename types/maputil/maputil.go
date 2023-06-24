package maputil

import (
	"fmt"

	"golang.org/x/exp/slices"
)

func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

// AllKeys returns all the keys from map[string]any.
func AllKeys(m map[string]any) []string {
	var fields []string

	var visitor func(k string, v any)
	visitor = func(k string, v any) {
		fields = append(fields, k)

		switch t := v.(type) {
		case map[string]any:
			for kk, vv := range t {
				key := fmt.Sprintf("%s.%s", k, kk)
				visitor(key, vv)
			}
		case []any:
			for _, v := range t {
				// We don't care about the indices, only the keys.
				key := fmt.Sprintf("%s[_]", k)
				visitor(key, v)
			}
		default:
		}
	}

	for k, v := range m {
		visitor(k, v)
	}

	slices.Sort(fields)
	return slices.Compact(fields)
}

func Values[K comparable, V any](m map[K]V) []V {
	vals := make([]V, 0, len(m))
	for _, v := range m {
		vals = append(vals, v)
	}

	return vals
}

func Invert[K, V comparable](m map[K]V) map[V]K {
	res := make(map[V]K)
	for k, v := range m {
		res[v] = k
	}

	return res
}
