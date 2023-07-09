package sqldump

import (
	"errors"
	"regexp"
	"strings"
)

// Split when there is 2 or more new lines.
var patEols = regexp.MustCompile(`[\r\n]{2,}`)

const querySection = "-- Query"
const argsSection = "-- Args"
const resultSection = "-- Result"

type SQL struct {
	Query string
	Args  []any
	// Key-value representation of the ArgsMap.
	// Since it can be yaml or json, we leave it to unmarshal to decide.
	ArgsMap any
	Result  any
}

func Read(b []byte, unmarshalFunc func([]byte) (any, error)) (*SQL, error) {
	s := string(b)
	sections := patEols.Split(s, -1)
	if len(sections) != 3 {
		return nil, errors.New("invalid dump format")
	}

	d := new(SQL)

	for _, section := range sections {
		i := strings.Index(section, "\n")
		head := section[:i]
		body := section[i+1:] // Skip the new line
		switch head {
		case querySection:
			d.Query = body
		case argsSection:
			// Due to type issue, this is hardcoded.
			a, err := unmarshalFunc([]byte(body))
			if err != nil {
				return nil, err
			}
			d.ArgsMap = a
		case resultSection:
			a, err := unmarshalFunc([]byte(body))
			if err != nil {
				return nil, err
			}
			d.Result = a
		}
	}

	return d, nil
}

func dump(q string, args, result []byte) string {
	var sb strings.Builder
	// Query.
	sb.WriteString(querySection)
	sb.WriteRune('\n')

	sb.WriteString(q)
	sb.WriteRune('\n')
	sb.WriteRune('\n')

	// Args.
	sb.WriteString(argsSection)
	sb.WriteRune('\n')

	sb.Write(args)
	sb.WriteRune('\n')
	sb.WriteRune('\n')

	// Result.
	sb.WriteString(resultSection)
	sb.WriteRune('\n')

	sb.Write(result)

	return sb.String()
}
