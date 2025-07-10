package mock

import (
	"fmt"
	"runtime"
	"slices"
	"strings"

	"github.com/alextanhongpin/core/types/structs"
)

// Options is a type-safe map for method options.
type Options map[string]string

// With returns a new Options with the given method/option pair added or replaced.
func (o Options) With(method, option string) Options {
	if o == nil {
		o = make(Options)
	}
	opts := make(Options, len(o))
	for k, v := range o {
		opts[k] = v
	}
	opts[method] = option
	return opts
}

// Mock provides method-based option lookup for test doubles and helpers.
type Mock struct {
	Options map[string]string
}

// New creates a new Mock for the exported methods of v, with the given options.
func New(v any, options Options) *Mock {
	methodList, err := structs.GetMethodNames(v)
	if err != nil {
		panic(err)
	}
	for method := range options {
		if !slices.Contains(methodList, method) {
			panic(fmt.Errorf("mock: unknown method %q, available methods: %v", method, methodList))
		}
	}
	return &Mock{Options: options}
}

// Option returns the option for the calling method, or an empty string if not set.
func (m *Mock) Option() string {
	return m.Options[m.getMethodName()]
}

func (m *Mock) getMethodName() string {
	name := CallerName(2) // Skip [getMethodName, Option]
	parts := strings.Split(name, ".")
	return parts[len(parts)-1]
}

// CallerName returns the name of the calling function.
func CallerName(skip int) string {
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
