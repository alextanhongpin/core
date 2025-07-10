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

func (s *ExampleService) Foo() string { return s.Option() }
func (s *ExampleService) Bar() string { return s.Option() }

func ExampleMock_Option() {
	s := new(ExampleService).WithOptions(mock.Options{"Foo": "fast", "Bar": "slow"})
	fmt.Println(s.Foo())
	fmt.Println(s.Bar())
	// Output:
	// fast
	// slow
}
