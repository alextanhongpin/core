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
		Body   string      `json:"body"`
	}

	d.Body.Seek(0, 0)
	b, err := io.ReadAll(d.Body)
	if err != nil {
		return nil, err
	}
	d.Body.Seek(0, 0)

	return json.Marshal(dump{
		Line:   d.Line,
		Header: d.Header,
		Body:   string(b),
	})
}

func (d *Dump) UnmarshalJSON(b []byte) error {
	type dump struct {
		Line   string      `json:"line"`
		Header http.Header `json:"headers"`
		Body   string      `json:"body"`
	}

	var a dump
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}

	d.Line = a.Line
	d.Header = a.Header
	d.Body = bytes.NewReader([]byte(a.Body))

	return nil
}
