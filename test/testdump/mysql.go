package testdump

import (
	"fmt"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/storage/sql/sqldump"
	"github.com/google/go-cmp/cmp"
)

type SQL = sqldump.SQL

type SQLOption struct {
	Hooks  []Hook[*SQL]
	Args   []cmp.Option
	Vars   []cmp.Option
	Result []cmp.Option
}

func MySQL(rw readerWriter, sql *SQL, opt *SQLOption) error {
	if opt == nil {
		opt = new(SQLOption)
	}

	type T = *SQL

	var s S[T] = &snapshot[T]{
		marshaler:   MarshalFunc[T](MarshalMySQL),
		unmarshaler: UnmarshalFunc[T](UnmarshalMySQL),
		comparer: &MySQLComparer{
			args:   opt.Args,
			vars:   opt.Vars,
			result: opt.Result,
		},
	}

	return Snapshot(rw, sql, s, opt.Hooks...)
}

func MarshalMySQL(s *SQL) ([]byte, error) {
	return sqldump.DumpMySQL(s, internal.MarshalYAMLPreserveKeysOrder)
}

func UnmarshalMySQL(b []byte) (*SQL, error) {
	return sqldump.Read(b, internal.UnmarshalYAMLPreserveKeysOrder[any])
}

type MySQLComparer struct {
	args   []cmp.Option
	vars   []cmp.Option
	result []cmp.Option
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

	if err := internal.ANSIDiff(x.ArgMap, y.ArgMap, cmp.args...); err != nil {
		return fmt.Errorf("Args: %w", err)
	}

	if err := internal.ANSIDiff(x.VarMap, y.VarMap, cmp.vars...); err != nil {
		return fmt.Errorf("Vars: %w", err)
	}

	if err := internal.ANSIDiff(x.Result, y.Result, cmp.result...); err != nil {
		return fmt.Errorf("Result: %w", err)
	}

	return nil
}
