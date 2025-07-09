package mock_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/mock"
)

// Fix Service methods and test usage for new Mock API

type Service struct {
	*mock.Mock
}

func (s *Service) WithOptions(options ...string) *Service {
	s.Mock = mock.New(s, options...)
	return s
}

func (s *Service) Foo() string { return s.Option() }
func (s *Service) Bar() string { return s.Option() }

func TestMock_Option(t *testing.T) {
	s := new(Service).WithOptions("Foo=fast", "Bar=slow")
	if got := s.Foo(); got != "fast" {
		t.Errorf("Option() in Foo: got %q, want %q", got, "fast")
	}
	if got := s.Bar(); got != "slow" {
		t.Errorf("Option() in Bar: got %q, want %q", got, "slow")
	}
}

func TestMock_PanicsOnUnknownMethod(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unknown method")
		}
	}()
	mock.New(Service{}, "Unknown=fail")
}

func TestMock_PanicsOnInvalidFormat(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid option format")
		}
	}()
	mock.New(Service{}, "Foo")
}
