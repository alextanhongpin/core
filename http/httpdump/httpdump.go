package httpdump

import (
	"bytes"
	"encoding/json"
)

func normalizeNewlines(b []byte) []byte {
	// httputil.DumpRequestOut and httputil.DumpResponse
	// uses \r\n as the carrier return.
	// Replace it for readability.
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))

	// Remove the trailing new lines.
	b = bytes.TrimSpace(b)

	return b
}

func denormalizeNewlines(b []byte) []byte {
	b = bytes.TrimSpace(b)
	// Replace the new lines with the carrier returned that http.ReadRequest and
	// http.ReadResponse understands.
	parts := bytes.Split(b, []byte("\n"))
	b = bytes.Join(parts, []byte("\r\n"))

	// The file must end with this.
	b = append(b, []byte("\r\n\r\n")...)
	return b
}

func bytesPretty(b []byte) ([]byte, error) {
	if !json.Valid(b) {
		return b, nil
	}

	bb := new(bytes.Buffer)
	if err := json.Indent(bb, b, "", " "); err != nil {
		return nil, err
	}

	b = bb.Bytes()

	// We need to replace the new lines with carrier return
	// that is supported by httputil.DumpXXX method.
	// This is required as they affect the content length of
	// the body that needs to be set.
	b = denormalizeNewlines(b)
	b = bytes.TrimSpace(b)

	return b, nil
}
