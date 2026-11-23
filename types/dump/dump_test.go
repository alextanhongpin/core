package dump_test

import (
	"strings"
	"testing"

	"github.com/alextanhongpin/core/types/dump"
)

func TestCycles(t *testing.T) {
	m := map[string]any{}
	m["self"] = m
	s := make([]any, 1)
	s[0] = s
	type node struct{ Next *node }
	n := &node{}
	n.Next = n
	for _, value := range []any{m, s, n} {
		if got := dump.SDump(value); !strings.Contains(got, "<cycle detected:") {
			t.Fatalf("expected cycle marker, got %q", got)
		}
	}
}

func TestSharedValuesAreNotCycles(t *testing.T) {
	x := 42
	if got := dump.SDump([]*int{&x, &x}); strings.Contains(got, "cycle detected") {
		t.Fatalf("shared pointer reported as cycle: %s", got)
	}
}

type redacted struct{ secret string }

func (r *redacted) Dump() string { return "redacted(" + r.secret[:1] + "…)" }

func TestNilDumperPointer(t *testing.T) {
	var r *redacted
	if got := dump.SDump(struct{ R *redacted }{r}); !strings.Contains(got, "R: nil") {
		t.Fatalf("want nil field, got %q", got)
	}
	if got := dump.SDump(&redacted{secret: "hunter2"}); got != "redacted(h…)\n" {
		t.Fatalf("got %q", got)
	}
}

func TestMapKeysSorted(t *testing.T) {
	got := dump.SDump(map[string]int{"b": 2, "a": 1, "c": 3})
	want := "map[string]int{\n  \"a\": 1,\n  \"b\": 2,\n  \"c\": 3,\n}\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
