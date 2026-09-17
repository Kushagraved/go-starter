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
	// DuplicateEmail means a user with the same email already exists.
	DuplicateEmail = errors.New("user with this email already exists")
	// DuplicatePhone means a user with the same phone already exists.
	DuplicatePhone = errors.New("user with this phone already exists")

	// ── Validation sentinels (bad input → HTTP 400 / CLI error) ──────────────
	// ErrEmptyName means the user's name was blank.
	ErrEmptyName = errors.New("name is required")
	// ErrEmptyEmail means the user's email was blank.
	ErrEmptyEmail = errors.New("email is required")
	// ErrEmptyPhone means the user's phone was blank.
	ErrEmptyPhone = errors.New("phone is required")
	// ErrInvalidEmail means the email is not a well-formed address.
	ErrInvalidEmail = errors.New("email is not valid")
	// ErrInvalidPhone means the phone is not a valid 10-digit number.
	ErrInvalidPhone = errors.New("phone is not valid")

	// ── Expense validation sentinels ─────────────────────────────────────────
	// ErrEmptyExpenseName means the expense name was blank.
	ErrEmptyExpenseName = errors.New("expense name is required")
	// ErrNonPositiveAmount means the total amount was zero or negative.
	ErrNonPositiveAmount = errors.New("total amount must be positive")
	// ErrEmptyUserList means no participant users were provided.
	ErrEmptyUserList = errors.New("at least one participant is required")
	// ErrInvalidSplitType means the split type was not EQUAL, EXACT, or PERCENT.
	ErrInvalidSplitType = errors.New("split type must be EQUAL, EXACT or PERCENT")
	// ErrDuplicateUserInSplit means the same user appeared twice in the list.
	ErrDuplicateUserInSplit = errors.New("a user appears more than once in the split")
	// ErrPayerNotInSplit means the payer is not among the participants.
	ErrPayerNotInSplit = errors.New("the payer must be one of the participants")
	// ErrShareCountMismatch means the number of shares (amounts/percents) does
	// not match the number of participants.
	ErrShareCountMismatch = errors.New("number of shares does not match number of participants")
	// ErrExactSumMismatch means the exact shares do not add up to the total.
	ErrExactSumMismatch = errors.New("sum of exact shares must equal the total amount")
	// ErrPercentSumMismatch means the percent shares do not add up to 100.
	ErrPercentSumMismatch = errors.New("sum of percent shares must equal 100")
	// ErrNegativeShare means a share (amount or percent) was negative.
	ErrNegativeShare = errors.New("a share cannot be negative")
)
