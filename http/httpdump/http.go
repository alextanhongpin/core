package httpdump

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

var ErrInvalidDumpFormat = errors.New("invalid http dump format")
var sep = []byte("\n###\n")

type HTTP struct {
	w *Response
	r *Request
}

func NewHTTP(w *http.Response, r *http.Request) (*HTTP, error) {
	req, err := NewRequest(r)
	if err != nil {
		return nil, err
	}

	res, err := NewResponse(w)
	if err != nil {
		return nil, err
	}

	return &HTTP{
		w: res,
		r: req,
	}, nil
}

func (h *HTTP) Request() *Request {
	return h.r
}

func (h *HTTP) Response() *Response {
	return h.w
}

func (d *HTTP) MarshalText() ([]byte, error) {
	req, err := d.r.MarshalText()
	if err != nil {
		return nil, err
	}

	res, err := d.w.MarshalText()
	if err != nil {
		return nil, err
	}

	out := [][]byte{req, sep, res}

	return bytes.Join(out, []byte("\n\n")), nil
}

func (d *HTTP) UnmarshalText(b []byte) error {
	req, res, ok := bytes.Cut(b, sep)
	if !ok {
		return ErrInvalidDumpFormat
	}

	r := new(Request)
	if err := r.UnmarshalText(req); err != nil {
		return err
	}

	w := new(Response)
	if err := w.UnmarshalText(res); err != nil {
		return err
	}

	d.w = w
	d.r = r

	return nil
}

func (d *HTTP) MarshalJSON() ([]byte, error) {
	type data struct {
		Request  *Request  `json:"request"`
		Response *Response `json:"response"`
	}

	return json.Marshal(data{
		Request:  d.r,
		Response: d.w,
	})
}

func (d *HTTP) UnmarshalJSON(b []byte) error {
	type data struct {
		Request  *Request  `json:"request"`
		Response *Response `json:"response"`
	}

	var r data
	if err := json.Unmarshal(b, &r); err != nil {
		return err
	}

	d.w = r.Response
	d.r = r.Request

	return nil
}