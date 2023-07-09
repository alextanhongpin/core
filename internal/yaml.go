package internal

import (
	"encoding/json"

	orderedmap "github.com/wk8/go-ordered-map/v2"
	"gopkg.in/yaml.v3"
)

func MarshalYAML(v any) ([]byte, error) {
	b, ok := v.([]byte)
	if ok {
		return b, nil
	}

	return yaml.Marshal(v)
}

func UnmarshalYAML[T any](b []byte) (T, error) {
	var t T
	err := yaml.Unmarshal(b, &t)
	return t, err
}

func MarshalYAMLPreserveKeysOrder[T any](v T) ([]byte, error) {
	switch t := any(v).(type) {
	case map[string]any:
		return yaml.Marshal(t)
	case []byte:
		return t, nil
	default:
		if !IsStruct(v) {
			return yaml.Marshal(t)
		}
		// The problem with yaml.Marshal is, if the keys are
		// not set in the struct tags, then the name will be
		// lowercase.
		// E.g.
		// MyBirthday => mybirthday
		// We can marshal to JSON first to get the pascal case
		// name, but when unmarshalled, it will just lose the
		// order.
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}

		// To preserve the order, we use the ordered map.
		om := orderedmap.New[string, any]()
		if err := json.Unmarshal(b, &om); err != nil {
			return nil, err
		}

		return yaml.Marshal(om)
	}
}
