package internal

import (
	"context"
	"time"

	"go-starter/internal/models"
)

// Repository is the single dependency a core service takes for state. It composes
// data access with locking so a service holds one collaborator, not three.
type Repository interface {
	DAO
	Lock
}

// Lock provides Redis-backed distributed locking.
type Lock interface {
	// DistributedLock runs execFunc while holding a cluster-wide lock on key.
	// expiry of 0 uses the default (15 minutes).
	// Returns constants.LockAlreadyHeld when the lock cannot be acquired.
	DistributedLock(ctx context.Context, key string, expiry time.Duration, execFunc func() error) error
}

// DAO composes every domain repository.
//
// Add one interface per domain and embed it here:
//
//	type DAO interface {
//	    Atomic(ctx context.Context, fn func(store DAO) error) error
//	    UserRepository
//	    OrderRepository
//	}
type DAO interface {
	/*
		Atomic runs fn inside one database transaction. The DAO handed to fn is
		scoped to that transaction, so every call through it participates:

			err := repo.Atomic(ctx, func(store internal.DAO) error {
				if err := store.CreateOrder(ctx, order); err != nil {
					return err
				}
				return store.DebitWallet(ctx, userID, order.Amount)
			})

		Returning an error rolls back. A panic inside fn also rolls back and is
		converted into an error.
	*/
	Atomic(ctx context.Context, fn func(store DAO) error) error
	ExpenseSharingRepository
}

// ExpenseSharingRepository is the persistence surface for the expense-sharing
// domain. It holds only the methods the core service actually calls — no
// speculative filter or list variants.
//
// Balances are derived on read, so there is no balance table and no balance
// method here: the core layer computes balances from the raw splits returned by
// ListAllSplits.
type ExpenseSharingRepository interface {
	// CreateUser inserts a user and returns it with the DB-assigned ID.
	// Returns constants.DuplicateEmail or constants.DuplicatePhone on a unique
	// constraint violation.
	CreateUser(ctx context.Context, user models.User) (models.User, error)

	// GetUserByID returns the user, or constants.RecordDoesntExists if absent.
	// Used to validate that a referenced user exists.
	GetUserByID(ctx context.Context, id int64) (models.User, error)

	// ListUsers returns every user. The core layer needs the full set to label
	// balances with names.
	ListUsers(ctx context.Context) ([]models.User, error)

	// CreateExpenseWithSplits inserts the expense and all of its splits in one
	// transaction, and returns the expense with the DB-assigned ID. The caller
	// has already validated the splits (sums, participants, payer).
	CreateExpenseWithSplits(ctx context.Context, expense models.Expense, splits []models.Split) (models.Expense, error)

	// ListAllSplits returns every split across all expenses. This is the raw
	// input for deriving both all-user balances and a single user's balances.
	ListAllSplits(ctx context.Context) ([]models.Split, error)

	// ListExpensesByUserID returns the expenses the user participates in, each
	// hydrated with its splits so the caller can show the user's owed/paid.
	ListExpensesByUserID(ctx context.Context, userID int64) ([]models.Expense, error)
}
