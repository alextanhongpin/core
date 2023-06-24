package testutil

import (
	"encoding/json"
	"fmt"
	"strings"

	querypb "vitess.io/vitess/go/vt/proto/query"
	"vitess.io/vitess/go/vt/sqlparser"
)

type MySQLDumper struct {
	dump *SQLDump
	opts *sqlOption
}

func NewMySQLDumper(dump *SQLDump, opts ...SQLOption) *MySQLDumper {
	return &MySQLDumper{
		dump: dump,
		opts: NewSQLOption(opts...),
	}
}

func (d *MySQLDumper) Dump() ([]byte, error) {
	stmt, known, err := sqlparser.Parse2(d.dump.Stmt)
	if err != nil {
		return nil, err
	}

	args := make(map[string]any)
	for i, v := range d.dump.Args {
		key := fmt.Sprintf("v%d", i+1)
		args[key] = v
	}

	if d.opts.parameterize {
		bv := make(map[string]*querypb.BindVariable)
		err = sqlparser.Normalize(stmt, sqlparser.NewReservedVars("bv", known), bv)
		if err != nil {
			return nil, err
		}

		for k, v := range bv {
			if _, ok := args[k]; ok {
				continue
			}

			if b := v.GetValue(); len(b) > 0 {
				args[k] = string(b)
			} else {
				vals := make([]string, len(v.GetValues()))
				for i, v := range v.GetValues() {
					vals[i] = string(v.GetValue())
				}
				args[k] = vals
			}
		}
	}

	// Unfortunately the prettier doesn't work with ":".
	query := sqlparser.String(stmt)
	queryPretty := query
	if isPythonInstalled {
		queryPrettyBytes, err := sqlformat(query)
		if err == nil {
			queryPretty = string(queryPrettyBytes)
		}
	}

	argsBytes, err := json.MarshalIndent(args, "", " ")
	if err != nil {
		return nil, err
	}

	rows, err := json.MarshalIndent(d.dump.Rows, "", " ")
	if err != nil {
		return nil, err
	}

	lineBreak := string(LineBreak)
	res := []string{
		queryStmtSection,
		queryPretty,
		lineBreak,

		argsStmtSection,
		string(argsBytes),
		lineBreak,

		rowsStmtSection,
		string(rows),
	}

	return []byte(strings.Join(res, string(LineBreak))), nil
}
