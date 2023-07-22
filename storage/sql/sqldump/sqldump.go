package sqldump

import (
	"bufio"
	"bytes"
	"errors"
	"regexp"
	"strings"
)

var ErrInvalidDumpFormat = errors.New("sqldump: invalid dump format")

// Split when there is 2 or more new lines.
var patEols = regexp.MustCompile(`[\r\n]{2,}`)

const querySection = "-- Query"
const argsSection = "-- Args"
const normalizedSection = "-- Normalized"
const varsSection = "-- Vars"
const resultSection = "-- Result"

type SQL struct {
	Query      string
	Normalized string
	Args       []any
	// Key-value representation of the ArgMap.
	// Since it can be yaml or json, we leave it to unmarshal to decide.
	ArgMap any
	VarMap any
	Result any
}

func Read(b []byte, unmarshalFunc func([]byte) (any, error)) (*SQL, error) {
	d := new(SQL)

	scanner := bufio.NewScanner(bytes.NewReader(b))

	for scanner.Scan() {
		text := scanner.Text()
		switch text {
		case querySection:
			d.Query = scanSection(scanner)
		case normalizedSection:
			d.Normalized = scanSection(scanner)
		case argsSection:
			body := scanSection(scanner)
			a, err := unmarshalFunc([]byte(body))
			if err != nil {
				return nil, err
			}
			d.ArgMap = a
		case varsSection:
			body := scanSection(scanner)
			a, err := unmarshalFunc([]byte(body))
			if err != nil {
				return nil, err
			}
			d.VarMap = a
		case resultSection:
			body := scanSection(scanner)
			a, err := unmarshalFunc([]byte(body))
			if err != nil {
				return nil, err
			}
			d.Result = a
		default:
			continue
		}
	}

	return d, nil
}

func scanSection(scanner *bufio.Scanner) string {
	var res []string
	for scanner.Scan() {
		s := scanner.Text()
		if len(s) == 0 {
			break
		}
		res = append(res, s)
	}

	return strings.Join(res, "\n")
}

func dump(q string, args []byte, n string, varMap, result []byte) string {
	var sb strings.Builder
	// Query.
	sb.WriteString(querySection)
	sb.WriteRune('\n')

	sb.WriteString(q)
	sb.WriteRune('\n')
	sb.WriteRune('\n')
	sb.WriteRune('\n')

	// Args.
	if len(args) != 0 {
		sb.WriteString(argsSection)
		sb.WriteRune('\n')

		sb.Write(args)
		sb.WriteRune('\n')
		sb.WriteRune('\n')
	}

	// Normalized.
	sb.WriteString(normalizedSection)
	sb.WriteRune('\n')

	sb.WriteString(n)
	sb.WriteRune('\n')
	sb.WriteRune('\n')
	sb.WriteRune('\n')

	// Vars.
	if len(varMap) != 0 {
		sb.WriteString(varsSection)
		sb.WriteRune('\n')

		sb.Write(varMap)
		sb.WriteRune('\n')
		sb.WriteRune('\n')
	}

	// Result.
	if result != nil {
		sb.WriteString(resultSection)
		sb.WriteRune('\n')

		sb.Write(result)
	}

	return strings.TrimSpace(sb.String())
}
