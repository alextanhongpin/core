package probs

import (
	"errors"
	"strings"
)

func KeyAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}

	return strings.HasSuffix(err.Error(), "key already exists")
}

func KeyDoesNotExistError(err error) bool {
	if err == nil {
		return false
	}

	return strings.HasSuffix(err.Error(), "key does not exist")
}

var errNilClient = errors.New("probs: Redis client is required")
