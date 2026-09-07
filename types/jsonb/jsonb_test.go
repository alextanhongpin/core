package jsonb_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/jsonb"
	"github.com/alextanhongpin/evaltest"
)

func TestSet(t *testing.T) {
	type Input struct {
		Data  any
		Key   string
		Value any
	}
	evaltest.Run(t, func(t *evaltest.T, input Input) (any, error) {
		// Given an input,
		// When setting a key to value,
		return input.Data, jsonb.Set(input.Data, input.Key, input.Value)
	})
}

func TestDelete(t *testing.T) {
	type Input struct {
		Data any
		Key  string
	}
	evaltest.Run(t, func(t *evaltest.T, input Input) (any, error) {
		// Given an input,
		// When setting a key to value,
		return input.Data, jsonb.Delete(input.Data, input.Key)
	})
}
