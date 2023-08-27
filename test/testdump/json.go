package testdump

import (
	"encoding/json"

	"github.com/alextanhongpin/core/internal"
	"github.com/google/go-cmp/cmp"
)

type JSONOption struct {
	Body []cmp.Option
}

// NOTE: Why using a type is bad - because if we serialize to structs, the keys
// that are removed won't be compared.
// Ideally, using map[string]any or just any should work better for snapshot
// testing.
func JSON[T any](rw readerWriter, t T, opt *JSONOption, hooks ...Hook[T]) error {
	if opt == nil {
		opt = new(JSONOption)
	}

	var s S[T] = &snapshot[T]{
		marshaler: MarshalFunc[T](MarshalJSON[T]),
		// This is only used for custom comparison. It does not benefit as much as
		// using map[string]any for comparison due to loss of information.
		unmarshaler:    UnmarshalFunc[T](UnmarshalJSON[T]),
		anyUnmarshaler: UnmarshalAnyFunc(UnmarshalJSON[any]),
		anyComparer:    CompareAnyFunc((&JSONComparer[any]{Body: opt.Body}).Compare),
	}

	s = Hooks[T](hooks).Apply(s)

	return Snapshot(rw, t, s)
}

func MarshalJSON[T any](t T) ([]byte, error) {
	return internal.PrettyJSON(t)
}

// The problem is here - the unmarshalling actually causes a loss of data.
func UnmarshalJSON[T any](b []byte) (T, error) {
	var t T
	if err := json.Unmarshal(b, &t); err != nil {
		return t, err
	}

	return t, nil
}

type JSONComparer[T any] struct {
	Body []cmp.Option
}

func (c *JSONComparer[T]) Compare(snapshot, received T) error {
	return internal.ANSIDiff(snapshot, received, c.Body...)
}
