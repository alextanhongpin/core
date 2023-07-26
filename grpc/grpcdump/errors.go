package grpcdump

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrMetadataNotFound      = errors.New("grpcdump: no metadata found")
	ErrUnknownMessageOrigin  = errors.New("grpcdump: unknown message origin")
	ErrInvalidLine           = errors.New("grpcdump: invalid line")
	ErrInvalidDump           = errors.New("grpcdump: invalid dump")
	ErrInvalidMetadata       = errors.New("grpcdump: invalid metadata")
	ErrInvalidMethod         = errors.New("grpcdump: invalid method")
	ErrInvalidOrigin         = errors.New("grpcdump: invalid origin")
	ErrBadClientStreamPrefix = errors.New("grpcdump: bad client stream prefix")
	ErrBadServerStreamPrefix = errors.New("grpcdump: bad server stream prefix")
	ErrMissingGRPCTestID     = errors.New("grpcdump: missing grpcdump test id")
)

func InvalidLineError(line string) error {
	if !strings.HasPrefix(line, linePrefix) {
		return fmt.Errorf("%w: %q", ErrInvalidLine, line)
	}

	return nil
}

func InvalidMethodError(line string) error {
	return fmt.Errorf("%w: %s", ErrInvalidMethod, line)
}

func InvalidOriginError(text string) error {
	return fmt.Errorf("%w: %s", ErrInvalidOrigin, text)
}

func BadClientStreamPrefixError(origin string) error {
	return fmt.Errorf("%w: %s", ErrBadClientStreamPrefix, origin)
}

func BadServerStreamPrefixError(origin string) error {
	return fmt.Errorf("%w: %s", ErrBadServerStreamPrefix, origin)
}

func InvalidMetadataError(text string) error {
	return fmt.Errorf("%w: %s", ErrInvalidMetadata, text)
}

func UnknownMessageOriginError(origin string) error {
	return fmt.Errorf("%w: %q", ErrUnknownMessageOrigin, origin)
}
