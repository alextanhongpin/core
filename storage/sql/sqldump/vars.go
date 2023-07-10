package sqldump

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alextanhongpin/core/internal"
	pg_query "github.com/pganalyze/pg_query_go/v4"
	querypb "vitess.io/vitess/go/vt/proto/query"
	"vitess.io/vitess/go/vt/sqlparser"
)

var placePat = regexp.MustCompile(`\$\d+`)

type Var struct {
	Name  string
	Value string
}

func PostgresVars(q string) ([]Var, error) {
	b, err := standardizePostgres(q)
	if err != nil {
		return nil, err
	}

	a, err := pg_query.Normalize(b)
	if err != nil {
		return nil, err
	}

	return postgresVars(a, b), nil
}

func postgresVars(normalized, original string) []Var {
	a := normalized
	b := original
	var res []Var

	placeholders := placePat.FindAllString(a, -1)

	for i, p := range placeholders {
		in := strings.Index(a, p)
		var next int
		if i == len(placeholders)-1 {
			next = len(a)
		} else {
			next = strings.Index(a, placeholders[i+1])
		}

		j := strings.Index(b[in:], a[in+len(p):next])
		if j == -1 {
			break
		}

		res = append(res, Var{
			Name:  p,
			Value: internal.TrimQuotes(b[in : in+j]),
		})
		a = a[in+len(p):]
		b = b[in+j:]
	}

	return res
}

func MySQLVars(q string) ([]Var, error) {
	_, vars, err := mySQLVars(q)
	return vars, err
}

func mySQLVars(q string) (string, []Var, error) {
	bv := make(map[string]*querypb.BindVariable)
	q, err := sqlparser.NormalizeAlphabetically(q)
	if err != nil {
		return "", nil, err
	}

	stmt, reservedVars, err := sqlparser.Parse2(q)
	if err != nil {
		return "", nil, err
	}

	err = sqlparser.Normalize(stmt, sqlparser.NewReservedVars("", reservedVars), bv)
	if err != nil {
		return "", nil, err
	}

	var res []Var
	for k, v := range bv {
		if b := v.GetValue(); len(b) > 0 {
			res = append(res, Var{
				Name:  k,
				Value: internal.TrimQuotes(string(b)),
			})

			continue
		}

		vals := make([]string, len(v.GetValues()))
		for i, v := range v.GetValues() {
			vals[i] = fmt.Sprintf("%q", internal.TrimQuotes(string(v.GetValue())))
		}

		res = append(res, Var{
			Name:  k,
			Value: strings.Join(vals, ","),
		})
	}

	return sqlparser.CanonicalString(stmt), res, nil
}
