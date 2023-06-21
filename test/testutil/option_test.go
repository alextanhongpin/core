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
			name: "testdir",
			opts: []testutil.JSONOption{
				testutil.TestDir("./testdir"),
			},
			want: "testdir",
		},
		{
			name: "filepath",
			opts: []testutil.JSONOption{
				testutil.FilePath("foo"),
			},
			want: "testdata/foo",
		},
		{
			name: "filepath nested",
			opts: []testutil.JSONOption{
				testutil.FilePath("foo/bar"),
			},
			want: "testdata/foo/bar",
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
			want: "testdata/baz.",
		},
		{
			name: "file extension with leading dot",
			opts: []testutil.JSONOption{
				testutil.FileExt(".json"),
			},
			want: "testdata/.json",
		},
		{
			name: "file extension without leading dot",
			opts: []testutil.JSONOption{
				testutil.FileExt("json"),
			},
			want: "testdata.json",
		},
		{
			name: "file ext takes precedence over filename with extension",
			opts: []testutil.JSONOption{
				testutil.FileName("foo.yaml"),
				testutil.FileExt("json"),
			},
			want: "testdata/foo.json",
		},
	}

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			path := testutil.NewPathOption(ts.opts...).String()
			assert.Equal(t, ts.want, path)
		})
	}
}
