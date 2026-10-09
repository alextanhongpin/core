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
