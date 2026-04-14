package mock

import (
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strings"

	"github.com/alextanhongpin/core/types/structs"
)

// Mock provides method-based option lookup for test doubles and helpers.
type Mock struct {
	options Options
	calls   Calls
}

// New creates a new Mock for the exported methods of v, with the given options.
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
		options: options,
		calls:   make(Calls),
	}
}

func (m *Mock) Calls() Calls {
	return maps.Clone(m.calls)
}

// Call stores the caller args.
func (m *Mock) Call(args ...any) string {
	name := m.getMethodName()
	values := m.options.Values(name)
	call := len(m.calls[name])
	val := values[call%len(values)]
	m.calls[name] = append(m.calls[name], args)
	return val
}

func (m *Mock) getMethodName() string {
	name := callerName(2) // Skip [getMethodName, Option]
	parts := strings.Split(name, ".")
	return parts[len(parts)-1]
}

// callerName returns the name of the calling function.
func callerName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip + 1)
	if !ok {
		return ""
	}
	f := runtime.FuncForPC(pc)
	if f == nil {
		return ""
	}
	return f.Name()
}
