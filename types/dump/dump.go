package dump

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// Dumper is the interface checked during reflection.
type Dumper interface {
	Dump() string
}

func Dump(v any) []byte {
	b := new(bytes.Buffer)
	FPrettyPrint(b, v)
	return b.Bytes()
}

func SDump(v any) string {
	b := new(bytes.Buffer)
	FPrettyPrint(b, v)
	return b.String()
}

// FPrettyPrint prints values recursively to an io.Writer with indentation and cycle detection.
func FPrettyPrint(w io.Writer, v any) {
	visited := make(map[visit]bool)
	printRecursive(w, reflect.ValueOf(v), 0, visited)
	fmt.Fprintln(w)
}

// PrettyPrint remains as a wrapper that directly outputs to standard output.
func PrettyPrint(v any) {
	FPrettyPrint(os.Stdout, v)
}

type visit struct {
	typ    reflect.Type
	ptr    uintptr
	length int
}

func printRecursive(w io.Writer, v reflect.Value, indent int, visited map[visit]bool) {
	pad := strings.Repeat("  ", indent)

	if !v.IsValid() {
		fmt.Fprint(w, "nil")
		return
	}

	// Check if the value implements the Dumper interface
	if v.CanInterface() {
		if dumper, ok := v.Interface().(Dumper); ok {
			fmt.Fprint(w, dumper.Dump())
			return
		}
	}
	if v.CanAddr() && v.Addr().CanInterface() {
		if dumper, ok := v.Addr().Interface().(Dumper); ok {
			fmt.Fprint(w, dumper.Dump())
			return
		}
	}

	// Handle pointer types and detect cycles
	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			fmt.Fprint(w, "nil")
			return
		}
		if v.Kind() == reflect.Pointer {
			ptr := visit{typ: v.Type(), ptr: v.Pointer()}
			if visited[ptr] {
				fmt.Fprintf(w, "<cycle detected: %s>", v.Type().String())
				return
			}
			visited[ptr] = true
			defer delete(visited, ptr)

			fmt.Fprint(w, "&")
			printRecursive(w, v.Elem(), indent, visited)
			return
		}
		printRecursive(w, v.Elem(), indent, visited)
		return
	}

	// Maps and slices can contain themselves through interface values.
	if (v.Kind() == reflect.Map || v.Kind() == reflect.Slice) && !v.IsNil() {
		key := visit{typ: v.Type(), ptr: v.Pointer()}
		if v.Kind() == reflect.Slice {
			key.length = v.Len()
		}
		if visited[key] {
			fmt.Fprintf(w, "<cycle detected: %s>", v.Type())
			return
		}
		visited[key] = true
		defer delete(visited, key)
	}

	switch v.Kind() {
	case reflect.Struct:
		fmt.Fprintf(w, "%s{", removeCompileGeneratedLink(v.Type().String()))
		var ok bool
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			fieldVal := v.Field(i)

			if !fieldVal.CanInterface() {
				continue
			}
			if !ok {
				fmt.Fprint(w, "\n")
				ok = true
			}

			fmt.Fprintf(w, "%s  %s: ", pad, field.Name)
			printRecursive(w, fieldVal, indent+1, visited)
			fmt.Fprintln(w, ",")
		}
		if !ok {
			fmt.Fprint(w, "}")
		} else {
			fmt.Fprintf(w, "%s}", pad)
		}

	case reflect.Slice, reflect.Array:
		if v.Len() == 0 {
			fmt.Fprintf(w, "%s{}", v.Type().String())
			return
		}
		fmt.Fprintf(w, "%s{\n", v.Type().String())
		for i := 0; i < v.Len(); i++ {
			fmt.Fprintf(w, "%s  ", pad)
			printRecursive(w, v.Index(i), indent+1, visited)
			fmt.Fprintln(w, ",")
		}
		fmt.Fprintf(w, "%s}", pad)

	case reflect.Map:
		if len(v.MapKeys()) == 0 {
			fmt.Fprintf(w, "%s{}", v.Type().String())
			return
		}
		fmt.Fprintf(w, "%s{\n", v.Type().String())

		keys := v.MapKeys()
		// 2. Determine if the key type is ordered and sort accordingly
		keyKind := keys[0].Kind()
		switch {
		case keyKind == reflect.String:
			sort.Slice(keys, func(i, j int) bool {
				return keys[i].String() < keys[j].String()
			})
		case keyKind >= reflect.Int && keyKind <= reflect.Int64:
			sort.Slice(keys, func(i, j int) bool {
				return keys[i].Int() < keys[j].Int()
			})
		case keyKind >= reflect.Uint && keyKind <= reflect.Uintptr:
			sort.Slice(keys, func(i, j int) bool {
				return keys[i].Uint() < keys[j].Uint()
			})
		case keyKind == reflect.Float32 || keyKind == reflect.Float64:
			sort.Slice(keys, func(i, j int) bool {
				return keys[i].Float() < keys[j].Float()
			})
		default:
			// The key type doesn't have a natural ordering (e.g., structs, pointers, interfaces)
			// Leave the keys slice in its random, unspecified reflection order.
		}
		for _, key := range keys {
			fmt.Fprintf(w, "%s  %#v: ", pad, key.Interface())
			printRecursive(w, v.MapIndex(key), indent+1, visited)
			fmt.Fprintln(w, ",")
		}
		fmt.Fprintf(w, "%s}", pad)

	default:
		fmt.Fprintf(w, "%#v", v.Interface())
	}
}

var re = regexp.MustCompile(`·\d+`)

func removeCompileGeneratedLink(s string) string {
	return re.ReplaceAllString(s, "")
}
