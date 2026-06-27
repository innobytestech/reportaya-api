package pgutil

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const uniqueViolationCode = "23505"

func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == uniqueViolationCode
	}

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "duplicate key") || strings.Contains(lower, uniqueViolationCode)
}

// UniqueConstraintName returns the constraint/index name that caused a unique
// violation, or "" if the error is not a unique violation or the name cannot
// be determined. Useful to discriminate which column collided when several
// unique constraints exist on the same table.
func UniqueConstraintName(err error) string {
	if err == nil {
		return ""
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
		return pgErr.ConstraintName
	}
	// Fallback for wrapped errors that only expose the message.
	msg := err.Error()
	const marker = `constraint "`
	if i := strings.Index(msg, marker); i >= 0 {
		rest := msg[i+len(marker):]
		if j := strings.Index(rest, `"`); j > 0 {
			return rest[:j]
		}
	}
	return ""
}
