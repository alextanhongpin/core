package httpdump

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func logError(err error) (b bool) {
	if _, ok := os.LookupEnv("DEBUG"); !ok {
		return
	}

	if err != nil {
		pkg := "github.com/alextanhongpin"
		pretty := func(s string) string {
			parts := strings.Split(s, pkg)
			part := parts[len(parts)-1]
			return strings.TrimPrefix(part, "/")
		}

		// Notice that we're using 1, so it will actually log the where
		// the error happened, 0 = this function, we don't want that.
		pc, filename, line, _ := runtime.Caller(1)
		fn := runtime.FuncForPC(pc).Name()

		fmt.Printf("%s[%s:%d]: %v\n", pretty(fn), pretty(filename), line, err)
		b = true
	}

	return
}

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
		logError(err)
		return err
	}
	dump, err := responseToDump(res)
	if err != nil {
		logError(err)
		return err
	}

	r.Response = res
	r.Dump = *dump

	return nil
}

func (r *Response) UnmarshalText(b []byte) error {
	b = normalizeNewlines(b)
	b = denormalizeNewlines(b)

	res, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(b)), nil)
	if err != nil {
		logError(err)
		return err
	}

	res, err = normalizeResponse(res)
	if err != nil {
		logError(err)
		return err
	}

	dump, err := responseToDump(res)
	if err != nil {
		logError(err)
		return err
	}

	r.Response = res
	r.Dump = *dump

	return nil
}

func (r *Response) MarshalText() ([]byte, error) {
	res, err := httputil.DumpResponse(r.Response, true)
	if err != nil {
		logError(err)
		return nil, err
	}

	res = normalizeNewlines(res)

	return res, nil
}

func (r *Response) MarshalJSON() ([]byte, error) {
	return r.Dump.MarshalJSON()
}

func (r *Response) UnmarshalJSON(b []byte) error {
	var dump Dump
	if err := json.Unmarshal(b, &dump); err != nil {
		logError(err)
		return err
	}

	res, err := dumpToResponse(&dump)
	if err != nil {
		logError(err)
		return fmt.Errorf("UnmarshalJSON: %w", err)
	}

	r.Dump = dump
	r.Response = res

	return nil
}

func normalizeResponse(res *http.Response) (*http.Response, error) {
	// Prettify the request body.
	b, err := io.ReadAll(res.Body)
	if err != nil {
		logError(err)
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	b, err = prettyBytes(b)
	if err != nil {
		logError(err)
		return nil, fmt.Errorf("failed to prettify body: %w", err)
	}

	b = denormalizeNewlines(b)
	b = bytes.TrimSpace(b)
	res.Body = io.NopCloser(bytes.NewReader(b))

	return res, nil
}

func dumpToResponse(dump *Dump) (*http.Response, error) {
	res, err := parseResponseLine(strings.NewReader(dump.Line))
	if err != nil {
		logError(err)
		return nil, err
	}

	res.Header = dump.Header.Clone()
	res.Body = io.NopCloser(dump.Body)
	res.Trailer = dump.Trailer.Clone()

	return normalizeResponse(res)
}

func responseToDump(res *http.Response) (*Dump, error) {
	resLine := formatResponseLine(res)

	b, err := io.ReadAll(res.Body)
	if err != nil {
		logError(err)
		return nil, fmt.Errorf("responseToDump: %w", err)
	}

	body := bytes.NewReader(b)
	res.Body = io.NopCloser(body)

	return &Dump{
		Line:    resLine,
		Header:  res.Header.Clone(),
		Body:    body,
		Trailer: res.Trailer.Clone(),
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
		logError(err)
		return nil, err
	}
	w.Header = make(http.Header)

	return &w, nil
}
