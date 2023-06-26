package testutil

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/alextanhongpin/core/types/maputil"
	"github.com/google/go-cmp/cmp"
)

type jsonOption struct {
	bodyOpts []cmp.Option
	bodyFn   InspectBody
	maskFn   MaskFn
}

func NewJSONOption(opts ...JSONOption) *jsonOption {
	j := &jsonOption{}
	for _, opt := range opts {
		switch o := opt.(type) {
		case InspectBody:
			j.bodyFn = o
		case IgnoreFieldsOption:
			j.bodyOpts = append(j.bodyOpts, IgnoreMapKeys(o...))
		case CmpOptionsOptions:
			j.bodyOpts = append(j.bodyOpts, o...)
		case MaskFn:
			j.maskFn = o
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

	want, err := unmarshal(a)
	if err != nil {
		return err
	}

	got, err := unmarshal(b)
	if err != nil {
		return err
	}

	if c.opt.bodyFn != nil {
		c.opt.bodyFn(b)
	}

	return ansiDiff(want, got, c.opt.bodyOpts...)
}

type JSONDumper struct {
	v      any
	maskFn func(key string) bool
}

func NewJSONDumper(v any, opts ...JSONOption) *JSONDumper {
	return &JSONDumper{
		v:      v,
		maskFn: NewJSONOption(opts...).maskFn,
	}
}

func (d *JSONDumper) Dump() ([]byte, error) {
	if d.maskFn == nil {
		return d.dump()
	}

	return d.maskBeforeDump()
}

func (d *JSONDumper) dump() ([]byte, error) {
	return marshal(d.v)
}

func (d *JSONDumper) maskBeforeDump() ([]byte, error) {
	b, err := marshal(d.v)
	if err != nil {
		return nil, err
	}

	// Mask can only be applied to map[string]any.
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	// Apply mask.
	masked := maputil.MaskFunc(m, d.maskFn)
	return marshal(masked)
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
