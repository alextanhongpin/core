package httpdump

import (
	"bytes"
	"encoding/json"
)

func normalizeNewlines(b []byte) []byte {
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	b = bytes.ReplaceAll(b, []byte("\r"), []byte("\n"))
	b = bytes.TrimSpace(b)
	return b
}

func denormalizeNewlines(b []byte) []byte {
	b = bytes.TrimSpace(b)
	parts := bytes.Split(b, []byte("\n"))
	b = bytes.Join(parts, []byte("\r\n"))
	b = append(b, []byte("\r\n\r\n")...)
	return b
}

func prettyBytes(b []byte) ([]byte, error) {
	if !json.Valid(b) {
		return b, nil
	}

	bb := new(bytes.Buffer)
	if err := json.Indent(bb, b, "", " "); err != nil {
		return nil, err
	}

	return bb.Bytes(), nil
}
