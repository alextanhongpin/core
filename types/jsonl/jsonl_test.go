package jsonl_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/alextanhongpin/core/types/jsonl"
	"github.com/go-openapi/testify/assert"
)

func open(t *testing.T, name string) *jsonl.File[int] {
	t.Helper()
	f, err := jsonl.Open[int](filepath.Join(t.TempDir(), "nested", name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func collect(t *testing.T, f *jsonl.File[int]) []int {
	t.Helper()
	seq, stop := f.ReadLines()
	lines := slices.Collect(seq)
	if err := stop(); err != nil {
		t.Fatal(err)
	}
	return lines
}

func TestJSONL(t *testing.T) {
	f := open(t, "src.jsonl")
	is := assert.New(t)
	is.Empty(collect(t, f))

	var want []int
	for i := range 10 {
		want = append(want, i)
		is.NoError(f.Write(i))
		is.Equal(want, collect(t, f))
	}

	t.Run("copy", func(t *testing.T) {
		dst := open(t, "copy.jsonl")
		assert.NoError(t, jsonl.Copy(f, dst))
		assert.Equal(t, want, collect(t, dst))
	})

	t.Run("copy func", func(t *testing.T) {
		dst := open(t, "copy_func.jsonl")
		assert.NoError(t, jsonl.CopyFunc(f, dst, func(i int) bool { return i%2 == 0 }))
		assert.Equal(t, []int{0, 2, 4, 6, 8}, collect(t, dst))
	})

	t.Run("early stop", func(t *testing.T) {
		seq, stop := f.ReadLines()
		var got []int
		for v := range seq {
			got = append(got, v)
			if v == 2 {
				break
			}
		}
		assert.NoError(t, stop())
		assert.Equal(t, []int{0, 1, 2}, got)
	})
}

func TestReadLinesDecodeError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.jsonl")
	if err := os.WriteFile(path, []byte("1\n\"x\"\n3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := jsonl.Open[int](path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	seq, stop := f.ReadLines()
	assert.Equal(t, []int{1}, slices.Collect(seq))
	assert.Error(t, stop())

	// Regression: a later iteration must not report a stale error.
	seq, stop = f.ReadLines()
	for range seq {
		break
	}
	assert.NoError(t, stop())
}

func TestRemove(t *testing.T) {
	f := open(t, "rm.jsonl")
	assert.NoError(t, f.Remove())
	_, err := os.Stat(f.Name())
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func Example() {
	dir, _ := os.MkdirTemp("", "jsonl")
	defer os.RemoveAll(dir)

	type event struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	f, err := jsonl.Open[event](filepath.Join(dir, "events.jsonl"))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_ = f.Write(event{1, "signup"})
	_ = f.Write(event{2, "login"})

	seq, stop := f.ReadLines()
	for e := range seq {
		fmt.Println(e.ID, e.Name)
	}
	if err := stop(); err != nil {
		panic(err)
	}
	// Output:
	// 1 signup
	// 2 login
}
