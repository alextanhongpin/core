package testutil

import (
	"bytes"
	"encoding/json"
	"os"
)

type LoadJSONOption interface {
	isLoadJSONOption()
}

type disallowUnknownFields struct{}

func (disallowUnknownFields) isLoadJSONOption() {}

func DisallowUnknownFields() *disallowUnknownFields { return nil }

func LoadJSON[T any](fileName string, opts ...LoadJSONOption) (*T, error) {
	var strict bool
	for _, opt := range opts {
		switch (opt).(type) {
		case *disallowUnknownFields:
			strict = true
		}
	}

	b, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	if strict {
		var t T
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&t); err != nil {
			return nil, err
		}

		return &t, nil
	}

	var t T
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}

	return &t, nil
}
