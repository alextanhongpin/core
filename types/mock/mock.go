package mock

import (
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/alextanhongpin/core/types/structs"
)

// Mock provides method-based option lookup for test doubles and helpers.
type Mock struct {
	mu      sync.Mutex // Protects access to internal state (calls, options)
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

// Calls returns a read-only copy of all recorded calls.
func (m *Mock) Calls() Calls {
	m.mu.Lock()
	defer m.mu.Unlock()
	return maps.Clone(m.calls)
}

// Call stores the caller args. This function is inherently risky due to reflection usage,
// but we keep it as per the original design intent while adding thread safety.
func (m *Mock) Call(args ...any) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := m.getMethodName()
	values := m.options.Values(name)
	callCount := len(m.calls[name]) // Use the current count before appending

	// Cycle through defined options if more calls are made than options exist
	val := values[callCount%len(values)]

	m.calls[name] = append(m.calls[name], args)
	return val
}

func (m *Mock) getMethodName() string {
	// WARNING: This remains fragile due to runtime reflection usage.
	// In a real-world scenario, this must be replaced by type-safe dependency injection.
	name := callerName(2) // Skip [getMethodName, Option]
	parts := strings.Split(name, ".")
	return parts[len(parts)-1]
}

// callerName returns the name of the calling function. Requires 3rd level runtime inspection.
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
