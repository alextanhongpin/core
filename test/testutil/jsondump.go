package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// DumpJSON dumps a type as json.
func DumpJSON(t *testing.T, v any, opts ...Option) {
	t.Helper()

	dumper := &jsonDumper{v}
	typeName := strings.Join(typeName(v), "_")
	fileName := filepath.Join("./testdata", t.Name(), typeName)
	fileName = fmt.Sprintf("./%s.json", fileName)
	want, got, err := dump(fileName, dumper)
	if err != nil {
		t.Fatal(err)
	}

	o := new(Options)
	for _, opt := range opts {
		opt(o)
	}

	if err := DiffJSON(want, got, o.bodyopts...); err != nil {
		t.Fatal(err)
	}
}

func DiffJSON(a, b []byte, opts ...cmp.Option) error {
	// Get slice of data with optional leading whitespace removed.
	// See RFC 7159, Section 2 for the definition of JSON whitespace.
	a = bytes.TrimLeft(a, " \t\r\n")
	b = bytes.TrimLeft(b, " \t\r\n")

	unmarshal := func(j []byte) (any, error) {
		isObject := len(j) > 0 && j[0] == '{'
		var m any
		if isObject {
			m = make(map[string]any)
		}
		if err := json.Unmarshal(j, &m); err != nil {
			return nil, err
		}
		return m, nil
	}

	want, err := unmarshal(a)
	if err != nil {
		return err
	}
	got, err := unmarshal(b)
	if err != nil {
		return err
	}

	return cmpDiff(want, got, opts...)
}

type jsonDumper struct {
	v any
}

func (d *jsonDumper) Dump() ([]byte, error) {
	return marshal(d.v)
}

func marshal(v any) ([]byte, error) {
	// If it is byte, pretty print.
	b, ok := v.([]byte)
	if ok {
		if !json.Valid(b) {
			return b, nil
		}

		var bb bytes.Buffer
		if err := json.Indent(&bb, b, "", " "); err != nil {
			return nil, err
		}

		return bb.Bytes(), nil
	}

	return json.MarshalIndent(v, "", " ")
}
