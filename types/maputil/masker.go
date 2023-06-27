package maputil

import (
	"encoding/json"
	"errors"
)

var ErrInvalidJSON = errors.New("maputil: invalid json")

const MaskValue = "*!REDACTED*"

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
	return MaskBytesFunc(b, MaskFields(fields...))
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
