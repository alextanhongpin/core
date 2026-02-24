package ratelimit

import "errors"

var ErrNegative = errors.New("ratelimit: requests cannot be negative")
