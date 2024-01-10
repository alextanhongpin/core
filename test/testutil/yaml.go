package testutil

import (
	"testing"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
)

type YAMLOption interface {
	isYAML()
}

func DumpYAML[T any](t *testing.T, v T, opts ...YAMLOption) {
	t.Helper()

	yamlOpt := newYAMLOption(opts...)

	var fileName string
	for _, opt := range opts {
		switch o := opt.(type) {
		case FileName:
			fileName = string(o)
		}
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: internal.Or(fileName, internal.TypeName(v)),
		FileExt:  ".yaml",
	}

	if err := testdump.YAML(testdump.NewFile(p.String()), v, yamlOpt); err != nil {
		t.Fatal(err)
	}
}

func newYAMLOption(opts ...YAMLOption) *testdump.YAMLOption {
	o := new(testdump.YAMLOption)

	for _, opt := range opts {
		switch v := opt.(type) {
		case ignoreFields:
			o.IgnoreFields = append(o.IgnoreFields, v...)
		case maskFields:
			o.MaskFields = append(o.MaskFields, v...)
		case CmpOption:
			o.Body = append(o.Body, v...)
		}
	}

	return o
}
