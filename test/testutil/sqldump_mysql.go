package testutil

import (
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

	query := sqlparser.String(stmt)
	queryPretty := sqlformatOrDefault(query)

	// Populate the known '?' variables that has been given a prefix name.
	args := make(map[string]any)
	for k := range known {
		args[k] = nil
	}

	// In case the args length does not match the known placeholders, the key
	// should still be there.
	for i, v := range d.dump.Args {
		// sqlparser.Parse2 parses the '?' and replaces them with digits with
		// prefix 'v'.
		key := fmt.Sprintf("v%d", i+1)
		args[key] = v
	}

	// Normalize the query and extract the constants into bind variables.
	bv := make(map[string]*querypb.BindVariable)
	err = sqlparser.Normalize(stmt, sqlparser.NewReservedVars("bv", known), bv)
	if err != nil {
		return nil, err
	}

	for k, v := range bv {
		if b := v.GetValue(); len(b) > 0 {
			args[k] = string(b)
			continue
		}

		vals := make([]string, len(v.GetValues()))
		for i, v := range v.GetValues() {
			vals[i] = string(v.GetValue())
		}

		args[k] = vals
	}

	// Unfortunately the prettier doesn't work with ":".
	queryNorm := sqlparser.String(stmt)
	queryNormPretty := sqlformatOrDefault(queryNorm)

	marshalFunc := marshalSelector(d.opts.format)
	argsBytes, err := marshalFunc(args)
	if err != nil {
		return nil, err
	}

	result, err := marshalFunc(d.dump.Result)
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
