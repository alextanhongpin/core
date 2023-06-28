package testutil

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type jsonOption struct {
	bodyOpts     []cmp.Option
	inspector    JSONInspector
	interceptors []JSONInterceptor
}

func NewJSONOption(opts ...JSONOption) *jsonOption {
	j := &jsonOption{}
	for _, opt := range opts {
		switch o := opt.(type) {
		case JSONCmpOptions:
			j.bodyOpts = append(j.bodyOpts, o...)
		case JSONInspector:
			j.inspector = o
		case JSONInterceptor:
			j.interceptors = append(j.interceptors, o)
		case FilePath, FileName:
		// Do nothing.
		default:
			panic("option not implemented")
		}
	}

	return j
}

func DumpJSONFile(fileName string, v any, opts ...JSONOption) error {
	type dumpAndCompare struct {
		dumper
		comparer
	}

	dnc := dumpAndCompare{
		dumper:   NewJSONDumper(v, opts...),
		comparer: NewJSONComparer(opts...),
	}

	return Dump(fileName, dnc)
}

// DumpJSON dumps a type as json.
func DumpJSON(t *testing.T, v any, opts ...JSONOption) string {
	t.Helper()

	p := NewJSONPath(opts...)
	if p.FilePath == "" {
		p.FilePath = t.Name()
	}

	if p.FileName == "" {
		p.FileName = typeName(v)
	}

	fileName := p.String()

	if err := DumpJSONFile(fileName, v, opts...); err != nil {
		t.Fatal(err)
	}

	return fileName
}

type JSONComparer struct {
	opt *jsonOption
}

func NewJSONComparer(opts ...JSONOption) *JSONComparer {
	return &JSONComparer{
		opt: NewJSONOption(opts...),
	}
}

func (c *JSONComparer) Compare(a, b []byte) error {
	// Get slice of data with optional leading whitespace removed.
	// See RFC 7159, Section 2 for the definition of JSON whitespace.
	a = bytes.TrimLeft(a, " \t\r\n")
	b = bytes.TrimLeft(b, " \t\r\n")

	if c.opt.inspector != nil {
		if err := c.opt.inspector(b); err != nil {
			return err
		}
	}

	want, err := unmarshal(a)
	if err != nil {
		return err
	}

	got, err := unmarshal(b)
	if err != nil {
		return err
	}

	return ansiDiff(want, got, c.opt.bodyOpts...)
}

type JSONDumper struct {
	v            any
	interceptors []JSONInterceptor
}

func NewJSONDumper(v any, opts ...JSONOption) *JSONDumper {
	return &JSONDumper{
		v:            v,
		interceptors: NewJSONOption(opts...).interceptors,
	}
}

func (d *JSONDumper) Dump() ([]byte, error) {
	if len(d.interceptors) == 0 {
		return marshal(d.v)
	}

	b, err := json.Marshal(d.v)
	if err != nil {
		return nil, err
	}

	for _, it := range d.interceptors {
		b, err = it(b)
		if err != nil {
			return nil, err
		}
	}

	return marshal(b)
}

func marshal(v any) ([]byte, error) {
	// If it is byte, pretty print.
	b, ok := v.([]byte)
	if ok {
		if !json.Valid(b) {
			return b, nil
		}

		// Prettify.
		var bb bytes.Buffer
		if err := json.Indent(&bb, b, "", " "); err != nil {
			return nil, err
		}

		return bb.Bytes(), nil
	}

	return json.MarshalIndent(v, "", " ")
}
