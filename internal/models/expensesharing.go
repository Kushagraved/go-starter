package models

import "time"

// SplitType is how an expense's total is divided among its participants.
type SplitType string

const (
	SplitEqual   SplitType = "EQUAL"
	SplitExact   SplitType = "EXACT"
	SplitPercent SplitType = "PERCENT"
)

// User is a person who can pay for and participate in expenses.
//
// ID is a Postgres bigserial assigned by the database on insert (zero until
// then). Email and Phone are each individually unique.
type User struct {
	ID        int64
	Name      string
	Email     string
	Phone     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Expense is a single recorded cost split across one or more users.
//
// There is no PaidByUserID: the payer is expressed inside the splits (the split
// row whose PaidPaisa > 0), which keeps a single source of truth and lets the
// model extend to multiple payers without a schema change.
//
// AmountPaisa is the total in paisa (integer). Across an expense's splits both
// sum(OwedPaisa) and sum(PaidPaisa) equal AmountPaisa; the core layer enforces
// this before the write.
//
// Splits is hydrated on read; it is empty on a freshly built entity.
type Expense struct {
	ID              int64
	CreatedByUserID int64
	Name            string
	AmountPaisa     int64
	SplitType       SplitType
	CreatedAt       time.Time
	Splits          []Split
}

// Split is one user's stake in one expense, in paisa.
//
//	OwedPaisa — what this user is responsible for.
//	PaidPaisa — what this user actually fronted.
//
// A user's net on an expense is PaidPaisa - OwedPaisa: positive means the group
// owes them, negative means they owe the group.
//
// ID is a Postgres bigserial assigned on insert. (ExpenseID, UserID) is still
// unique, so a user cannot appear twice in one expense.
type Split struct {
	ID        int64
	ExpenseID int64
	UserID    int64
	OwedPaisa int64
	PaidPaisa int64
}
