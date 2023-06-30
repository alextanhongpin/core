package testutil

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/mjibson/sqlfmt"
	pg_query "github.com/pganalyze/pg_query_go/v4"
)

type PostgresSQLDumper struct {
	dump *SQLDump
	opts *sqlOption
}

func NewPostgresSQLDumper(dump *SQLDump, opts ...SQLOption) *PostgresSQLDumper {
	return &PostgresSQLDumper{
		dump: dump,
		opts: NewSQLOption(opts...),
	}
}

func (d *PostgresSQLDumper) Dump() ([]byte, error) {
	stmt, err := pg_query.Parse(d.dump.Stmt)
	if err != nil {
		return nil, err
	}

	query, err := pg_query.Deparse(stmt)
	if err != nil {
		return nil, err
	}

	queryNorm, args, err := normalizePostgres(query)
	if err != nil {
		return nil, err
	}

	queryNormPretty, err := formatPostgresSQL(queryNorm)
	if err != nil {
		return nil, err
	}

	queryPretty, err := formatPostgresSQL(query)
	if err != nil {
		return nil, err
	}

	for i, v := range d.dump.Args {
		// Postgres uses $n as the placeholder, and n starts from 1.
		k := fmt.Sprintf("$%d", i+1)
		args[k] = v
	}

	argsBytes, err := json.MarshalIndent(args, "", " ")
	if err != nil {
		return nil, err
	}

	result, err := json.MarshalIndent(d.dump.Result, "", " ")
	if err != nil {
		return nil, err
	}

	lineBreak := string(LineBreak)
	res := []string{
		querySection,
		queryPretty,
		lineBreak,

		queryNormalizedSection,
		queryNormPretty,
		lineBreak,

		argsSection,
		string(argsBytes),
		lineBreak,

		resultSection,
		string(result),
	}

	return []byte(strings.Join(res, lineBreak)), nil
}

func formatPostgresSQL(stmt string) (string, error) {
	if isPythonInstalled {
		// Use sqlformat by default, since it is prettier.
		b, err := sqlformat(stmt)
		if err == nil {
			return string(b), nil
		}
	}

	return sqlfmtPostgres(stmt)
}

// sqlfmtPostgres only works for formatting postgres.
func sqlfmtPostgres(stmt string) (string, error) {
	pretty, err := sqlfmt.FmtSQL(tree.PrettyCfg{
		LineWidth: dynamicLineWidth(stmt),
		TabWidth:  2,
		JSONFmt:   true,
	}, []string{stmt})
	if err != nil {
		return "", err
	}

	return pretty, nil
}
