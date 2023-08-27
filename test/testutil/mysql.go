package testutil

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/test/testdump"
)

func DumpMySQL(t *testing.T, dump *SQL, opts ...SQLOption) {
	t.Helper()

	var fileName string
	var hooks []testdump.Hook[*SQL]
	sqlOpt := new(testdump.MySQLOption)

	for _, opt := range opts {
		switch o := opt.(type) {
		case FileName:
			fileName = string(o)
		case *sqlHookOption:
			hooks = append(hooks, o.hook)
		case *sqlCmpOption:
			sqlOpt.Args = append(sqlOpt.Args, o.args...)
			sqlOpt.Vars = append(sqlOpt.Vars, o.vars...)
			sqlOpt.Result = append(sqlOpt.Result, o.result...)
		default:
			panic(fmt.Errorf("testutil: unhandled SQL option: %#v", opt))
		}
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: fileName,
		FileExt:  ".sql",
	}

	if err := testdump.MySQL(testdump.NewFile(p.String()), dump, sqlOpt, hooks...); err != nil {
		t.Fatal(err)
	}
}
