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

	BranchRepository
	CarRepository
	PriceConfigurationRepository
	UserRepository
	BookingRepository
}

// BranchRepository persists branches and reads them back.
type BranchRepository interface {
	// CreateBranch inserts a branch and returns it with the DB-assigned ID.
	CreateBranch(ctx context.Context, branch models.Branch) (models.Branch, error)
	// GetBranchByID returns constants.RecordDoesntExists when absent.
	GetBranchByID(ctx context.Context, id int64) (models.Branch, error)
}

// CarRepository persists cars and reads them back.
type CarRepository interface {
	// CreateCar inserts a car into a branch's fleet and returns it with the
	// DB-assigned ID.
	CreateCar(ctx context.Context, car models.Car) (models.Car, error)
	// GetCarByID returns constants.RecordDoesntExists when absent.
	GetCarByID(ctx context.Context, id int64) (models.Car, error)
}

// PriceConfigurationRepository persists per-(branch, category) pricing.
type PriceConfigurationRepository interface {
	// UpsertPriceConfiguration sets the price for a (branch, category) pair,
	// appending to the audit trail when an existing amount changes.
	UpsertPriceConfiguration(ctx context.Context, cfg models.PriceConfiguration) (models.PriceConfiguration, error)
	// GetPriceConfiguration returns the pricing for a (branch, category) pair, or
	// constants.PriceConfigNotFound when none is configured.
	GetPriceConfiguration(ctx context.Context, branchID int64, category models.CarCategory) (models.PriceConfiguration, error)
}

// UserRepository persists users and reads them back.
type UserRepository interface {
	// CreateUser inserts a user and returns it with the DB-assigned ID.
	CreateUser(ctx context.Context, user models.User) (models.User, error)
	// GetUserByID returns constants.RecordDoesntExists when absent.
	GetUserByID(ctx context.Context, id int64) (models.User, error)
}

// BookingRepository is the heart of the system: candidate selection under
// concurrency, plus idempotent creation and paginated reads.
type BookingRepository interface {
	// FindAvailableCars returns cars of the requested category that have no
	// booking overlapping [startTime, endTime), joined with their branch pricing
	// and ordered by amount ascending (cheapest first).
	//
	// It must be called inside Atomic: the implementation locks the candidate
	// rows with SELECT ... FOR UPDATE SKIP LOCKED so concurrent bookers never
	// contend on the same car and each sees a different cheapest-available car.
	FindAvailableCars(ctx context.Context, category models.CarCategory, startTime, endTime time.Time) ([]models.CandidateCar, error)

	// CreateBooking inserts a confirmed booking. Returns
	// constants.DuplicateIdempotencyKey if the idempotency key was already used,
	// and constants.BookingOverlap if the car's window was taken concurrently.
	CreateBooking(ctx context.Context, booking models.Booking) (models.Booking, error)

	// GetBookingByIdempotencyKey returns the existing booking for a key, or
	// constants.RecordDoesntExists when the key has not been used.
	GetBookingByIdempotencyKey(ctx context.Context, key string) (models.Booking, error)

	// ListUserBookings returns a filtered, paginated page of a user's bookings
	// and the total count matching the filter (for has_next calculation).
	ListUserBookings(ctx context.Context, query models.BookingQuery) (bookings []models.Booking, total int64, err error)
}
