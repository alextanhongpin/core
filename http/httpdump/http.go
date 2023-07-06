package httpdump

import (
	"bytes"
	"errors"
	"net/http"
)

var ErrInvalidDumpFormat = errors.New("invalid http dump format")

var sep = []byte("\n###\n")

func DumpHTTP(w *http.Response, r *http.Request) ([]byte, error) {
	req, err := DumpRequest(r)
	if err != nil {
		return nil, err
	}

	res, err := DumpResponse(w)
	if err != nil {
		return nil, err
	}

	out := [][]byte{req, sep, res}

	return bytes.Join(out, []byte("\n\n")), nil
}

func ReadHTTP(b []byte) (w *http.Response, r *http.Request, err error) {
	req, res, ok := bytes.Cut(b, sep)
	if !ok {
		return nil, nil, ErrInvalidDumpFormat
	}

	r, err = ReadRequest(req)
	if err != nil {
		return
	}

	w, err = ReadResponse(res)
	if err != nil {
		return
	}

	return
}
