package db

import (
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	UniqueViolation     = "23505"
	ForeignKeyViolation = "23503"
)

func ErrCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}

	// fallback for testing purposes
	var testErr interface {
		Error() string
		Code() string
	}

	if errors.As(err, &testErr) {
		return testErr.Code()
	}

	return ""
}

func IsErrUniqueViolation(err error) bool {
	if errCode := ErrCode(err); errCode == UniqueViolation {
		return true
	}
	return false
}

func IsErrNoRows(err error) bool {
	if errors.Is(err, pgx.ErrNoRows) {
		return true
	}
	return false
}

func IsErrForeignKeyViolation(err error) bool {
	if errCode := ErrCode(err); errCode == ForeignKeyViolation {
		return true
	}
	return false
}
