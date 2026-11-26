package repository

import (
	"strings"

	"go-starter/internal/constants"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
)

// Postgres SQLSTATE codes we care about.
const (
	pgUniqueViolation    = "23505" // unique_violation
	pgExclusionViolation = "23P01" // exclusion_violation
)

// Named constraints from scripts/db/schema.sql. Matching on the constraint name
// lets us map a single violation to a specific domain error instead of guessing.
const (
	constraintBookingIdempotency = "uq_bookings_idempotency_key"
	constraintBookingNoOverlap   = "ex_bookings_no_overlap"
)

// asPgError unwraps err to a *pgconn.PgError when the driver produced one.
func asPgError(err error) (*pgconn.PgError, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr, true
	}
	return nil, false
}

// translateBookingWriteErr maps a booking insert failure onto a domain sentinel:
// a duplicate idempotency key, or an overlapping window caught by the exclusion
// constraint. Anything else is returned unchanged.
func translateBookingWriteErr(err error) error {
	pgErr, ok := asPgError(err)
	if !ok {
		return err
	}

	switch pgErr.Code {
	case pgUniqueViolation:
		if strings.Contains(pgErr.ConstraintName, constraintBookingIdempotency) {
			return constants.DuplicateIdempotencyKey
		}
	case pgExclusionViolation:
		if strings.Contains(pgErr.ConstraintName, constraintBookingNoOverlap) {
			return constants.BookingOverlap
		}
	}
	return err
}
