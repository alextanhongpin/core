package testdump

import (
	"fmt"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/storage/sql/sqldump"
	"github.com/google/go-cmp/cmp"
)

type PostgresOption struct {
	Args   []cmp.Option
	Vars   []cmp.Option
	Result []cmp.Option
}

func Postgres(rw readerWriter, sql *SQL, opt *PostgresOption, hooks ...Hook[*SQL]) error {
	if opt == nil {
		opt = new(PostgresOption)
	}

	type T = *SQL

	var s S[T] = &snapshot[T]{
		marshaler:   MarshalFunc[T](MarshalPostgres),
		unmarshaler: UnmarshalFunc[T](UnmarshalPostgres),
		comparer: &PostgresComparer{
			Args:   opt.Args,
			Vars:   opt.Vars,
			Result: opt.Result,
		},
	}

	s = Hooks[T](hooks).Apply(s)

	return Snapshot(rw, sql, s)
}

func MarshalPostgres(s *SQL) ([]byte, error) {
	return sqldump.DumpPostgres(s)
}

func UnmarshalPostgres(b []byte) (*SQL, error) {
	return sqldump.Read(b)
}

type PostgresComparer struct {
	Args   []cmp.Option
	Vars   []cmp.Option
	Result []cmp.Option
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
