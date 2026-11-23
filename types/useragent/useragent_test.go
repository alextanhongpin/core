package useragent_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/types/useragent"
	"github.com/go-openapi/testify/assert"
)

const page = `<html><body>
<div id="menu"><a href="/x">Not an agent</a></div>
<div id="liste">
  <h3>Google</h3>
  <ul>
    <li><a href="/1">Mozilla/5.0 (X11; Linux x86_64) Firefox/120.0</a></li>
    <li><a href="/2"><b>Mozilla/5.0</b> (Windows NT 10.0) Firefox/119.0</a></li>
    <li><a href="/3"></a></li>
    <li><a href="/4">Chrome</a></li>
  </ul>
</div>
</body></html>`

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func client(status int, body string, seen *[]string) *http.Client {
	return &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		if seen != nil {
			*seen = append(*seen, r.URL.String())
		}
		return &http.Response{
			StatusCode: status,
			Status:     http.StatusText(status),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})}
}

var want = []string{
	"Mozilla/5.0 (X11; Linux x86_64) Firefox/120.0",
	"Mozilla/5.0 (Windows NT 10.0) Firefox/119.0",
}

// Regression: an empty <a> used to panic on a nil FirstChild.
func TestParse(t *testing.T) {
	got, err := useragent.Parse(strings.NewReader(page))
	assert.NoError(t, err)
	assert.Equal(t, want, got)

	got, err = useragent.Parse(strings.NewReader("<html></html>"))
	assert.NoError(t, err)
	assert.Empty(t, got)
}

// Regression: browser names with spaces were not query-escaped.
func TestFetch(t *testing.T) {
	var seen []string
	got, err := useragent.Fetch(context.Background(), client(200, page, &seen), useragent.InternetExplorer)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, []string{"http://www.useragentstring.com/pages/useragentstring.php?name=Internet+Explorer"}, seen)
}

func TestFetchStatus(t *testing.T) {
	_, err := useragent.Fetch(context.Background(), client(503, page, nil), useragent.Firefox)
	assert.ErrorContains(t, err, "unexpected status")
}

func TestFetchTransportError(t *testing.T) {
	c := &http.Client{Transport: roundTripper(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("offline")
	})}
	_, err := useragent.Fetch(context.Background(), c, useragent.Firefox)
	assert.ErrorContains(t, err, "offline")
}

func TestLoader(t *testing.T) {
	l := useragent.NewLoader([]string{useragent.Firefox, useragent.Safari})
	l.Client = client(200, page, nil)

	// Regression: Load used to panic before the first ReadFrom.
	assert.Nil(t, l.Load())

	var buf bytes.Buffer
	n, err := l.WriteTo(&buf)
	assert.NoError(t, err)
	assert.Equal(t, int64(buf.Len()), n)
	assert.Equal(t, strings.Repeat(strings.Join(want, "\n")+"\n", 2), buf.String())

	m, err := l.ReadFrom(strings.NewReader(buf.String() + "\n"))
	assert.NoError(t, err)
	assert.Equal(t, int64(buf.Len()+1), m)
	assert.Equal(t, append(append([]string{}, want...), want...), l.Load())

	got := l.Load()
	got[0] = "changed"
	assert.Equal(t, want[0], l.Load()[0])
}

func TestLoaderWriteToError(t *testing.T) {
	l := useragent.NewLoader([]string{useragent.Firefox})
	l.Client = client(404, "", nil)
	_, err := l.WriteTo(io.Discard)
	assert.Error(t, err)
}

func ExampleLoader_ReadFrom() {
	l := useragent.NewLoader(nil)
	if _, err := l.ReadFrom(strings.NewReader("AgentA/1.0\nAgentB/2.0\n")); err != nil {
		panic(err)
	}
	fmt.Println(l.Load())
	// Output: [AgentA/1.0 AgentB/2.0]
}
