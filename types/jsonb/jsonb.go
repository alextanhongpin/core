package jsonb

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

var (
	ErrSkip = errors.New("skip")
	ErrType = errors.New("type")
)

type Path []string

func (p Path) Match(pattern string) bool {
	paths := strings.Split(pattern, ".")
	if len(paths) != len(p) {
		return false
	}
	for i := range len(paths) {
		want := paths[i]
		got := p[i]
		// Match wildcard.
		if want == "*" {
			continue
		}
		// Check equality.
		if want != got {
			return false
		}
	}
	return true
}

func (p Path) String() string {
	return strings.Join(p, ".")
}

func (p Path) AsArrayIndex(i int) Path {
	if len(p) > 0 {
		path := slices.Clone(p)
		path[len(path)-1] += fmt.Sprintf("[%d]", i)
		return path
	}
	return Path{fmt.Sprintf("[%d]", i)}
}

func (p Path) AsArray() Path {
	if len(p) > 0 {
		path := slices.Clone(p)
		path[len(path)-1] += "[]"
		return path
	}
	return Path{"[]"}
}

func (p Path) Base() (string, bool) {
	if len(p) != 0 {
		return p[len(p)-1], true
	}
	return "", false
}

func Set(a any, key string, value any) error {
	typ := reflect.TypeOf(value)
	return Reviver(a, func(path Path, val any) (any, error) {
		if path.Match(key) {
			if reflect.TypeOf(val) == typ {
				return value, nil
			}
			return nil, fmt.Errorf("reviver: %w: cannot set %T to key %s of type %T", ErrType, value, key, val)
		}
		return val, nil
	})
}

func Get(a any, key string) ([]any, error) {
	var res []any
	err := Reviver(a, func(path Path, value any) (any, error) {
		if path.Match(key) {
			res = append(res, value)
		}
		return value, nil
	})

	return res, err
}

func Reviver(a any, fn func(path Path, value any) (any, error)) error {
	var next func(Path, any) (any, error)
	next = func(path Path, a any) (any, error) {
		switch m := a.(type) {
		case []any:
			arr := path.AsArray()
			for i, v := range m {
				n, err := next(arr, v)
				if err != nil {
					return m, err
				}
				m[i] = n

				o, err := next(path.AsArrayIndex(i), n)
				if err != nil {
					return m, err
				}
				m[i] = o
			}

			return fn(path, m)
		case map[string]any:
			for k, v := range m {
				nk := append(path, k)
				nv, err := next(nk, v)
				if err != nil {
					return m, err
				}
				if nv == nil {
					delete(m, k)
				} else {
					m[k] = nv
				}
			}

			return fn(path, m)
		default:
			return fn(path, a)
		}
	}

	a, err := next(nil, a)
	if errors.Is(err, ErrSkip) {
		return nil
	}
	return err
}
