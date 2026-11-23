package useragent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Browser names accepted by useragentstring.com.
const (
	Chrome           = "Chrome"
	Edge             = "Edge"
	Firefox          = "Firefox"
	InternetExplorer = "Internet Explorer"
	Mozilla          = "Mozilla"
	Opera            = "Opera"
	OperaMini        = "Opera Mini"
	Safari           = "Safari"
)

// Browsers lists the browser name constants.
var Browsers = []string{
	Chrome,
	Edge,
	Firefox,
	InternetExplorer,
	Mozilla,
	Opera,
	OperaMini,
	Safari,
}

const baseURL = "http://www.useragentstring.com/pages/useragentstring.php?name="

// Fetch downloads the user-agent strings listed for browser on
// useragentstring.com. A nil client uses http.DefaultClient. It returns an
// error for a non-200 response.
func Fetch(ctx context.Context, client *http.Client, browser string) ([]string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+url.QueryEscape(browser), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("useragent: fetch %s: %w", browser, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("useragent: fetch %s: unexpected status %s", browser, resp.Status)
	}
	return Parse(resp.Body)
}

// Parse extracts the user-agent strings from a useragentstring.com listing
// page: the text of the links inside the element with id "liste", skipping
// the "Google" and "Chrome" section headings.
func Parse(r io.Reader) ([]string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("useragent: parse: %w", err)
	}
	var res []string
	for n := range doc.Descendants() {
		if n.Type != html.ElementNode || !hasID(n, "liste") {
			continue
		}
		for a := range n.Descendants() {
			if a.Type != html.ElementNode || a.DataAtom != atom.A {
				continue
			}
			text := strings.TrimSpace(textOf(a))
			if text == "" || text == "Google" || text == "Chrome" {
				continue
			}
			res = append(res, text)
		}
		break
	}
	return res, nil
}

func hasID(n *html.Node, id string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "id" && attr.Val == id {
			return true
		}
	}
	return false
}

func textOf(n *html.Node) string {
	var b strings.Builder
	for c := range n.Descendants() {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
	}
	return b.String()
}

// Loader holds a list of user-agent strings. WriteTo fetches them and
// ReadFrom loads them back, so a Loader can be cached in a file, for
// example with markdown.Loader. Load and ReadFrom are safe for concurrent
// use.
type Loader struct {
	// Client is used by WriteTo; nil means http.DefaultClient.
	Client *http.Client

	val      atomic.Pointer[[]string]
	browsers []string
}

// NewLoader returns a Loader that fetches the given browsers.
func NewLoader(browsers []string) *Loader {
	return &Loader{browsers: slices.Clone(browsers)}
}

// Load returns a copy of the strings last read by ReadFrom, or nil before
// the first successful ReadFrom.
func (l *Loader) Load() []string {
	p := l.val.Load()
	if p == nil {
		return nil
	}
	return slices.Clone(*p)
}

// ReadFrom replaces the loaded strings with the non-empty lines of r. On
// error the loaded strings are unchanged.
func (l *Loader) ReadFrom(r io.Reader) (int64, error) {
	var n int64
	var res []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		n += int64(len(text)) + 1
		if text = strings.TrimSpace(text); text != "" {
			res = append(res, text)
		}
	}
	if err := scanner.Err(); err != nil {
		return n, err
	}
	l.val.Store(&res)
	return n, nil
}

// WriteTo fetches each configured browser in turn and writes one string
// per line. It stops at the first fetch or write error.
func (l *Loader) WriteTo(w io.Writer) (int64, error) {
	var total int64
	for _, b := range l.browsers {
		uas, err := Fetch(context.Background(), l.Client, b)
		if err != nil {
			return total, err
		}
		for _, ua := range uas {
			n, err := fmt.Fprintln(w, ua)
			total += int64(n)
			if err != nil {
				return total, err
			}
		}
	}
	return total, nil
}
