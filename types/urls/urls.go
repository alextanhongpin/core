package urls

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// IsEqual reports whether a and b parse and have the same normalized form.
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

// Normalize parses link and returns it without query string, fragment,
// trailing slash, or default port (80 for http, 443 for https), with the
// scheme and host in lower case.
func Normalize(link string) (*url.URL, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	return normalize(u), nil
}

func normalize(u *url.URL) *url.URL {
	v := *u
	v.User = nil
	if u.User != nil {
		user := *u.User
		v.User = &user
	}
	v.RawQuery = ""
	v.ForceQuery = false
	v.Fragment = ""
	v.RawFragment = ""
	v.Scheme = strings.ToLower(v.Scheme)
	v.Host = strings.ToLower(v.Host)
	if host, port, err := net.SplitHostPort(v.Host); err == nil &&
		(v.Scheme == "http" && port == "80" || v.Scheme == "https" && port == "443") {
		v.Host = host
		if strings.Contains(host, ":") {
			v.Host = "[" + host + "]"
		}
	}
	v.Path = strings.TrimRight(v.Path, "/")
	if v.RawPath != "" {
		v.RawPath = strings.TrimRight(v.RawPath, "/")
	}
	return &v
}

// IsScopedDomain reports whether u has the same scheme and host as root,
// ignoring case and default ports.
func IsScopedDomain(root, u *url.URL) bool {
	r, v := normalize(root), normalize(u)
	return r.Scheme == v.Scheme && r.Host == v.Host
}

// IsScopedPrefix reports whether u is strictly below root: it has the same
// scheme and host, and its path continues root's path after a "/". A URL is
// not scoped under itself, and "/path-is" is not under "/path".
func IsScopedPrefix(root, u *url.URL) bool {
	if !IsScopedDomain(root, u) {
		return false
	}
	rest, ok := strings.CutPrefix(normalize(u).EscapedPath(), normalize(root).EscapedPath())
	return ok && len(rest) > 1 && rest[0] == '/'
}

// Hash returns the hex SHA-256 of u's normalized form, so URLs that differ
// only in what Normalize removes have the same hash.
func Hash(u *url.URL) string {
	sum := sha256.Sum256([]byte(normalize(u).String()))
	return hex.EncodeToString(sum[:])
}

// Extract returns the href attribute values of the <a> elements in an HTML
// document, in document order. Values are returned as written: relative
// links are not resolved and fragment-only or mailto: links are included.
// Reading stops at the end of the input or the first read error.
func Extract(r io.Reader) []string {
	var hrefs []string
	z := html.NewTokenizer(r)
	for {
		switch z.Next() {
		case html.ErrorToken:
			return hrefs
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			if string(name) != "a" {
				continue
			}
			for hasAttr {
				var key, val []byte
				key, val, hasAttr = z.TagAttr()
				if string(key) == "href" {
					hrefs = append(hrefs, string(val))
				}
			}
		}
	}
}
