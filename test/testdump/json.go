package testdump

import (
	"encoding/json"

	"github.com/alextanhongpin/core/internal"
	"github.com/google/go-cmp/cmp"
)

// NOTE: Why using a type is bad - because if we serialize to structs, the keys
// that are removed won't be compared.
// Ideally, using map[string]any or just any should work better for snapshot
// testing.
func JSON[T any](fileName string, t T, opt *JSONOption[T]) error {
	if opt == nil {
		opt = new(JSONOption[T])
	}

	s := snapshot[T]{
		Marshaller: MarshalFunc[T](MarshalJSON[T]),
		// This is only used for custom comparison. It does not benefit as much as
		// using map[string]any for comparison due to loss of information.
		Unmarshaller: UnmarshalFunc[T](UnmarshalJSON[T]),
		Comparer:     CompareFunc[T](nopComparer[T]),
		// The core logic, unmarshalling into map type and comparing it.
		unmarshalAny: UnmarshalFunc[any](UnmarshalJSON[any]),
		compareAny:   CompareFunc[any](CompareJSON[any](opt.Body...)),
	}

	return Snapshot(fileName, t, &s, opt.Hooks...)
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

func CompareJSON[T any](opts ...cmp.Option) func(a, b T) error {
	return func(snapshot, received T) error {
		return internal.ANSIDiff(snapshot, received, opts...)
	}
}

type JSONOption[T any] struct {
	Hooks []Hook[T]
	Body  []cmp.Option
}
