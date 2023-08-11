package maputil

import (
	"fmt"
	"strings"

	"golang.org/x/exp/slices"
)

// AllKeys returns all the keys from map[string]any.
func AllKeys(m map[string]any) []string {
	var fields []string

	var visitor func(k string, v any)
	visitor = func(k string, v any) {
		if !strings.HasSuffix(k, "[_]") {
			fields = append(fields, k)
		}

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

func Invert[K, V comparable](m map[K]V) map[V]K {
	res := make(map[V]K)
	for k, v := range m {
		res[v] = k
	}

	return res
}

func GroupBy[K comparable, V any](v []V, fn func(i int) K) map[K][]V {
	m := make(map[K][]V)
	for i := range v {
		k := fn(i)
		m[k] = append(m[k], v[i])
	}

	return m
}
