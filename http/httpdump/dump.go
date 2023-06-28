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
		Line   string      `json:"line"`
		Header http.Header `json:"headers"`
		Body   any         `json:"body"`
	}

	b, err := io.ReadAll(d.Body)
	if err != nil {
		return nil, err
	}
	d.Body.Seek(0, 0)
	var a any
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, err
	}

	return json.Marshal(dump{
		Line:   d.Line,
		Header: d.Header,
		Body:   a,
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
