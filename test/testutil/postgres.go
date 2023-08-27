package testutil

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
	"github.com/google/go-cmp/cmp"
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

func DumpPostgres(t *testing.T, dump *SQL, opts ...SQLOption) {
	t.Helper()

	var fileName string
	var hooks []testdump.Hook[*SQL]
	sqlOpt := new(testdump.PostgresOption)

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

	if err := testdump.Postgres(testdump.NewFile(p.String()), dump, sqlOpt, hooks...); err != nil {
		t.Fatal(err)
	}
}

type sqlHookOption struct {
	hook testdump.Hook[*SQL]
}

func (sqlHookOption) isSQL() {}

type sqlCmpOption struct {
	args   []cmp.Option
	vars   []cmp.Option
	result []cmp.Option
}

func (sqlCmpOption) isSQL() {}

func IgnoreResultFields(fields ...string) *sqlCmpOption {
	return &sqlCmpOption{
		result: []cmp.Option{internal.IgnoreMapEntries(fields...)},
	}
}

func IgnoreArgs(fields ...string) *sqlCmpOption {
	return &sqlCmpOption{
		args: []cmp.Option{internal.IgnoreMapEntries(fields...)},
	}
}

func IgnoreVars(fields ...string) *sqlCmpOption {
	return &sqlCmpOption{
		vars: []cmp.Option{internal.IgnoreMapEntries(fields...)},
	}
}

func InspectSQL(hook func(snapshot, received *SQL) error) *sqlHookOption {
	return &sqlHookOption{
		hook: testdump.CompareHook(hook),
	}
}

func InterceptSQL(hook func(t *SQL) (*SQL, error)) *sqlHookOption {
	return &sqlHookOption{
		hook: testdump.MarshalHook(hook),
	}
}
