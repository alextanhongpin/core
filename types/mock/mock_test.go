package mock_test

import (
	"reflect"
	"testing"

	"github.com/alextanhongpin/core/types/mock"
)

// Fix Service methods and test usage for new Mock API

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
	if want, got := []int{10, 20, 30}, s.Calls().Values("Add"); reflect.DeepEqual(want, got) {
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
