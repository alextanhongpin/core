package testutil

import (
	"fmt"
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

type SQLOption interface {
	isSQL()
}

type DumpSQLOption = testdump.SQLOption

func DumpPostgres(t *testing.T, dump *SQL, opts ...SQLOption) {
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

	if err := testdump.Postgres(p.String(), dump, o.Dump); err != nil {
		t.Fatal(err)
	}
}

type sqlOptionHook func(*sqlOption)

func (s sqlOptionHook) isSQL() {}

type sqlOption struct {
	Dump     *DumpSQLOption
	FileName string
}

func IgnoreResultFields(fields ...string) sqlOptionHook {
	return func(o *sqlOption) {
		o.Dump.Result = append(o.Dump.Result, internal.IgnoreMapEntries(fields...))
	}
}

func IgnoreArgs(fields ...string) sqlOptionHook {
	return func(o *sqlOption) {
		o.Dump.Args = append(o.Dump.Args, internal.IgnoreMapEntries(fields...))
	}
}

func IgnoreVars(fields ...string) sqlOptionHook {
	return func(o *sqlOption) {
		o.Dump.Vars = append(o.Dump.Vars, internal.IgnoreMapEntries(fields...))
	}
}

func InspectSQL(hook func(snapshot, received *SQL) error) sqlOptionHook {
	return func(o *sqlOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.CompareHook(hook))
	}
}

func InterceptSQL(hook func(t *SQL) (*SQL, error)) sqlOptionHook {
	return func(o *sqlOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MarshalHook(hook))
	}
}
