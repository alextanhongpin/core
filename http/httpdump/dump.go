package httpdump

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type Dump struct {
	Line   string
	Header http.Header
	Body   *bytes.Reader
}

func (d *Dump) MarshalJSON() ([]byte, error) {
	type dump struct {
		Line   string          `json:"line"`
		Header http.Header     `json:"headers"`
		Body   json.RawMessage `json:"body"`
	}

	d.Body.Seek(0, 0)
	b, err := io.ReadAll(d.Body)
	if err != nil {
		return nil, err
	}
	d.Body.Seek(0, 0)

	// This will error if body is empty.
	// Set to nil to avoid error.
	body := json.RawMessage(b)
	if len(b) == 0 {
		body = nil
	}

	return json.Marshal(dump{
		Line:   d.Line,
		Header: d.Header,
		Body:   body,
	})
}

func (d *Dump) UnmarshalJSON(b []byte) error {
	type dump struct {
		Line   string          `json:"line"`
		Header http.Header     `json:"headers"`
		Body   json.RawMessage `json:"body"`
	}

	var a dump
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}

	d.Line = a.Line
	d.Header = a.Header
	d.Body = bytes.NewReader(a.Body)

	return nil
}
