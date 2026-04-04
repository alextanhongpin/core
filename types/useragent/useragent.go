package useragent

import (
	"bufio"
	"fmt"
	"io"
	"iter"
	"net/http"
	"slices"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var Browsers = []string{"Chrome", "Firefox", "Mozilla", "Safari", "Opera", "Opera Mini", "Edge", "Internet Explorer"}

type Loader struct {
	browsers []string
}

func NewLoader(browsers []string) *Loader {
	return &Loader{browsers: browsers}
}

func (l *Loader) Read(r io.Reader) ([]string, error) {
	var res []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		res = append(res, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

func (l *Loader) Write(w io.Writer) error {
	for _, b := range l.browsers {
		seq, stop := Fetch(b)
		for ua := range seq {
			fmt.Fprintf(w, "%s\n", ua)
		}
		if err := stop(); err != nil {
			return err
		}
	}

	return nil
}

func Fetch(browser string) (iter.Seq[string], func() error) {
	baseURL := "http://www.useragentstring.com/pages/useragentstring.php?name="
	var iterErr error
	seq := func(yield func(string) bool) {
		resp, err := http.Get(fmt.Sprintf("%s%s", baseURL, browser))
		if err != nil {
			iterErr = err
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		iterUA, err := parseUserAgentFromHTML(resp.Body)
		if err != nil {
			iterErr = err
			return
		}

		for ua := range iterUA {
			if !yield(ua) {
				break
			}
		}
	}

	return seq, func() error {
		return iterErr
	}
}

func parseUserAgentFromHTML(r io.Reader) (iter.Seq[string], error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
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
