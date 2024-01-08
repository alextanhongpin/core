package testdump

import (
	"github.com/alextanhongpin/core/internal"
	"github.com/google/go-cmp/cmp"
)

type YAMLOption struct {
	Body         []cmp.Option
	IgnoreFields []string
	MaskFields   []string
}

func YAML[T any](rw readerWriter, t T, opt *YAMLOption) error {
	if opt == nil {
		opt = new(YAMLOption)
	}

	t, err := maskFieldsFromTags(t, "yaml", opt.MaskFields...)
	if err != nil {
		return err
	}

	b, err := MarshalYAML(t)
	if err != nil {
		return err
	}

	if err := rw.Write(b); err != nil {
		return err
	}

	received, err := UnmarshalYAML[any](b)
	if err != nil {
		return err
	}

	b, err = rw.Read()
	if err != nil {
		return err
	}

	snapshot, err := UnmarshalYAML[any](b)
	if err != nil {
		return err
	}

	opt.Body = append(opt.Body, ignoreFieldsFromTags(t, "yaml", opt.IgnoreFields...)...)
	cmp := &YAMLComparer[any]{Body: opt.Body}
	return cmp.Compare(snapshot, received)
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
