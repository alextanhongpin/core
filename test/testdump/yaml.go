package testdump

import (
	"github.com/alextanhongpin/core/internal"
	"github.com/google/go-cmp/cmp"
)

func YAML[T any](fileName string, t T, opt *YAMLOption[T]) error {
	if opt == nil {
		opt = new(YAMLOption[T])
	}

	s := snapshot[T]{
		Marshaller:   MarshalFunc[T](MarshalYAML[T]),
		Unmarshaller: UnmarshalFunc[T](UnmarshalYAML[T]),
		Comparer:     CompareFunc[T](nopComparer[T]),
		// Custom.
		unmarshalAny: UnmarshalFunc[any](UnmarshalYAML[any]),
		compareAny:   CompareFunc[any](CompareYAML[any](opt.Body...)),
	}

	return Snapshot(fileName, t, &s, opt.Hooks...)
}

type YAMLOption[T any] struct {
	Hooks []Hook[T]
	Body  []cmp.Option
}

func MarshalYAML[T any](t T) ([]byte, error) {
	return internal.MarshalYAMLPreserveKeysOrder(t)
}

func UnmarshalYAML[T any](b []byte) (T, error) {
	return internal.UnmarshalYAMLPreserveKeysOrder[T](b)
}

func CompareYAML[T any](opts ...cmp.Option) func(a, b T) error {
	return func(snapshot, received T) error {
		return internal.ANSIDiff(snapshot, received, opts...)
	}
}
