package testdump

import (
	"github.com/alextanhongpin/core/internal"
	"github.com/google/go-cmp/cmp"
)

func YAML[T any](rw readerWriter, t T, opt *YAMLOption[T]) error {
	if opt == nil {
		opt = new(YAMLOption[T])
	}

	var s S[T] = &snapshot[T]{
		marshaler:      MarshalFunc[T](MarshalYAML[T]),
		unmarshaler:    UnmarshalFunc[T](UnmarshalYAML[T]),
		anyUnmarshaler: UnmarshalAnyFunc(UnmarshalYAML[any]),
		anyComparer:    CompareAnyFunc((&YAMLComparer[any]{opts: opt.Body}).Compare),
	}

	return Snapshot(rw, t, s, opt.Hooks...)
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

type YAMLComparer[T any] struct {
	opts []cmp.Option
}

func (c *YAMLComparer[T]) Compare(snapshot, received T) error {
	return internal.ANSIDiff(snapshot, received, c.opts...)
}
