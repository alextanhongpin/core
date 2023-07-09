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
func JSON(fileName string, t any, opt *JSONOption) error {
	if opt == nil {
		opt = new(JSONOption)
	}

	type T = any

	s := snapshot[T]{
		Marshaller:   MarshalFunc[T](MarshalJSON[T]),
		Unmarshaller: UnmarshalFunc[T](UnmarshalJSON[T]),
		Comparer:     &JSONComparer[T]{opts: opt.Body},
	}

	return Snapshot[T](fileName, t, &s, opt.Hooks...)
}

type JSONOption struct {
	Hooks []Hook[any]
	Body  []cmp.Option
}

func MarshalJSON[T any](t T) ([]byte, error) {
	return internal.PrettyJSON(t)
}

func UnmarshalJSON[T any](b []byte) (T, error) {
	var t T
	err := json.Unmarshal(b, &t)

	return t, err
}

type JSONComparer[T any] struct {
	opts []cmp.Option
}

func (cmp JSONComparer[T]) Compare(snapshot, received T) error {
	return internal.ANSIDiff(snapshot, received, cmp.opts...)
}
