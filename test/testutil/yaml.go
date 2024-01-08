package testutil

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
)

type YAMLOption interface {
	isYAML()
}

func DumpYAML[T any](t *testing.T, v T, opts ...YAMLOption) {
	t.Helper()

	var fileName string
	yamlOpt := new(testdump.YAMLOption)

	for _, opt := range opts {
		switch o := opt.(type) {
		case ignoreFields:
			yamlOpt.IgnoreFields = append(yamlOpt.IgnoreFields, o...)
		case maskFields:
			yamlOpt.MaskFields = append(yamlOpt.MaskFields, o...)
		case CmpOption:
			yamlOpt.Body = append(yamlOpt.Body, o...)
		case FileName:
			fileName = string(o)
		default:
			panic(fmt.Errorf("testutil: unhandled YAML option: %#v", opt))
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
