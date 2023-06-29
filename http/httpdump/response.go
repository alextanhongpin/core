package httpdump

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
)

type Response struct {
	*http.Response
	Dump Dump
}

func NewResponse(r *http.Response) (*Response, error) {
	res := &Response{
		Response: r,
	}

	if err := res.Parse(); err != nil {
		return nil, err
	}

	return res, nil
}

func (r *Response) Parse() error {
	res, err := normalizeResponse(r.Response)
	if err != nil {
		return err
	}
	dump, err := responseToDump(res)
	if err != nil {
		return err
	}

	r.Response = res
	r.Dump = *dump

	return nil
}

func (r *Response) UnmarshalText(b []byte) error {
	b = bytes.TrimSpace(b)

	var res *http.Response

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
				var err error
				res, err = parseResponseLine(strings.NewReader(text))
				if err != nil {
					return err
				}

				break scan
			case 1:
				k, v, ok := strings.Cut(text, ": ")
				if !ok {
					return errors.New("invalid response header format")
				}
				res.Header.Add(k, v)
			case 2:
				bb.WriteString(text)
				bb.WriteString("\n")
			}
		}
	}

	body := bytes.NewReader(bytes.TrimSpace(bb.Bytes()))
	res.Body = io.NopCloser(body)

	var err error
	res, err = normalizeResponse(res)
	if err != nil {
		return err
	}

	dump, err := responseToDump(res)
	if err != nil {
		return err
	}

	r.Response = res
	r.Dump = *dump

	return nil
}

func (r *Response) MarshalText() ([]byte, error) {
	res, err := httputil.DumpResponse(r.Response, true)
	if err != nil {
		return nil, err
	}

	res = NormalizeNewlines(res)
	res = bytes.TrimSpace(res)

	return res, nil
}

func (r *Response) MarshalJSON() ([]byte, error) {
	return r.Dump.MarshalJSON()
}

func (r *Response) UnmarshalJSON(b []byte) error {
	var dump Dump
	if err := json.Unmarshal(b, &dump); err != nil {
		return err
	}

	res, err := dumpToResponse(&dump)
	if err != nil {
		return err
	}

	r.Dump = dump
	r.Response = res

	return nil
}

func normalizeResponse(res *http.Response) (*http.Response, error) {
	// Prettify the request body.
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	b, err = prettyBytes(b)
	if err != nil {
		return nil, err
	}

	res.Body = io.NopCloser(bytes.NewReader(b))

	return res, nil
}

func dumpToResponse(dump *Dump) (*http.Response, error) {
	res, err := parseResponseLine(strings.NewReader(dump.Line))
	if err != nil {
		return nil, err
	}

	res.Header = dump.Header.Clone()
	res.Body = io.NopCloser(dump.Body)

	return normalizeResponse(res)
}

func responseToDump(res *http.Response) (*Dump, error) {
	resLine := formatResponseLine(res)

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	body := bytes.NewReader(b)
	res.Body = io.NopCloser(body)

	return &Dump{
		Line:   resLine,
		Header: res.Header.Clone(),
		Body:   body,
	}, nil
}

func formatResponseLine(r *http.Response) string {
	// Status line
	text := r.Status
	if text == "" {
		text = http.StatusText(r.StatusCode)
		if text == "" {
			text = "status code " + strconv.Itoa(r.StatusCode)
		}
	} else {
		// Just to reduce stutter, if user set r.Status to "200 OK" and StatusCode to 200.
		// Not important.
		text = strings.TrimPrefix(text, strconv.Itoa(r.StatusCode)+" ")
	}

	return fmt.Sprintf("HTTP/%d.%d %03d %s", r.ProtoMajor, r.ProtoMinor, r.StatusCode, text)
}

func parseResponseLine(r io.Reader) (*http.Response, error) {
	var w http.Response
	if _, err := fmt.Fscanf(r, "HTTP/%d.%d %03d",
		&w.ProtoMajor,
		&w.ProtoMinor,
		&w.StatusCode,
	); err != nil {
		return nil, err
	}
	w.Header = make(http.Header)

	return &w, nil
}
