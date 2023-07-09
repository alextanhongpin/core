package testutil

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mitchellh/copystructure"
)

type jsonTypeOption[T any] struct {
	bodyOpts     []cmp.Option
	interceptors []JSONTypeInterceptor[T]
}

func NewJSONTypeOption[T any](opts ...JSONTypeOption) *jsonTypeOption[T] {
	j := &jsonTypeOption[T]{}
	for _, opt := range opts {
		switch o := any(opt).(type) {
		case CmpOptions:
			j.bodyOpts = append(j.bodyOpts, o...)
		case JSONTypeInterceptor[T]:
			j.interceptors = append(j.interceptors, o)
		case FilePath, FileName:
		// Do nothing.
		default:
			panic("option not implemented")
		}
	}

	return j
}

func DumpJSONTypeFile[T any](fileName string, t T, opts ...JSONTypeOption) error {
	type dumpAndCompare struct {
		dumper
		comparer
	}

	dnc := dumpAndCompare{
		dumper:   NewJSONTypeDumper(t, opts...),
		comparer: NewJSONTypeComparer[T](opts...),
	}

	return Dump(fileName, dnc)
}

// DumpJSONType dumps a type as json.
func DumpJSONType[T any](t *testing.T, v T, opts ...JSONTypeOption) string {
	t.Helper()

	p := NewJSONTypePath(opts...)
	if p.FilePath == "" {
		p.FilePath = t.Name()
	}

	if p.FileName == "" {
		p.FileName = typeName(v)
	}

	fileName := p.String()
	if err := DumpJSONTypeFile(fileName, v, opts...); err != nil {
		t.Fatal(err)
	}

	return fileName
}

type JSONTypeComparer[T any] struct {
	opt *jsonTypeOption[T]
}

func NewJSONTypeComparer[T any](opts ...JSONTypeOption) *JSONTypeComparer[T] {
	return &JSONTypeComparer[T]{
		opt: NewJSONTypeOption[T](opts...),
	}
}

func (c *JSONTypeComparer[T]) Compare(a, b []byte) error {
	// Get slice of data with optional leading whitespace removed.
	// See RFC 7159, Section 2 for the definition of JSONType whitespace.
	a = bytes.TrimLeft(a, " \t\r\n")
	b = bytes.TrimLeft(b, " \t\r\n")

	want, err := unmarshalJSON[T](a)
	if err != nil {
		return err
	}

	got, err := unmarshalJSON[T](b)
	if err != nil {
		return err
	}

	return ansiDiff(want, got, c.opt.bodyOpts...)
}

type JSONTypeDumper[T any] struct {
	t            T
	interceptors []JSONTypeInterceptor[T]
}

func NewJSONTypeDumper[T any](t T, opts ...JSONTypeOption) *JSONTypeDumper[T] {
	return &JSONTypeDumper[T]{
		t:            t,
		interceptors: NewJSONTypeOption[T](opts...).interceptors,
	}
}

func (d *JSONTypeDumper[T]) Dump() ([]byte, error) {
	if len(d.interceptors) == 0 {
		return marshalJSON(d.t)
	}

	// Create a copy so that the original won't be modified.
	v, err := copystructure.Copy(d.t)
	if err != nil {
		return nil, err
	}

	t, ok := v.(T)
	if !ok {
		return nil, errors.New("impossible type assertion")
	}

	for _, it := range d.interceptors {
		var err error
		t, err = it(t)
		if err != nil {
			return nil, err
		}
	}

	return marshalJSON(t)
}
