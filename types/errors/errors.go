package errors

import (
	_ "embed"

	"github.com/BurntSushi/toml"
	"github.com/alextanhongpin/errors"
)

// Alias to avoid referencing the original package.
type (
	Error = errors.Error
	Kind  = errors.Kind
)

const (
	AlreadyExists errors.Kind = "already_exists"
	BadInput      errors.Kind = "bad_input"
	Conflict      errors.Kind = "conflict"
	Forbidden     errors.Kind = "forbidden"
	Internal      errors.Kind = "internal"
	NotFound      errors.Kind = "not_found"
	Unauthorized  errors.Kind = "unauthorized"
	Unknown       errors.Kind = "unknown"
	Unprocessable errors.Kind = "unprocessable"
)

var (
	//go:embed errors.toml
	errorBytes []byte
	_          = errors.MustAddKinds(AlreadyExists,
		BadInput,
		Conflict,
		Forbidden,
		Internal,
		NotFound,
		Unauthorized,
		Unknown,
		Unprocessable,
	)
	_           = errors.MustLoad(errorBytes, toml.Unmarshal)
	ErrInternal = errors.Get("errors.internal")

	// Export out functionality
	MustLoad = errors.MustLoad
	Get      = errors.Get
)

func ToPartial[T any](err *errors.Error) *errors.PartialError[T] {
	return errors.ToPartial[T](err)
}
