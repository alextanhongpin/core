package testutil

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/test/testdump"
)

type MySQLOption struct {
	Dump     *DumpSQLOption
	FileName string
}

func DumpMySQL(t *testing.T, dump *SQL, opts ...SQLOption) {
	t.Helper()

	o := new(sqlOption)
	o.Dump = new(DumpSQLOption)

	for _, opt := range opts {
		switch ot := opt.(type) {
		case FileName:
			o.FileName = string(ot)
		case sqlOptionHook:
			ot(o)
		default:
			panic(fmt.Errorf("testutil: unhandled SQL option: %#v", opt))
		}
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: o.FileName,
		FileExt:  ".sql",
	}

	fileName := p.String()
	if err := testdump.MySQL(testdump.NewFile(fileName), dump, o.Dump); err != nil {
		t.Fatal(err)
	}
}
