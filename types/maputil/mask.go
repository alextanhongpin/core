package maputil

import (
	"encoding/json"
	"errors"
	"fmt"
)

var ErrInvalidJSON = errors.New("maputil: invalid json")
var ErrMaskKeyNotFound = errors.New("maputil: mask key not found")

const MaskValue = "/* !REDACTED */"

func MaskFunc(m map[string]any, fn func(k string) bool) map[string]any {
	return replaceFunc(m, func(k, v string) string {
		if fn(k) {
			return MaskValue
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

func MaskBytes(b []byte, fields ...string) ([]byte, error) {
	cache := make(map[string]bool)
	for _, f := range fields {
		cache[f] = false
	}

	b, err := MaskBytesFunc(b, func(k string) bool {
		_, ok := cache[k]
		if ok {
			cache[k] = true
		}

		return ok
	})
	if err != nil {
		return nil, err
	}

	for f := range cache {
		if !cache[f] {
			return nil, fmt.Errorf("%w: %q", ErrMaskKeyNotFound, f)
		}
	}

	return b, nil
}

func MaskBytesFunc(b []byte, fn func(key string) bool) ([]byte, error) {
	if !json.Valid(b) {
		return nil, ErrInvalidJSON
	}

	var a any
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, err
	}

	var recurse func(v any) any
	recurse = func(v any) any {
		switch t := v.(type) {
		case map[string]any:
			return MaskFunc(t, fn)
		case []any:
			res := make([]any, len(t))
			for i, v := range t {
				res[i] = recurse(v)
			}

			return res
		default:
			return t
		}
	}

	return json.Marshal(recurse(a))
}
