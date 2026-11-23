package mock

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/alextanhongpin/core/types/structs"
)

// Mock returns canned string options per method and records call arguments.
// Embed *Mock in a test double and call Call from each exported method.
// It is safe for concurrent use.
type Mock struct {
	mu      sync.Mutex
	options Options
	calls   Calls
}

// New returns a Mock for the exported methods of v (a struct or pointer to
// one). It panics if options names a method that v does not have.
func New(v any, options Options) *Mock {
	methodNames, err := structs.GetMethodNames(v)
	if err != nil {
		panic(err)
	}
	for method := range options {
		if !slices.Contains(methodNames, method) {
			panic(fmt.Errorf("mock: unknown method %q, available methods: %v", method, methodNames))
		}
	}
	return &Mock{
		options: options.clone(),
		calls:   make(Calls),
	}
}

// Calls returns a copy of the recorded arguments, keyed by method name.
func (m *Mock) Calls() Calls {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls.clone()
}

// Call records args under the name of the method that called it and returns
// that method's next option. Options cycle: the nth call returns
// options[n % len(options)]. It returns "" when the method has no options.
//
// Call must be invoked directly from the exported method; calling it from a
// closure or helper records the wrong name.
func (m *Mock) Call(args ...any) string {
	name := callerMethodName(2)

	m.mu.Lock()
	defer m.mu.Unlock()

	n := len(m.calls[name])
	m.calls[name] = append(m.calls[name], args)

	values := m.options[name]
	if len(values) == 0 {
		return ""
	}
	return values[n%len(values)]
}

// callerMethodName returns the unqualified name of the function skip frames
// above its caller, e.g. "Foo" for "pkg.(*Service).Foo".
func callerMethodName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	f := runtime.FuncForPC(pc)
	if f == nil {
		return ""
	}
	name := f.Name()
	return name[strings.LastIndexByte(name, '.')+1:]
}
