package mock_test

import (
	"fmt"

	"github.com/alextanhongpin/core/types/mock"
)

type ExampleService struct{}

func (ExampleService) Foo(m *mock.Mock) string { return m.Option() }
func (ExampleService) Bar(m *mock.Mock) string { return m.Option() }

func ExampleMock_Option() {
	m := mock.New(ExampleService{}, mock.Options{"Foo": "fast", "Bar": "slow"})
	s := ExampleService{}
	fmt.Println(s.Foo(m))
	fmt.Println(s.Bar(m))
	// Output:
	// fast
	// slow
}
