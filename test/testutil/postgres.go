package testutil

import (
	"testing"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
)

type SQL = testdump.SQL

func NewSQL(query string, args []any, result any) *SQL {
	return &SQL{
		Query:  query,
		Args:   args,
		Result: result,
	}
}

type DumpSQLOption = testdump.SQLOption

type SQLOption func(*SqlOption)

type SqlOption struct {
	Dump     *DumpSQLOption
	FileName string
}

func DumpPostgres(t *testing.T, dump *testdump.SQL, opts ...SQLOption) {
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

	if err := testdump.Postgres(p.String(), dump, o.Dump); err != nil {
		t.Fatal(err)
	}
}

func IgnoreResultFields(fields ...string) SQLOption {
	return func(o *SqlOption) {
		o.Dump.Result = append(o.Dump.Result, internal.IgnoreMapEntries(fields...))
	}
}

func IgnoreArgs(fields ...string) SQLOption {
	return func(o *SqlOption) {
		o.Dump.Args = append(o.Dump.Args, internal.IgnoreMapEntries(fields...))
	}
}

func IgnoreVars(fields ...string) SQLOption {
	return func(o *SqlOption) {
		o.Dump.Vars = append(o.Dump.Vars, internal.IgnoreMapEntries(fields...))
	}
}

func SQLFileName(name string) SQLOption {
	return func(o *SqlOption) {
		o.FileName = name
	}
}

func InspectSQL(hook func(snapshot, received *SQL) error) SQLOption {
	return func(o *SqlOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.CompareHook(hook))
	}
}

func InterceptSQL(hook func(t *SQL) (*SQL, error)) SQLOption {
	return func(o *SqlOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MarshalHook(hook))
	}
}
