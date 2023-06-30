package testutil

import (
	"bytes"
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/google/go-cmp/cmp"
)

type yamlOption struct {
	bodyOpts     []cmp.Option
	inspector    YAMLInspector
	interceptors []YAMLInterceptor
}

func NewYAMLOption(opts ...YAMLOption) *yamlOption {
	j := &yamlOption{}
	for _, opt := range opts {
		switch o := opt.(type) {
		case YAMLCmpOptions:
			j.bodyOpts = append(j.bodyOpts, o...)
		case YAMLInspector:
			j.inspector = o
		case YAMLInterceptor:
			j.interceptors = append(j.interceptors, o)
		case FilePath, FileName:
		// Do nothing.
		default:
			panic("option not implemented")
		}
	}

	return j
}

func DumpYAMLFile(fileName string, v any, opts ...YAMLOption) error {
	type dumpAndCompare struct {
		dumper
		comparer
	}

	dnc := dumpAndCompare{
		dumper:   NewYAMLDumper(v, opts...),
		comparer: NewYAMLComparer(opts...),
	}

	return Dump(fileName, dnc)
}

// DumpYAML dumps a type as yaml.
func DumpYAML(t *testing.T, v any, opts ...YAMLOption) string {
	t.Helper()

	p := NewYAMLPath(opts...)
	if p.FilePath == "" {
		p.FilePath = t.Name()
	}

	if p.FileName == "" {
		p.FileName = typeName(v)
	}

	fileName := p.String()

	if err := DumpYAMLFile(fileName, v, opts...); err != nil {
		t.Fatal(err)
	}

	return fileName
}

type YAMLComparer struct {
	opt *yamlOption
}

func NewYAMLComparer(opts ...YAMLOption) *YAMLComparer {
	return &YAMLComparer{
		opt: NewYAMLOption(opts...),
	}
}

func (c *YAMLComparer) Compare(a, b []byte) error {
	// Get slice of data with optional leading whitespace removed.
	// See RFC 7159, Section 2 for the definition of YAML whitespace.
	a = bytes.TrimLeft(a, " \t\r\n")
	b = bytes.TrimLeft(b, " \t\r\n")

	if c.opt.inspector != nil {
		if err := c.opt.inspector(b); err != nil {
			return err
		}
	}

	want, err := unmarshalYAML(a)
	if err != nil {
		return err
	}

	got, err := unmarshalYAML(b)
	if err != nil {
		return err
	}

	return ansiDiff(want, got, c.opt.bodyOpts...)
}

type YAMLDumper struct {
	v            any
	interceptors []YAMLInterceptor
}

func NewYAMLDumper(v any, opts ...YAMLOption) *YAMLDumper {
	return &YAMLDumper{
		v:            v,
		interceptors: NewYAMLOption(opts...).interceptors,
	}
}

func (d *YAMLDumper) Dump() ([]byte, error) {
	if len(d.interceptors) == 0 {
		return marshalYAML(d.v)
	}

	b, ok := d.v.([]byte)
	if ok {
		return b, nil
	}

	// Marshal to JSON first to enable fields masking etc.
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

	var a any
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, err
	}

	return marshalYAML(a)
}

func marshalYAML(v any) ([]byte, error) {
	b, ok := v.([]byte)
	if ok {
		return b, nil
	}

	return yaml.Marshal(v)
}

func unmarshalYAML(b []byte) (any, error) {
	var a any
	err := yaml.Unmarshal(b, &a)
	return a, err
}
