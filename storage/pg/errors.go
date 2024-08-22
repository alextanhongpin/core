/*
	Postgres error handling.
	Reference: https://www.postgresql.org/docs/9.2/errcodes-appendix.html

	We want to handle the following codes

	23000	integrity_constraint_violation
	23001	restrict_violation
	23502	not_null_violation
	23503	foreign_key_violation
	23505	unique_violation
	23514	check_violation
	23P01	exclusion_violation
*/

package pg

import (
	"errors"

	"github.com/lib/pq"
)

const (
	IntegrityConstraintViolation = "23000"
	RestrictVioloation           = "23001"
	NotNullViolation             = "23502"
	ForeignKeyViolation          = "23503"
	UniqueViolation              = "23505"
	CheckViolation               = "23514"
	ExclusionViolation           = "23P01"
	TriggerException             = "P0000"
)

func As(err error) (*pq.Error, bool) {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr, true
	}

	return nil, false
}

func IsCode(err error, code string) bool {
	pqErr, ok := As(err)
	return ok && pqErr.Code == pq.ErrorCode(code)
}

func IsUniqueViolation(err error) bool {
	return IsCode(err, UniqueViolation)
}

// IsForeignKeyViolation can be used to test if the row to hard delete contains
// foreign keys references.
// If this error is detected, then just soft delete.
func IsForeignKeyViolation(err error) bool {
	return IsCode(err, ForeignKeyViolation)
}
