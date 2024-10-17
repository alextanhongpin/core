package assert

import (
	"reflect"
	"strings"
)

func Is(is bool, msg string) string {
	if is {
		return ""
	}

	return msg
}

func Required(v any, assertions ...string) string {
	var sb strings.Builder

	if isZero(v) {
		sb.WriteString("required")
		sb.WriteString(", ")
	}

	for _, s := range assertions {
		if s == "" {
			continue
		}
		sb.WriteString(s)
		sb.WriteString(", ")
	}

	return strings.TrimSuffix(sb.String(), ", ")
}

func Optional(v any, assertions ...string) string {
	if isZero(v) {
		return ""
	}

	return Required(v, assertions...)
}

func Map(kv map[string]string) map[string]string {
	res := make(map[string]string)
	for k, v := range kv {
		if v == "" {
			continue
		}
		res[k] = v
	}

	return res
}

func isZero(v any) bool {
	return v == nil || reflect.ValueOf(v).IsZero()
}
