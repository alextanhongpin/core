package sqldump

import (
	"fmt"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/storage/sql/sqlformat"
	querypb "vitess.io/vitess/go/vt/proto/query"

	"vitess.io/vitess/go/vt/sqlparser"
)

func DumpMySQL(sql *SQL) ([]byte, error) {
	q, err := standardizeMySQL(sql.Query)
	if err != nil {
		return nil, err
	}

	q, err = sqlformat.Format(q)
	if err != nil {
		return nil, err
	}

	var a []byte
	if len(sql.Args) > 0 {
		args := make(map[string]any)
		for i, v := range sql.Args {
			// sqlparser replaces all '?' with ':v1', ':v2', ':vn'
			// ...
			k := fmt.Sprintf("v%d", i+1)
			args[k] = v
		}

		a, err = internal.MarshalYAMLPreserveKeysOrder(args)
		if err != nil {
			return nil, err
		}
	}

	n, vars, err := mySQLVars(sql.Query)
	if err != nil {
		return nil, err
	}

	n, err = sqlformat.Format(n)
	if err != nil {
		return nil, err
	}

	var v []byte
	if len(vars) > 0 {
		kv := make(map[string]any)
		for _, v := range vars {
			kv[fmt.Sprintf("%v", v.Name)] = v.Value
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

// MatchMySQLQuery checks if two queries are equal,
// ignoring variables.
func MatchMySQLQuery(a, b string) (bool, error) {
	return sqlparser.QueryMatchesTemplates(a, []string{b})
}

func standardizeMySQL(q string) (string, error) {
	stmt, err := sqlparser.Parse(q)
	if err != nil {
		return "", err
	}

	q = sqlparser.String(stmt)

	// sqlparser replaces all ? with the format :v1, :v2,
	// :vn ...
	return q, nil
}

// Referred from sqlparser.QueryMatchesTemplates(q, []string{q})
func normalizeMySQL(q string) (string, error) {
	bv := make(map[string]*querypb.BindVariable)
	stmt, reservedVars, err := sqlparser.Parse2(q)
	if err != nil {
		return "", err
	}

	err = sqlparser.Normalize(stmt, sqlparser.NewReservedVars("", reservedVars), bv)
	if err != nil {
		return "", err
	}

	normalized := sqlparser.CanonicalString(stmt)

	return normalized, nil
}
