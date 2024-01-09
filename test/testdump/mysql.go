package testdump

import (
	"fmt"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/storage/sql/sqldump"
	"github.com/google/go-cmp/cmp"
)

type SQL = sqldump.SQL

type MySQLOption struct {
	Args   []cmp.Option
	Vars   []cmp.Option
	Result []cmp.Option
}

func MySQL(rw readerWriter, sql *SQL, opt *MySQLOption, hooks ...Hook[*SQL]) error {
	if opt == nil {
		opt = new(MySQLOption)
	}

	type T = *SQL

	var s S[T] = &snapshot[T]{
		marshaler:   MarshalFunc[T](MarshalMySQL),
		unmarshaler: UnmarshalFunc[T](UnmarshalMySQL),
		comparer: &MySQLComparer{
			Args:   opt.Args,
			Vars:   opt.Vars,
			Result: opt.Result,
		},
	}

	s = Hooks[T](hooks).Apply(s)

	return Snapshot(rw, sql, s)
}

func MarshalMySQL(s *SQL) ([]byte, error) {
	return sqldump.DumpMySQL(s)
}

func UnmarshalMySQL(b []byte) (*SQL, error) {
	return sqldump.Read(b)
}

type MySQLComparer struct {
	Args   []cmp.Option
	Vars   []cmp.Option
	Result []cmp.Option
}

func (cmp *MySQLComparer) Compare(snapshot, received *SQL) error {
	x := snapshot
	y := received

	ok, err := sqldump.MatchMySQLQuery(x.Query, y.Query)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("Query: %w", internal.ANSIDiff(x.Query, y.Query))
	}

	if err := internal.ANSIDiff(x.ArgMap, y.ArgMap, cmp.Args...); err != nil {
		return fmt.Errorf("Args: %w", err)
	}

	if err := internal.ANSIDiff(x.VarMap, y.VarMap, cmp.Vars...); err != nil {
		return fmt.Errorf("Vars: %w", err)
	}

	if err := internal.ANSIDiff(x.Result, y.Result, cmp.Result...); err != nil {
		return fmt.Errorf("Result: %w", err)
	}

	return nil
}
