package testutil

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
)

type JSONOption interface {
	isJSON()
}

type cueOption struct {
	opt *testdump.CUEOption
}

func (c *cueOption) isJSON() {}

func CUESchema(schema string, opts ...cue.Option) JSONOption {
	return &cueOption{
		opt: &testdump.CUEOption{
			Schema:  schema,
			Options: opts,
		},
	}
}

func CUESchemaPath(schemaPath string, opts ...cue.Option) JSONOption {
	return &cueOption{
		opt: &testdump.CUEOption{
			SchemaPath: schemaPath,
			Options:    opts,
		},
	}
}

func DumpJSON[T any](t *testing.T, v T, opts ...JSONOption) {
	t.Helper()

	var fileName string
	jsonOpt := new(testdump.JSONOption)

	for _, opt := range opts {
		switch o := opt.(type) {
		case ignoreFields:
			jsonOpt.IgnoreFields = append(jsonOpt.IgnoreFields, o...)
		case maskFields:
			jsonOpt.MaskFields = append(jsonOpt.MaskFields, o...)
		case CmpOption:
			jsonOpt.Body = append(jsonOpt.Body, o...)
		case *cueOption:
			jsonOpt.CUEOption = o.opt
		case FileName:
			fileName = string(o)
		default:
			panic(fmt.Errorf("testutil: unhandled JSON option: %#v", opt))
		}
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: internal.Or(fileName, internal.TypeName(v)),
		FileExt:  ".json",
	}

	if err := testdump.JSON(testdump.NewFile(p.String()), v, jsonOpt); err != nil {
		t.Fatal(err)
	}
}
