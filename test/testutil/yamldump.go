package testutil

import (
	"testing"

	"github.com/alextanhongpin/core/test/testdump"
)

type YAMLOption[T any] struct {
	Dump     *testdump.YAMLOption[T]
	FileName string
}

func DumpYAML[T any](t *testing.T, v T, opt *YAMLOption[T]) {
	t.Helper()

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: opt.FileName,
		FileExt:  ".yaml",
	}

	if err := testdump.YAML(p.String(), v, opt.Dump); err != nil {
		t.Fatal(err)
	}
}
