package useragent

import (
	"bufio"
	"fmt"
	"io"
	"iter"
	"net/http"
	"slices"
	"sync/atomic"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

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

func For(browser string) (iter.Seq[string], error) {
	baseURL := "http://www.useragentstring.com/pages/useragentstring.php?name="
	resp, err := http.Get(fmt.Sprintf("%s%s", baseURL, browser))
	if err != nil {
		return nil, fmt.Errorf("http.Get error: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("html.Parse error: %w", err)
	}

	seq := func(yield func(string) bool) {
		for n := range doc.Descendants() {
			if n.Type == html.ElementNode && n.DataAtom == atom.Div {
				for _, attr := range n.Attr {
					if attr.Key == "id" && attr.Val == "liste" {
						for nn := range n.Descendants() {
							if nn.Type == html.ElementNode && nn.DataAtom == atom.A {
								text := nn.FirstChild.Data
								if slices.Contains([]string{"Google", "Chrome"}, text) {
									continue
								}
								if !yield(text) {
									break
								}
							}
						}
						break
					}
				}
			}
		}
	}

	return seq, nil
}

type loader interface {
	io.ReaderFrom
	io.WriterTo
}

var _ loader = (*Loader)(nil)

type Loader struct {
	val      atomic.Pointer[[]string]
	browsers []string
}

func NewLoader(browsers []string) *Loader {
	return &Loader{browsers: browsers}
}

func (l *Loader) Load() []string {
	p := l.val.Load()
	return slices.Clone(*p)
}

func (l *Loader) ReadFrom(r io.Reader) (int64, error) {
	var n int
	var res []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		n += len(text)
		res = append(res, text)
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	l.val.Store(&res)

	return int64(n), nil
}

func (l *Loader) WriteTo(w io.Writer) (int64, error) {
	var total int
	for _, b := range l.browsers {
		seq, err := For(b)
		if err != nil {
			return 0, err
		}
		for ua := range seq {
			n, err := fmt.Fprintf(w, "%s\n", ua)
			if err != nil {
				return 0, err
			}
			total += n
		}
	}

	return int64(total), nil
}
