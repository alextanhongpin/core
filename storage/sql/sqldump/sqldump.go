package sqldump

import (
	"bytes"

	"github.com/alextanhongpin/core/internal"
	"golang.org/x/tools/txtar"
)

const querySection = "query"
const argsSection = "args"
const normalizedSection = "normalized"
const varsSection = "vars"
const resultSection = "result"

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

func Read(b []byte) (*SQL, error) {
	d := new(SQL)

	arc := txtar.Parse(b)
	for _, f := range arc.Files {
		f := f
		name, data := f.Name, f.Data
		data = bytes.TrimSpace(data)

		switch name {
		case querySection:
			d.Query = string(data)
		case normalizedSection:
			d.Normalized = string(data)
		case argsSection:
			a, err := internal.UnmarshalYAMLPreserveKeysOrder[any](data)
			if err != nil {
				return nil, err
			}
			d.ArgMap = a
		case varsSection:
			a, err := internal.UnmarshalYAMLPreserveKeysOrder[any](data)
			if err != nil {
				return nil, err
			}
			d.VarMap = a
		case resultSection:
			a, err := internal.UnmarshalYAMLPreserveKeysOrder[any](data)
			if err != nil {
				return nil, err
			}
			d.Result = a
		}
	}

	return d, nil
}

func dump(q string, args []byte, n string, varMap, result []byte) []byte {
	arc := new(txtar.Archive)
	// Query.
	arc.Files = append(arc.Files, txtar.File{
		Name: querySection,
		Data: appendNewLine([]byte(q)),
	})

	// Args.
	if len(args) != 0 {
		arc.Files = append(arc.Files, txtar.File{
			Name: argsSection,
			Data: appendNewLine(args),
		})
	}

	// Normalized.
	arc.Files = append(arc.Files, txtar.File{
		Name: normalizedSection,
		Data: appendNewLine([]byte(n)),
	})

	// Vars.
	if len(varMap) != 0 {
		arc.Files = append(arc.Files, txtar.File{
			Name: varsSection,
			Data: appendNewLine(varMap),
		})
	}

	// Result.
	if result != nil {
		arc.Files = append(arc.Files, txtar.File{
			Name: resultSection,
			Data: result,
		})
	}

	return txtar.Format(arc)
}

func appendNewLine(b []byte) []byte {
	b = append(b, '\n')
	b = append(b, '\n')
	return b
}
