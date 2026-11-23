package env_test

import (
	"errors"
	"testing"

	"github.com/alextanhongpin/core/types/env"
)

func TestParseRejectsTrailingInput(t *testing.T) {
	if _, err := env.Parse[int]("12abc"); !errors.Is(err, env.ErrParseFailed) {
		t.Fatalf("want ErrParseFailed, got %v", err)
	}
	if _, err := env.Parse[uint8]("300"); !errors.Is(err, env.ErrParseFailed) {
		t.Fatalf("want overflow error, got %v", err)
	}
}

func TestParseStringKeepsWholeValue(t *testing.T) {
	for _, s := range []string{"hello world", ""} {
		got, err := env.Parse[string](s)
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %q, %v", s, got, err)
		}
	}
}

func TestParseNamedAndPrefixedTypes(t *testing.T) {
	type Port int
	if got, err := env.Parse[Port]("0x1F"); err != nil || got != 31 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestLoadEmptyString(t *testing.T) {
	t.Setenv("EMPTY", "")
	if got, err := env.Load[string]("EMPTY"); err != nil || got != "" {
		t.Fatalf("got %q, %v", got, err)
	}
	if got, err := env.LoadSlice[int]("EMPTY", ","); err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestLoadNotSet(t *testing.T) {
	if _, err := env.Load[int]("ENV_TEST_NOT_SET"); !errors.Is(err, env.ErrNotSet) {
		t.Fatalf("want ErrNotSet, got %v", err)
	}
}
