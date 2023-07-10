package testutil

import (
	"testing"

	"github.com/alextanhongpin/core/test/testdump"
)

type DumpTextOption = testdump.TextOption

type TextOption struct {
	Dump     *DumpTextOption
	FileName string
}

func DumpText(t *testing.T, s string, opt *TextOption) {
	t.Helper()

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: opt.FileName,
		FileExt:  ".txt",
	}

	if err := testdump.Text(p.String(), s, opt.Dump); err != nil {
		t.Fatal(err)
	}
}
