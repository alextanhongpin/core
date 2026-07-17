package urls_test

import (
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
