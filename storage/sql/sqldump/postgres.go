package sqldump

import (
	"fmt"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/storage/sql/sqlformat"
	pg_query "github.com/pganalyze/pg_query_go/v4"
)

func DumpPostgres(sql *SQL) ([]byte, error) {
	q, err := standardizePostgres(sql.Query)
	if err != nil {
		return nil, err
	}

	n, err := pg_query.Normalize(q)
	if err != nil {
		return nil, err
	}

	vars := postgresVars(n, q)

	q, err = sqlformat.Format(q)
	if err != nil {
		return nil, err
	}

	n, err = sqlformat.Format(n)
	if err != nil {
		return nil, err
	}

	var a []byte
	if len(sql.Args) > 0 {
		args := make(map[string]any)
		for i, v := range sql.Args {
			k := fmt.Sprintf("$%d", i+1)
			args[k] = v
		}

		a, err = internal.MarshalYAMLPreserveKeysOrder(args)
		if err != nil {
			return nil, err
		}
	}

	var v []byte
	if len(vars) > 0 {
		kv := make(map[string]any)
		for _, v := range vars {
			kv[v.Name] = v.Value
		}

		v, err = internal.MarshalYAMLPreserveKeysOrder(kv)
		if err != nil {
			return nil, err
		}
	}

	var b []byte
	if sql.Result != nil {
		b, err = internal.MarshalYAMLPreserveKeysOrder(sql.Result)
		if err != nil {
			return nil, err
		}
	}

	return dump(q, a, n, v, b), nil
}

// MatchPostgresQuery checks if two queries are equal,
// ignoring variables.
func MatchPostgresQuery(a, b string) (bool, error) {
	fa, err := pg_query.Fingerprint(a)
	if err != nil {
		return false, err
	}
	fb, err := pg_query.Fingerprint(b)
	if err != nil {
		return false, err
	}

	return fa == fb, nil
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
