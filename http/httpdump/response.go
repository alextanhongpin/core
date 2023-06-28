package httpdump

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
)

type Response struct {
	*http.Response
	Dump
}

func NewResponse(r *http.Response) *Response {
	return &Response{
		Response: r,
	}
}

func (r *Response) UnmarshalBinary(b []byte) error {
	b = bytes.TrimSpace(b)

	w := new(http.Response)
	w.Header = make(http.Header)

	scanner := bufio.NewScanner(bytes.NewReader(b))
	var bb bytes.Buffer

	sections := 3
	for i := 0; i < sections; i++ {
	scan:
		for scanner.Scan() {
			text := scanner.Text()
			if len(text) == 0 {
				break scan
			}

			switch i {
			case 0:
				if _, err := fmt.Fscanf(strings.NewReader(text), "HTTP/%d.%d %03d",
					&w.ProtoMajor,
					&w.ProtoMinor,
					&w.StatusCode,
				); err != nil {
					return err
				}

				r.Dump.Line = text

				break scan
			case 1:
				k, v, ok := strings.Cut(text, ": ")
				if !ok {
					return errors.New("invalid response header format")
				}
				w.Header.Add(k, v)
			case 2:
				bb.WriteString(text)
				bb.WriteString("\n")
			}
		}
	}

	body := bytes.NewReader(bytes.TrimSpace(bb.Bytes()))
	w.Body = io.NopCloser(body)

	r.Dump.Header = w.Header.Clone()
	r.Dump.Body = body

	r.Response = new(http.Response)
	*r.Response = *w

	return nil
}

func (r *Response) MarshalBinary() ([]byte, error) {
	w := r.Response

	b, err := io.ReadAll(w.Body)
	if err != nil {
		return nil, err
	}

	b, err = prettyBytes(b)
	if err != nil {
		return nil, err
	}

	w.Body = io.NopCloser(bytes.NewReader(b))

	res, err := httputil.DumpResponse(w, true)
	if err != nil {
		return nil, err
	}

	res = NormalizeNewlines(res)
	res = bytes.TrimSpace(res)

	return res, nil
}
