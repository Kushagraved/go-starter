package internal

import (
	"context"
	"time"
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
}
