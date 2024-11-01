package probs

import "strings"

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
