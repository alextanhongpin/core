package testutil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

var ErrNonStruct = errors.New("cannot dump non-struct")

var scs = spew.ConfigState{
	Indent:                  "  ",
	DisableCapacities:       true,
	DisablePointerAddresses: true,
}

func DumpStructFile(fileName string, v any) error {
	if !isStruct(v) {
		return fmt.Errorf("%w: %v", ErrNonStruct, v)
	}
	type dumpAndCompare struct {
		dumper
		comparer
	}

	dnc := dumpAndCompare{
		dumper:   NewStructDumper(v),
		comparer: NewStructComparer(),
	}

	return Dump(fileName, dnc)
}

// DumpStruct dumps a type as json.
func DumpStruct(t *testing.T, v any, opts ...TextOption) string {
	t.Helper()

	p := NewTextPath(opts...)
	if p.FilePath == "" {
		p.FilePath = t.Name()
	}

	if p.FileName == "" {
		p.FileName = typeName(v)
	}

	fileName := p.String()
	if err := DumpStructFile(fileName, v); err != nil {
		t.Fatal(err)
	}

	return fileName
}

type StructDumper struct {
	v any
}

func NewStructDumper(v any) *StructDumper {
	return &StructDumper{
		v: v,
	}
}

func (d *StructDumper) Dump() ([]byte, error) {
	res := scs.Sdump(d.v)
	return []byte(res), nil
}

type StructComparer struct{}

func NewStructComparer() *StructComparer {
	return &StructComparer{}
}

func (c *StructComparer) Compare(a, b []byte) error {
	return ansiDiff(a, b)
}
