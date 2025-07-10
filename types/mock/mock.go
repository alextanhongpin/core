package mock

import (
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strings"

	"github.com/alextanhongpin/core/types/structs"
)

type Options map[string]string

func (o Options) With(method, option string) Options {
	opts := maps.Clone(o)
	opts[method] = option

	return opts
}

type Mock struct {
	methods []string
	options map[string]string
}

func New(v any, options Options) *Mock {
	methods, err := structs.GetMethodNames(v)
	if err != nil {
		panic(err)
	}

	base := &Mock{
		methods: methods,
		options: make(map[string]string),
	}

	for method := range options {
		if !slices.Contains(base.methods, method) {
			panic(fmt.Errorf("mock: unknown method %q, available methods: %v", method, base.methods))
		}
		base.options[method] = options[method]
	}

	return base
}

func (m Mock) Option() string {
	return m.options[m.getMethodName()]
}

func (m Mock) getMethodName() string {
	name := CallerName(2) // Skip [getMethodName, Option]
	parts := strings.Split(name, ".")
	return parts[len(parts)-1]
}

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
