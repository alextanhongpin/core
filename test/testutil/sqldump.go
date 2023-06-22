package testutil

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/mjibson/sqlfmt"
	pg_query "github.com/pganalyze/pg_query_go/v4"
)

const queryStmtSection = "-- Query"
const argsStmtSection = "-- Args"
const rowsStmtSection = "-- Rows"

func DumpSQL(t *testing.T, dump *SQLDump, opts ...SQLOption) {
	t.Helper()
	opt := NewSQLPath(opts...)
	if opt.FilePath == "" {
		opt.FilePath = t.Name()
	}

	fileName := opt.String()

	if err := DumpSQLFile(fileName, dump); err != nil {
		t.Fatal(err)
	}
}

func DumpSQLFile(fileName string, dump *SQLDump) error {
	type dumpAndCompare struct {
		dumper
		comparer
	}

	dnc := dumpAndCompare{
		dumper:   dump,
		comparer: NewSQLComparer(),
	}

	return Dump(fileName, dnc)
}

type SQLDump struct {
	Stmt string
	Args []any
	Rows any
}

func NewSQLDump(stmt string, args []any, rows any) *SQLDump {
	return &SQLDump{
		Stmt: stmt,
		Args: args,
		Rows: rows,
	}
}

func (d *SQLDump) Dump() ([]byte, error) {
	result, err := pg_query.Parse(d.Stmt)
	if err != nil {
		return nil, err
	}

	query, err := pg_query.Deparse(result)
	if err != nil {
		return nil, err
	}

	// Conditionally determine the line width so that the text is not a single
	// line when it is too short.
	n := len(query)
	if n < 120 {
		n = n / 2
	} else {
		n = 80
	}
	if n < 32 {
		n = 32
	}

	prettyStmt, err := sqlfmt.FmtSQL(tree.PrettyCfg{
		LineWidth: n,
		TabWidth:  2,
		JSONFmt:   true,
	}, []string{query})
	if err != nil {
		return nil, err
	}

	args := make(map[string]any)
	for i, arg := range d.Args {
		// For Postgres, it starts from 1.
		key := fmt.Sprintf("$%d", i+1)
		args[key] = arg
	}

	argsBytes, err := json.MarshalIndent(args, "", " ")
	if err != nil {
		return nil, err
	}

	rows, err := json.MarshalIndent(d.Rows, "", " ")
	if err != nil {
		return nil, err
	}

	lineBreak := string(LineBreak)
	res := []string{
		queryStmtSection,
		prettyStmt,
		lineBreak,

		argsStmtSection,
		string(argsBytes),
		lineBreak,

		rowsStmtSection,
		string(rows),
	}

	return []byte(strings.Join(res, string(LineBreak))), nil
}

type SQLComparer struct{}

func NewSQLComparer() *SQLComparer {
	return &SQLComparer{}
}

func (c *SQLComparer) Compare(a, b []byte) error {
	a = bytes.TrimLeft(a, " \t\r\n")
	b = bytes.TrimLeft(b, " \t\r\n")

	l, err := parseSQLDump(a)
	if err != nil {
		return err
	}

	r, err := parseSQLDump(b)
	if err != nil {
		return err
	}

	return cmpDiff(l, r)
}

func parseSQLDump(b []byte) (*SQLDump, error) {
	br := bytes.NewReader(b)
	s := bufio.NewScanner(br)

	dump := new(SQLDump)

	for s.Scan() {
		line := s.Text()

		switch line {
		case queryStmtSection:
			var tmp [][]byte

			for s.Scan() {
				line := s.Bytes()
				if len(line) == 0 {
					break
				}

				tmp = append(tmp, line)
			}
			dump.Stmt = string(bytes.Join(tmp, LineBreak))
		case argsStmtSection:
			var tmp [][]byte

			for s.Scan() {
				line := s.Bytes()
				if len(line) == 0 {
					break
				}

				tmp = append(tmp, line)
			}

			jsonBytes := bytes.Join(tmp, LineBreak)

			var m map[string]any
			if err := json.Unmarshal(jsonBytes, &m); err != nil {
				return nil, err
			}

			args := make([]any, len(m))
			for i := 0; i < len(m); i++ {
				key := fmt.Sprintf("$%d", i+1)
				args[i] = m[key]
			}

			dump.Args = args
		case rowsStmtSection:
			var tmp [][]byte

			for s.Scan() {
				line := s.Bytes()
				if len(line) == 0 {
					break
				}

				tmp = append(tmp, line)
			}

			jsonBytes := bytes.Join(tmp, LineBreak)
			if json.Valid(jsonBytes) {
				switch jsonBytes[0] {
				case '{':
					var m map[string]any
					if err := json.Unmarshal(jsonBytes, &m); err != nil {
						return nil, err
					}
					dump.Rows = m
				case '[':
					var m []map[string]any
					if err := json.Unmarshal(jsonBytes, &m); err != nil {
						return nil, err
					}
					dump.Rows = m
				default:
					var m any
					if err := json.Unmarshal(jsonBytes, &m); err != nil {
						return nil, err
					}
					dump.Rows = m
				}
			} else {
				dump.Rows = string(jsonBytes)
			}
		}
	}

	return dump, nil
}
