package httpdump

import (
	"bufio"
	"bytes"
	"errors"
	"net/http"
)

var ErrInvalidDumpFormat = errors.New("invalid http dump format")

var sep = []byte("###")

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
	scanner := bufio.NewScanner(bytes.NewReader(b))
	req := scanCond(scanner, func(b []byte) bool {
		return bytes.Equal(b, sep)
	})
	res := scanCond(scanner, func(b []byte) bool {
		return false
	})

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

func scanCond(scanner *bufio.Scanner, cond func(b []byte) bool) []byte {
	bb := new(bytes.Buffer)
	for scanner.Scan() {
		b := scanner.Bytes()
		if cond(b) {
			return bb.Bytes()
		}
		bb.Write(b)
		bb.WriteRune('\n')
	}

	return bb.Bytes()
}
