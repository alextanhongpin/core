package testutil_test

import (
	"testing"

	"github.com/alextanhongpin/core/test/testutil"
	"github.com/stretchr/testify/assert"
)

func TestPathOption(t *testing.T) {

	tests := []struct {
		name string
		opts []testutil.JSONOption
		want string
	}{
		{
			name: "filepath",
			opts: []testutil.JSONOption{
				testutil.FilePath("foo"),
			},
			want: "testdata/foo.json",
		},
		{
			name: "filepath nested",
			opts: []testutil.JSONOption{
				testutil.FilePath("foo/bar"),
			},
			want: "testdata/foo/bar.json",
		},
		{
			name: "filename with extension",
			opts: []testutil.JSONOption{
				testutil.FileName("baz.json"),
			},
			want: "testdata/baz.json",
		},
		{
			name: "filename without extension",
			opts: []testutil.JSONOption{
				testutil.FileName("baz"),
			},
			want: "testdata/baz.json",
		},
		{
			name: "filename with non-json extension",
			opts: []testutil.JSONOption{
				testutil.FileName("baz.yaml"),
			},
			want: "testdata/baz.yaml.json",
		},
	}

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			path := testutil.NewJSONPath(ts.opts...).String()
			assert.Equal(t, ts.want, path)
		})
	}
}
