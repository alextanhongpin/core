package testdump

import (
	"fmt"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/storage/sql/sqldump"
	"github.com/google/go-cmp/cmp"
)

func Postgres(fileName string, sql *SQL, opt *SQLOption) error {
	if opt == nil {
		opt = new(SQLOption)
	}

	type T = *SQL

	s := snapshot[T]{
		Marshaller:   MarshalFunc[T](MarshalPostgres),
		Unmarshaller: UnmarshalFunc[T](UnmarshalPostgres),
		Comparer: &PostgresComparer{
			args:   opt.Args,
			result: opt.Result,
			vars:   opt.Vars,
		},
	}

	return Snapshot(fileName, sql, &s, opt.Hooks...)
}

func MarshalPostgres(s *SQL) ([]byte, error) {
	return sqldump.DumpPostgres(s, internal.MarshalYAMLPreserveKeysOrder)
}

func UnmarshalPostgres(b []byte) (*SQL, error) {
	return sqldump.Read(b, internal.UnmarshalYAMLPreserveKeysOrder[any])
}

type PostgresComparer struct {
	args   []cmp.Option
	vars   []cmp.Option
	result []cmp.Option
}

func (cmp *PostgresComparer) Compare(snapshot, received *SQL) error {
	x := snapshot
	y := received

	ok, err := sqldump.MatchPostgresQuery(x.Query, y.Query)
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
