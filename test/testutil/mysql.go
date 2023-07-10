package testutil

import (
	"testing"

	"github.com/alextanhongpin/core/test/testdump"
)

type MySQLOption struct {
	Dump     *DumpSQLOption
	FileName string
}

func DumpMySQL(t *testing.T, dump *testdump.SQL, opts ...SQLOption) {
	t.Helper()

	o := new(SqlOption)
	o.Dump = new(DumpSQLOption)

	for _, opt := range opts {
		opt(o)
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: o.FileName,
		FileExt:  ".sql",
	}

	if err := testdump.MySQL(p.String(), dump, o.Dump); err != nil {
		t.Fatal(err)
	}
}
