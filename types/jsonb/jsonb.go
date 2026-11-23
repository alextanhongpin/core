package jsonb

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

var (
	// ErrSkip, returned by a Reviver callback, stops the traversal. Reviver
	// then returns nil.
	ErrSkip = errors.New("jsonb: skip")

	// ErrDelete, returned by a Reviver callback, removes the visited value
	// from its parent map or array. The root value cannot be deleted.
	ErrDelete = errors.New("jsonb: delete")

	// ErrType is returned by Set when the existing value has a different Go
	// type from the new value.
	ErrType = errors.New("jsonb: type mismatch")
)

// Path is the location of a value, one segment per map key. Array elements
// add an index suffix to the last segment, for example "items[0]", or
// "[0]" for an element of a root array.
type Path []string

// Match reports whether p matches pattern. Pattern segments are separated by
// dots. A "*" segment matches any one segment, and "[]" matches any array
// index, so "items[].name" matches "items[3].name".
func (p Path) Match(pattern string) bool {
	segments := strings.Split(pattern, ".")
	if len(segments) != len(p) {
		return false
	}
	for i, want := range segments {
		if want != "*" && !matchSegment(want, p[i]) {
			return false
		}
	}
	return true
}

// matchSegment compares one segment, letting each "[]" in want match a
// "[<digits>]" in got.
func matchSegment(want, got string) bool {
	for want != "" {
		if rest, ok := strings.CutPrefix(want, "[]"); ok {
			if !strings.HasPrefix(got, "[") {
				return false
			}
			end := strings.IndexByte(got, ']')
			if end < 2 || strings.Trim(got[1:end], "0123456789") != "" {
				return false
			}
			want, got = rest, got[end+1:]
			continue
		}
		if got == "" || want[0] != got[0] {
			return false
		}
		want, got = want[1:], got[1:]
	}
	return got == ""
}

// String joins the segments with dots.
func (p Path) String() string {
	return strings.Join(p, ".")
}

// Base returns the last segment, or false for the root path.
func (p Path) Base() (string, bool) {
	if len(p) != 0 {
		return p[len(p)-1], true
	}
	return "", false
}

func (p Path) index(i int) Path {
	if len(p) == 0 {
		return Path{fmt.Sprintf("[%d]", i)}
	}
	path := slices.Clone(p)
	path[len(path)-1] += fmt.Sprintf("[%d]", i)
	return path
}

func (p Path) key(k string) Path {
	return append(slices.Clip(p), k)
}

// Delete removes every map entry and array element whose path matches
// pattern.
func Delete(a any, pattern string) error {
	return Reviver(a, func(path Path, val any) (any, error) {
		if path.Match(pattern) {
			return nil, ErrDelete
		}
		return val, nil
	})
}

// Set replaces every value whose path matches pattern with value. Only
// existing values are replaced; it returns an error wrapping ErrType when a
// matched value has a different Go type, such as float64 for a decoded JSON
// number.
func Set(a any, pattern string, value any) error {
	typ := reflect.TypeOf(value)
	return Reviver(a, func(path Path, val any) (any, error) {
		if !path.Match(pattern) {
			return val, nil
		}
		if reflect.TypeOf(val) != typ {
			return nil, fmt.Errorf("%w: cannot set %T at %s holding %T", ErrType, value, path, val)
		}
		return value, nil
	})
}

// Get returns the values whose paths match pattern, in traversal order. Map
// keys are visited in random order.
func Get(a any, pattern string) ([]any, error) {
	var res []any
	err := Reviver(a, func(path Path, value any) (any, error) {
		if path.Match(pattern) {
			res = append(res, value)
		}
		return value, nil
	})
	return res, err
}

// Reviver walks a, which should hold map[string]any and []any values as
// produced by encoding/json, and calls fn for every value, children before
// their parents and the root last. The value fn returns replaces the visited
// value in place. Returning ErrDelete removes it, and ErrSkip stops the
// traversal without an error. Changes made before an error are kept.
func Reviver(a any, fn func(path Path, value any) (any, error)) error {
	var next func(Path, any) (any, error)
	next = func(path Path, a any) (any, error) {
		switch m := a.(type) {
		case []any:
			var drop []int
			for i, v := range m {
				n, err := next(path.index(i), v)
				if errors.Is(err, ErrDelete) && len(path) > 0 {
					drop = append(drop, i)
					continue
				}
				if err != nil {
					return m, err
				}
				m[i] = n
			}
			if len(drop) > 0 {
				kept := make([]any, 0, len(m)-len(drop))
				for i, v := range m {
					if len(drop) > 0 && drop[0] == i {
						drop = drop[1:]
						continue
					}
					kept = append(kept, v)
				}
				m = kept
			}
			return fn(path, m)
		case map[string]any:
			for k, v := range m {
				n, err := next(path.key(k), v)
				if errors.Is(err, ErrDelete) {
					delete(m, k)
					continue
				}
				if err != nil {
					return m, err
				}
				m[k] = n
			}
			return fn(path, m)
		default:
			return fn(path, a)
		}
	}

	_, err := next(nil, a)
	switch {
	case errors.Is(err, ErrSkip):
		return nil
	case errors.Is(err, ErrDelete):
		return fmt.Errorf("jsonb: cannot delete the root value or an element of a root array: %w", err)
	}
	return err
}
