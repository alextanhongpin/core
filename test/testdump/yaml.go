package testdump

import (
	"github.com/alextanhongpin/core/internal"
	"github.com/google/go-cmp/cmp"
)

type YAMLOption struct {
	Body []cmp.Option
}

func YAML[T any](rw readerWriter, t T, opt *YAMLOption, hooks ...Hook[T]) error {
	if opt == nil {
		opt = new(YAMLOption)
	}

	opt.Body = append(opt.Body, ignoreFieldsFromTags(t, "yaml")...)

	var s S[T] = &snapshot[T]{
		marshaler:      MarshalFunc[T](MarshalYAML[T]),
		unmarshaler:    UnmarshalFunc[T](UnmarshalYAML[T]),
		anyUnmarshaler: UnmarshalAnyFunc(UnmarshalYAML[any]),
		anyComparer:    CompareAnyFunc((&YAMLComparer[any]{Body: opt.Body}).Compare),
	}

	s = Hooks[T](append(hooks, maskFieldsFromTags(t, "yaml")...)).Apply(s)

	return Snapshot(rw, t, s)
}

func MarshalYAML[T any](t T) ([]byte, error) {
	return internal.MarshalYAMLPreserveKeysOrder(t)
}

func UnmarshalYAML[T any](b []byte) (T, error) {
	return internal.UnmarshalYAMLPreserveKeysOrder[T](b)
}

type YAMLComparer[T any] struct {
	Body []cmp.Option
}

func (c *YAMLComparer[T]) Compare(snapshot, received T) error {
	return internal.ANSIDiff(snapshot, received, c.Body...)
}
