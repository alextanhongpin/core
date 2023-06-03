package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// DumpJSON dumps a type as json.
func DumpJSON(t *testing.T, v any, opts ...Option) {
	t.Helper()

	if !isStruct(v) {
		t.Fatalf("DumpJSON value must be a struct: %#v", v)
	}

	dumper := &jsonDumper{v}
	fileName := fmt.Sprintf("./testdata/%s.json", t.Name())
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
	var want, got map[string]any
	if err := json.Unmarshal(a, &want); err != nil {
		return err
	}

	if err := json.Unmarshal(b, &got); err != nil {
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
