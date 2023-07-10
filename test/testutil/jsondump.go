package testutil

import (
	"testing"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
)

type JSONOption[T any] struct {
	Dump     *testdump.JSONOption[T]
	FileName string
}

func DumpJSON[T any](t *testing.T, v T, opt *JSONOption[T]) {
	t.Helper()

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: internal.Or(opt.FileName, internal.TypeName(v)),
		FileExt:  ".json",
	}

	if err := testdump.JSON(p.String(), v, opt.Dump); err != nil {
		t.Fatal(err)
	}
}
