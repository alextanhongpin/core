package urls

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func IsEqual(a, b string) bool {
	c, err := Normalize(a)
	if err != nil {
		return false
	}
	d, err := Normalize(b)
	if err != nil {
		return false
	}
	return c.String() == d.String()
}

// Normalize removes trailing slash, as well as removing fragments and query
// string in the URL.
func Normalize(link string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSuffix(link, "/"))
	if err != nil {
		return nil, err
	}
	u.Fragment = ""
	u.RawQuery = ""
	return u, nil
}

func IsScopedDomain(root, u *url.URL) bool {
	return root.Host == u.Host &&
		root.Scheme == u.Scheme
}

func IsScopedPrefix(root, u *url.URL) bool {
	path := strings.TrimPrefix(u.String(), root.String())
	return strings.HasPrefix(path, "/")
}

func Hash(u *url.URL) string {
	v, err := Normalize(u.String())
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256([]byte(v.String()))
	return fmt.Sprintf("%x", sum)
}

// Extract parses HTML and returns all href values.
func Extract(htmlContent io.Reader) []string {
	var urls []string
	tokenizer := html.NewTokenizer(htmlContent)

	for {
		tokenType := tokenizer.Next()

		switch tokenType {
		case html.ErrorToken:
			// End of the document or invalid HTML
			return urls

		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()

			// Check if the tag is an anchor tag <a>
			if token.Data == "a" {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						urls = append(urls, attr.Val)
					}
				}
			}
		}
	}
}
