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

	jsonOpt := newJSONOption(opts...)

	var fileName string
	for _, opt := range opts {
		switch v := opt.(type) {
		case FileName:
			fileName = string(v)
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

type jsonOption = testdump.JSONOption

func newJSONOption(opts ...JSONOption) *jsonOption {
	o := new(jsonOption)

	for _, opt := range opts {
		switch v := opt.(type) {
		case ignoreFields:
			o.IgnoreFields = append(o.IgnoreFields, v...)
		case maskFields:
			o.MaskFields = append(o.MaskFields, v...)
		case CmpOption:
			o.Body = append(o.Body, v...)
		case FileName:

		default:
			panic(fmt.Errorf("testutil: unhandled JSON option: %#v", opt))
		}
	}

	return o
}
