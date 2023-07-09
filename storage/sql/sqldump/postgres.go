package sqldump

import (
	"fmt"

	"github.com/alextanhongpin/core/storage/sql/sqlformat"
	pg_query "github.com/pganalyze/pg_query_go/v4"
)

func DumpPostgres(sql *SQL, marshalFunc func(v any) ([]byte, error)) ([]byte, error) {
	q, err := standardizePostgres(sql.Query)
	if err != nil {
		return nil, err
	}

	q, err = sqlformat.Format(q)
	if err != nil {
		return nil, err
	}

	args := make(map[string]any)
	for i, v := range sql.Args {
		k := fmt.Sprintf("$%d", i+1)
		args[k] = v
	}
	a, err := marshalFunc(args)
	if err != nil {
		return nil, err
	}

	b, err := marshalFunc(sql.Result)
	if err != nil {
		return nil, err
	}

	return []byte(dump(q, a, b)), nil
}

// MatchPostgresQuery checks if two queries are equal,
// ignoring variables.
func MatchPostgresQuery(a, b string) (bool, error) {
	x, err := normalizePostgres(a)
	if err != nil {
		return false, err
	}

	y, err := normalizePostgres(b)
	if err != nil {
		return false, err
	}

	return x == y, nil
}

// standardizePostgres standardize the capitalization and strip of new lines etc.
func standardizePostgres(q string) (string, error) {
	stmt, err := pg_query.Parse(q)
	if err != nil {
		return "", err
	}

	q, err = pg_query.Deparse(stmt)
	if err != nil {
		return "", err
	}

	return q, nil
}

// normalize applies standardization and replaces the
// constant variables with placeholders, e.g. $1, $2 etc.
func normalizePostgres(q string) (string, error) {
	q, err := standardizePostgres(q)
	if err != nil {
		return "", err
	}

	q, err = pg_query.Normalize(q)
	return q, err
}
