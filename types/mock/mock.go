package mock

import (
	"runtime"
	"slices"
	"strings"

	"github.com/alextanhongpin/core/types/structs"
)

type Mock struct {
	methods []string
	options map[string]string
}

func New(v any, options ...string) *Mock {
	methods, err := structs.GetMethodNames(v)
	if err != nil {
		panic(err)
	}

	base := &Mock{
		methods: methods,
		options: make(map[string]string),
	}

	for _, opt := range options {
		method, option, ok := strings.Cut(opt, "=")
		if !ok {
			panic("mock: option must be in the format method=option")
		}
		if !slices.Contains(base.methods, method) {
			panic("mock: unknown method " + method)
		}
		base.options[method] = option
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
