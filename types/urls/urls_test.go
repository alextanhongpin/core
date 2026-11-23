package urls_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/types/must"
	"github.com/alextanhongpin/core/types/urls"
	"github.com/go-openapi/testify/assert"
)

func TestIsScopedPrefix(t *testing.T) {
	tests := []struct {
		name  string
		base  string
		child string
		want  bool
	}{
		{
			name:  "equal",
			base:  "http://example.com/",
			child: "http://example.com/",
		},
		{
			name:  "equal base slash",
			base:  "http://example.com/",
			child: "http://example.com",
		},
		{
			name:  "equal child slash",
			base:  "http://example.com",
			child: "http://example.com/",
		},
		{
			name:  "is suburl",
			want:  true,
			base:  "http://example.com",
			child: "http://example.com/path",
		},
		{
			name:  "prefix",
			base:  "http://example.com/path",
			child: "http://example.com/path-is",
		},
		{
			want:  true,
			name:  "nested path",
			base:  "http://example.com/path",
			child: "http://example.com/path/is/valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := must.Value(urls.Normalize(tt.base))
			c := must.Value(urls.Normalize(tt.child))
			got := urls.IsScopedPrefix(b, c)

			is := assert.New(t)
			is.Equal(tt.want, got)
			t.Logf("urls.IsScopedPrefix(%q, %q) = %t", tt.base, tt.child, got)
		})
	}
}

func TestIsScopedDomain(t *testing.T) {
	tests := []struct {
		name  string
		base  string
		child string
		want  bool
	}{
		{
			want:  true,
			name:  "equal",
			base:  "http://example.com/",
			child: "http://example.com/",
		},
		{
			name:  "equal base slash",
			base:  "http://example.com/",
			child: "http://example.com",
			want:  true,
		},
		{
			name:  "equal child slash",
			base:  "http://example.com",
			child: "http://example.com/",
			want:  true,
		},
		{
			name:  "is suburl",
			base:  "http://example.com",
			child: "http://example.com/path",
			want:  true,
		},
		{
			name:  "prefix",
			base:  "http://example.com/path",
			child: "http://example.com/path-is",
			want:  true,
		},
		{
			name:  "nested path",
			base:  "http://example.com/path",
			child: "http://example.com/path/is/valid",
			want:  true,
		},
		{
			name:  "wrong domain",
			base:  "http://example/path",
			child: "http://example.com/path/is/valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := must.Value(urls.Normalize(tt.base))
			c := must.Value(urls.Normalize(tt.child))
			got := urls.IsScopedDomain(b, c)

			is := assert.New(t)
			is.Equal(tt.want, got)
			t.Logf("urls.IsScopedDomain(%q, %q) = %t", tt.base, tt.child, got)
		})
	}
}

// Regression: IsScopedPrefix compared raw strings, so a root with a trailing
// slash or a different host case rejected URLs below it.
func TestIsScopedPrefixUnnormalized(t *testing.T) {
	root := must.Value(url.Parse("http://Example.com/docs/"))
	assert.True(t, urls.IsScopedPrefix(root, must.Value(url.Parse("http://example.com/docs/a?x=1"))))
	assert.False(t, urls.IsScopedPrefix(root, must.Value(url.Parse("https://example.com/docs/a"))))
	assert.False(t, urls.IsScopedPrefix(root, must.Value(url.Parse("http://example.com.evil/docs/a"))))
}

func TestNormalize(t *testing.T) {
	tests := map[string]string{
		"http://example.com/":             "http://example.com",
		"HTTP://Example.COM/Path/":        "http://example.com/Path",
		"http://example.com:80/a":         "http://example.com/a",
		"https://example.com:443/a":       "https://example.com/a",
		"https://example.com:8443/a":      "https://example.com:8443/a",
		"http://[::1]:80/a":               "http://[::1]/a",
		"http://example.com/a?b=1#c":      "http://example.com/a",
		"http://example.com/?":            "http://example.com", // regression: ForceQuery kept "?"
		"http://example.com/a?next=/b/":   "http://example.com/a",
		"http://example.com/a%2Fb/":       "http://example.com/a%2Fb",
		"http://user:pass@example.com/a/": "http://user:pass@example.com/a",
	}
	for in, want := range tests {
		got, err := urls.Normalize(in)
		assert.NoError(t, err, in)
		assert.Equal(t, want, got.String(), in)
	}
	_, err := urls.Normalize("http://%zz")
	assert.Error(t, err)
}

func TestNormalizeDoesNotModifyInput(t *testing.T) {
	u := must.Value(url.Parse("http://Example.com/a/?q=1"))
	_ = urls.Hash(u)
	_ = urls.IsScopedDomain(u, u)
	assert.Equal(t, "http://Example.com/a/?q=1", u.String())
}

func TestIsEqualAndHash(t *testing.T) {
	assert.True(t, urls.IsEqual("http://example.com/a/", "HTTP://EXAMPLE.com/a#top"))
	assert.False(t, urls.IsEqual("http://example.com/a", "http://example.com/b"))
	assert.False(t, urls.IsEqual("http://%zz", "http://%zz"))

	a := urls.Hash(must.Value(url.Parse("http://example.com/a/")))
	b := urls.Hash(must.Value(url.Parse("http://example.com/a?x=1")))
	assert.Equal(t, a, b)
	assert.Len(t, a, 64)
}

func TestExtract(t *testing.T) {
	doc := `<html><body>
<a href="/one">1</a>
<A HREF="https://example.com/two">2</A>
<a name="x">no href</a>
<link href="/style.css">
<a href="#frag"/>
</body></html>`
	assert.Equal(t, []string{"/one", "https://example.com/two", "#frag"}, urls.Extract(strings.NewReader(doc)))
	assert.Empty(t, urls.Extract(strings.NewReader("")))
}

func Example() {
	u, _ := urls.Normalize("HTTPS://Example.com:443/docs/?page=2#intro")
	fmt.Println(u)

	root, _ := url.Parse("https://example.com/docs")
	fmt.Println(urls.IsScopedPrefix(root, must.Value(url.Parse("https://example.com/docs/install"))))
	fmt.Println(urls.IsScopedPrefix(root, must.Value(url.Parse("https://example.com/docs-old"))))
	// Output:
	// https://example.com/docs
	// true
	// false
}
