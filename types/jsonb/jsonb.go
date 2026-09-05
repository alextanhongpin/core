package jsonb

import (
	"errors"
	"strings"
)

var ErrSkip = errors.New("skip")

type Path []string

func (p Path) String() string {
	return strings.Join(p, ".")
}

func (p Path) Base() (string, bool) {
	if len(p) != 0 {
		return p[len(p)-1], true
	}
	return "", false
}

func Reviver(a any, fn func(path Path, value any) error) error {
	var next func(Path, any) error
	next = func(paths Path, a any) error {
		switch m := a.(type) {
		case []any:
			if len(paths) > 0 {
				last := paths[len(paths)-1]
				paths[len(paths)-1] = last + "[]"
			}
			for _, v := range m {
				if err := next(paths, v); err != nil {
					if errors.Is(err, ErrSkip) {
						return nil
					}
					return err
				}
			}
			return nil
		case map[string]any:
			for k, v := range m {
				nk := append(paths, k)
				if err := next(nk, v); err != nil {
					if errors.Is(err, ErrSkip) {
						return nil
					}
					return err
				}
			}

			return nil
		default:
			return fn(paths, a)
		}
	}

	return next(nil, a)
}
