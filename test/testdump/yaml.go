package testdump

import (
	"github.com/alextanhongpin/core/internal"
	"github.com/google/go-cmp/cmp"
)

func YAML(fileName string, v any, opt *YAMLOption) error {
	if opt == nil {
		opt = new(YAMLOption)
	}
	type T = any

	var s S[T] = &snapshot[T]{
		Marshaller:   MarshalFunc[T](MarshalYAML[T]),
		Unmarshaller: UnmarshalFunc[T](UnmarshalYAML[T]),
		Comparer:     &YAMLComparer[T]{opts: opt.Body},
	}

	return Snapshot(fileName, v, s, opt.Hooks...)
}

type YAMLOption struct {
	Hooks []Hook[any]
	Body  []cmp.Option
}

func MarshalYAML[T any](t T) ([]byte, error) {
	return internal.MarshalYAMLPreserveKeysOrder(t)
}

func UnmarshalYAML[T any](b []byte) (T, error) {
	return internal.UnmarshalYAMLPreserveKeysOrder[T](b)
}

type YAMLComparer[T any] struct {
	opts []cmp.Option
}

func (cmp YAMLComparer[T]) Compare(snapshot, received T) error {
	return internal.ANSIDiff(snapshot, received, cmp.opts...)
}
