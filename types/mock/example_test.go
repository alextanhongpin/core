package mock_test

import (
	"fmt"

	"github.com/alextanhongpin/core/types/mock"
)

type ExampleService struct {
	*mock.Mock
}

func (s *ExampleService) WithOptions(options mock.Options) *ExampleService {
	s.Mock = mock.New(s, options)
	return s
}

func (s *ExampleService) Foo() string { return s.Call() }
func (s *ExampleService) Bar() string { return s.Call() }

func ExampleMock_Call() {
	s := new(ExampleService).WithOptions(mock.Options{"Foo": []string{"fast"}, "Bar": []string{"slow"}})
	fmt.Println(s.Foo())
	fmt.Println(s.Bar())
	// Output:
	// fast
	// slow
}
