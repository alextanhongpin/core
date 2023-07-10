package testdump

import (
	"fmt"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/storage/sql/sqldump"
	"github.com/google/go-cmp/cmp"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type SQL = sqldump.SQL

func MySQL(fileName string, sql *SQL, opt *MySQLOption) error {
	if opt == nil {
		opt = new(MySQLOption)
	}

	type T = *SQL

	s := snapshot[T]{
		Marshaller:   MarshalFunc[T](MarshalMySQL),
		Unmarshaller: UnmarshalFunc[T](UnmarshalMySQL),
		Comparer: &MySQLComparer{
			args:   opt.Args,
			result: opt.Result,
		},
	}

	return Snapshot(fileName, sql, &s, opt.Hooks...)
}

type MySQLOption struct {
	Hooks  []Hook[*SQL]
	Args   []cmp.Option
	Result []cmp.Option
}

func MarshalMySQL(s *SQL) ([]byte, error) {
	return sqldump.DumpMySQL(s, internal.MarshalYAMLPreserveKeysOrder)
}

func UnmarshalMySQL(b []byte) (*SQL, error) {
	return sqldump.Read(b, internal.UnmarshalYAMLPreserveKeysOrder[any])
}

type MySQLComparer struct {
	args   []cmp.Option
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
		dmp := diffmatchpatch.New()

		diffs := dmp.DiffMain(x.Query, y.Query, false)
		diffs = dmp.DiffCleanupEfficiency(diffs)
		diff := dmp.DiffPrettyText(diffs)

		return fmt.Errorf("\nThe SQL query has been modified:\n\n%s", diff)
	}

	if err := internal.ANSIDiff(x.ArgsMap, y.ArgsMap, cmp.args...); err != nil {
		return fmt.Errorf("Args: %w", err)
	}

	if err := internal.ANSIDiff(x.Result, y.Result, cmp.result...); err != nil {
		return fmt.Errorf("Result: %w", err)
	}

	return nil
}
