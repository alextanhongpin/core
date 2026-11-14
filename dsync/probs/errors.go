package probs

import (
	"errors"
	"strings"
)

// KeyAlreadyExistsError reports whether err is the Redis module error
// returned when creating a CMS, Top-K or t-digest that already exists
// ("... key already exists"). The Init and Create methods already treat
// this case as success. Bloom and Cuckoo Reserve return a different error
// ("item exists") that this function does not match.
func KeyAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}

	return strings.HasSuffix(err.Error(), "key already exists")
}

// KeyDoesNotExistError reports whether err is the Redis module error
// returned when reading or updating a CMS, Top-K or t-digest key that does
// not exist ("... key does not exist"), for example from
// CountMinSketch.Query.
func KeyDoesNotExistError(err error) bool {
	if err == nil {
		return false
	}

	return strings.HasSuffix(err.Error(), "key does not exist")
}

var errNilClient = errors.New("probs: Redis client is required")
