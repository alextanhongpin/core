package httpdump

import (
	"bytes"
	"encoding/json"
)

// NormalizeNewlines normalizes \r\n (windows) and \r (mac)
// into \n (unix)
// Reference [here].
// [here]: https://www.programming-books.io/essential/go/normalize-newlines-1d3abcf6f17c4186bb9617fa14074e48
func NormalizeNewlines(d []byte) []byte {
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = bytes.Replace(d, []byte{13, 10}, []byte{10}, -1)
	// replace CF \r (mac) with LF \n (unix)
	d = bytes.Replace(d, []byte{13}, []byte{10}, -1)
	return d
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
