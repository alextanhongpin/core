package mock_test

import (
	"reflect"
	"testing"

	"github.com/alextanhongpin/core/types/mock"
)

type Service struct {
	*mock.Mock
}

func (s *Service) With(options mock.Options) *Service {
	s.Mock = mock.New(s, options)
	return s
}

func (s *Service) Foo() string      { return s.Call() }
func (s *Service) Bar() string      { return s.Call() }
func (s *Service) Add(n int) string { return s.Call(n) }

func TestMock_Option(t *testing.T) {
	s := new(Service).With(mock.Options{
		"Foo": []string{"fast"},
		"Bar": []string{"slow"},
		"Add": []string{"first", "second"},
	})
	if want, got := "fast", s.Foo(); want != got {
		t.Errorf("Foo(): want %q, got %q", want, got)
	}
	if want, got := "slow", s.Bar(); want != got {
		t.Errorf("Bar(): want %q, got %q", want, got)
	}
	if want, got := "first", s.Add(10); want != got {
		t.Errorf("Add(10): want %q, got %q", want, got)
	}
	if want, got := "second", s.Add(20); want != got {
		t.Errorf("Add(20): want %v, got %v", want, got)
	}
	if want, got := "first", s.Add(30); want != got {
		t.Errorf("Add(30): want %v, got %v", want, got)
	}
	if want, got := [][]any{{10}, {20}, {30}}, s.Calls().Values("Add"); !reflect.DeepEqual(want, got) {
		t.Fatalf("Values('Add'): want %v, got %v", want, got)
	}
}

func TestMock_PanicsOnUnknownMethod(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unknown method")
		}
	}()
	mock.New(Service{}, mock.Options{"Unknown": []string{"fail"}})
}

func TestMock_NoOptions(t *testing.T) {
	s := new(Service).With(nil)
	if got := s.Foo(); got != "" {
		t.Fatalf("Foo() = %q, want empty", got)
	}
	if got := len(s.Calls().Values("Foo")); got != 1 {
		t.Fatalf("recorded %d calls, want 1", got)
	}
}

func TestMap_WithDoesNotAlias(t *testing.T) {
	base := mock.Options{"Foo": make([]string, 1, 10)}
	a := base.With("Foo", "a")
	b := base.With("Foo", "b")
	if a["Foo"][1] != "a" || b["Foo"][1] != "b" || len(base["Foo"]) != 1 {
		t.Fatalf("With aliased the original: a=%v b=%v base=%v", a, b, base)
	}
}

func TestMock_CallsIsCopy(t *testing.T) {
	s := new(Service).With(nil)
	s.Add(1)
	calls := s.Calls()
	calls["Add"][0] = []any{99}
	if got := s.Calls().Values("Add")[0][0]; got != 1 {
		t.Fatalf("Calls exposed internal state: %v", got)
	}
}
