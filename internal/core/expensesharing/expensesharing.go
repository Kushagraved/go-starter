// Package expensesharing holds the business logic for the expense-sharing
// domain: creating users and expenses, validating splits, and deriving balances.
// It speaks models.* entities and depends only on the repository interface and
// the logger — never on gin, gorm, or any transport concern.
package expensesharing

import (
	"context"

	"go-starter/internal"
	"go-starter/internal/logger"
	"go-starter/internal/models"
)

// Service is the expense-sharing business API. Delivery layers (the CLI driver,
// or an HTTP controller) call these methods.
type Service interface {
	// AddUser validates the request and creates a user. Returns a validation
	// sentinel on bad input, or constants.DuplicateEmail / DuplicatePhone when
	// the email or phone is already taken.
	AddUser(ctx context.Context, req AddUserRequest) (models.User, error)

	// AddExpense validates the request (users exist, no duplicate participants,
	// payer is a participant, shares are consistent with the split type) and
	// records the expense with one split per participant. Returns a validation
	// sentinel on bad input.
	AddExpense(ctx context.Context, req AddExpenseRequest) (models.Expense, error)

	// GetAllBalances derives every non-zero pairwise balance across all
	// expenses. Each returned Balance reads "FromUser owes ToUser AmountPaisa".
	GetAllBalances(ctx context.Context) ([]Balance, error)

	// GetUserData returns the non-zero balances that involve the given user —
	// where they are either the debtor or the creditor. Returns
	// constants.RecordDoesntExists if the user does not exist.
	GetUserData(ctx context.Context, userID int64) ([]Balance, error)

	// GetUserExpenses returns the expenses the given user participates in, each
	// annotated with that user's owed and paid amounts. Returns
	// constants.RecordDoesntExists if the user does not exist.
	GetUserExpenses(ctx context.Context, userID int64) ([]UserExpense, error)
}

// UserExpense is one expense from a single user's point of view: the expense
// itself plus how much that user owed and paid on it.
type UserExpense struct {
	Expense       models.Expense
	UserOwedPaisa int64
	UserPaidPaisa int64
}

// Balance is a settled debt between two users: FromUserID owes ToUserID
// AmountPaisa. AmountPaisa is always positive — the direction lives in the
// From/To fields, never in the sign.
type Balance struct {
	FromUserID   int64
	FromUserName string
	ToUserID     int64
	ToUserName   string
	AmountPaisa  int64
}

type service struct {
	repo   internal.Repository
	logger logger.Logger
}

// NewService wires the core service over the repository.
func NewService(repo internal.Repository, loggerObj logger.Logger) Service {
	return &service{repo: repo, logger: loggerObj}
}
