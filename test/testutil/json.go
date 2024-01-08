package testutil

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
)

type JSONOption interface {
	isJSON()
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
