package maputil

const MaskValue = "*!REDACTED*"

func MaskFunc(m map[string]any, fn func(k string) bool) map[string]any {
	return replaceFunc(m, func(k, v string) string {
		if fn(k) {
			return MaskValue
		}

		return v
	})
}

func MaskFields(fields ...string) func(k string) bool {
	return func(k string) bool {
		for _, f := range fields {
			if f == k {
				return true
			}
		}

		return false
	}
}
