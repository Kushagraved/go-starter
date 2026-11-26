package constants

import "github.com/pkg/errors"

// Error codes returned in the API error envelope.
const (
	ErrInternalServer = "internal_server_error"
	ErrBadRequest     = "bad_request"
	ErrUnauthorized   = "unauthorized"
	ErrNotFound       = "not_found"
	ErrConflict       = "conflict"
)

// ContextKey is the type for every value stashed on a context.Context.
type ContextKey string

const (
	// TraceID correlates all logs for one request or one queue message.
	TraceID ContextKey = "trace_id"
	// RequestID is the inbound HTTP request ID.
	RequestID ContextKey = "request_id"
)

// Sentinel errors shared across layers. Compare with errors.Is, never by string.
var (
	// RecordDoesntExists means a lookup found nothing.
	RecordDoesntExists = errors.New("record does not exist")
	// LockAlreadyHeld means a distributed lock could not be acquired.
	LockAlreadyHeld = errors.New("lock is already held")
	// NoCarAvailable means no car satisfies the booking request for the window.
	NoCarAvailable = errors.New("no car available for the requested window")
	// BookingOverlap means the chosen car already has an overlapping booking.
	// It surfaces when a concurrent request grabbed the same car first.
	BookingOverlap = errors.New("car already booked for an overlapping window")
	// DuplicateIdempotencyKey means a booking already exists for the given
	// client-supplied idempotency key.
	DuplicateIdempotencyKey = errors.New("idempotency key already used")
	// PriceConfigNotFound means the (branch, category) pair has no pricing.
	PriceConfigNotFound = errors.New("price configuration not found")
	// ErrValidation marks a request that failed business validation. Controllers
	// map it to 400 Bad Request.
	ErrValidation = errors.New("validation failed")
)
