package testutil

import (
	"testing"

	"github.com/alextanhongpin/core/test/testdump"
)

type MySQLDumpOption = testdump.MySQLOption

type MySQLOption struct {
	Dump     *MySQLDumpOption
	FileName string
}

func DumpMySQL(t *testing.T, dump *testdump.SQL, opt *MySQLOption) {
	t.Helper()

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: opt.FileName,
		FileExt:  ".sql",
	}

	if err := testdump.MySQL(p.String(), dump, opt.Dump); err != nil {
		t.Fatal(err)
	}
}
