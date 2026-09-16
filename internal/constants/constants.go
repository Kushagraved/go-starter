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
)
