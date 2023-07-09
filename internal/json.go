package internal

import (
	"bytes"
	"encoding/json"
)

func PrettyJSON(v any) ([]byte, error) {
	if b, ok := v.([]byte); ok {
		if !json.Valid(b) {
			return b, nil
		}

		var bb bytes.Buffer
		if err := json.Indent(&bb, b, "", "  "); err != nil {
			return nil, err
		}

		return bb.Bytes(), nil
	}

	return json.MarshalIndent(v, "", "  ")
}
