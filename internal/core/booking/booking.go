// Package booking holds the allocation logic: given a category and a time window,
// pick the cheapest available car and confirm a booking, safely under concurrency.
package booking

import (
	"context"
	"time"

	"go-starter/internal"
	"go-starter/internal/constants"
	"go-starter/internal/logger"
	"go-starter/internal/models"

	"github.com/pkg/errors"
)

const bucket = "BookingService"

// Service is the booking API used by controllers.
type Service interface {
	// Create allocates the cheapest available car of the requested category for
	// the window and confirms a booking. It is idempotent on IdempotencyKey.
	Create(ctx context.Context, req CreateRequest) (Result, error)
	// ListByUser returns a filtered, paginated page of a user's bookings plus the
	// total count matching the filter.
	ListByUser(ctx context.Context, query models.BookingQuery) ([]Result, int64, error)
}

// Result is a booking together with the branch and category of the car it holds,
// so the API can render the embedded car object without a second lookup.
type Result struct {
	Booking     models.Booking
	CarBranchID int64
	CarCategory models.CarCategory
}

// CreateRequest is a booking request. The user does not choose a branch or car —
// the service allocates one.
type CreateRequest struct {
	UserID         int64
	Category       models.CarCategory
	StartTime      time.Time
	EndTime        time.Time
	IdempotencyKey string
}

type service struct {
	repo   internal.Repository
	logger logger.Logger
}

// NewService builds the booking service.
func NewService(repo internal.Repository, loggerObj logger.Logger) Service {
	return &service{repo: repo, logger: loggerObj}
}

func (s *service) validate(req CreateRequest) error {
	if req.UserID <= 0 {
		return errors.Wrap(constants.ErrValidation, "user_id is required")
	}
	if !req.Category.IsValid() {
		return errors.Wrapf(constants.ErrValidation, "invalid car category %q", req.Category)
	}
	if req.IdempotencyKey == "" {
		return errors.Wrap(constants.ErrValidation, "idempotency key is required")
	}
	if req.StartTime.IsZero() || req.EndTime.IsZero() {
		return errors.Wrap(constants.ErrValidation, "start_time and end_time are required")
	}
	if !req.EndTime.After(req.StartTime) {
		return errors.Wrap(constants.ErrValidation, "end_time must be after start_time")
	}
	return nil
}

/*
Create confirms a booking for the cheapest available car.

Idempotency: if the client-supplied key was already used, the existing booking is
returned unchanged (a retried request is a no-op, not a second car).

Concurrency: the allocation runs in one transaction (Atomic). FindAvailableCars
takes FOR UPDATE SKIP LOCKED on candidate car rows, so two simultaneous requests
for the same category each lock a different car and never collide. The database's
exclusion constraint is the final backstop: if a race still produced an
overlapping booking, the insert fails with constants.BookingOverlap and the whole
transaction rolls back.
*/
func (s *service) Create(ctx context.Context, req CreateRequest) (Result, error) {
	if err := s.validate(req); err != nil {
		return Result{}, err
	}

	// Fast path: a prior request with this key already succeeded.
	if existing, err := s.repo.GetBookingByIdempotencyKey(ctx, req.IdempotencyKey); err == nil {
		return s.enrich(ctx, existing)
	} else if !errors.Is(err, constants.RecordDoesntExists) {
		return Result{}, err
	}

	// The user must exist.
	if _, err := s.repo.GetUserByID(ctx, req.UserID); err != nil {
		return Result{}, err
	}

	var result Result
	err := s.repo.Atomic(ctx, func(store internal.DAO) error {
		// Re-check inside the transaction: a concurrent request holding the same
		// key may have committed between the fast-path check and here.
		if existing, err := store.GetBookingByIdempotencyKey(ctx, req.IdempotencyKey); err == nil {
			enriched, encErr := s.enrichWith(ctx, store, existing)
			if encErr != nil {
				return encErr
			}
			result = enriched
			return nil
		} else if !errors.Is(err, constants.RecordDoesntExists) {
			return err
		}

		candidates, err := store.FindAvailableCars(ctx, req.Category, req.StartTime, req.EndTime)
		if err != nil {
			return err
		}
		if len(candidates) == 0 {
			return constants.NoCarAvailable
		}

		// Candidates are ordered cheapest-first and the rows are locked, so the
		// first one is both the best price and safe to claim.
		chosen := candidates[0]

		created, err := store.CreateBooking(ctx, models.Booking{
			UserID:         req.UserID,
			CarID:          chosen.CarID,
			StartTime:      req.StartTime,
			EndTime:        req.EndTime,
			Amount:         chosen.Amount,
			Unit:           chosen.Unit,
			Status:         models.BookingConfirmed,
			IdempotencyKey: req.IdempotencyKey,
		})
		if err != nil {
			return err
		}

		result = Result{
			Booking:     created,
			CarBranchID: chosen.BranchID,
			CarCategory: chosen.Category,
		}
		return nil
	})
	if err != nil {
		// A key that raced us to commit surfaces as a duplicate; return the winner.
		if errors.Is(err, constants.DuplicateIdempotencyKey) {
			existing, getErr := s.repo.GetBookingByIdempotencyKey(ctx, req.IdempotencyKey)
			if getErr != nil {
				return Result{}, getErr
			}
			return s.enrich(ctx, existing)
		}
		// NoCarAvailable and BookingOverlap are expected outcomes (they map to a
		// 409), not faults — don't log them at error level. Everything else is
		// genuinely unexpected and worth a log line.
		if s.logger != nil && !isExpectedBookingErr(err) {
			s.logger.Error(ctx, bucket, "Create", "booking failed",
				logger.NewInt64("user_id", req.UserID),
				logger.NewString("category", req.Category.String()),
				logger.NewError(err))
		}
		return Result{}, err
	}

	return result, nil
}

func (s *service) ListByUser(ctx context.Context, query models.BookingQuery) ([]Result, int64, error) {
	if query.UserID <= 0 {
		return nil, 0, errors.Wrap(constants.ErrValidation, "user_id is required")
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	bookings, total, err := s.repo.ListUserBookings(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	results := make([]Result, 0, len(bookings))
	for _, b := range bookings {
		enriched, err := s.enrich(ctx, b)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, enriched)
	}
	return results, total, nil
}

// isExpectedBookingErr reports whether err is a normal, caller-facing outcome
// rather than an internal fault.
func isExpectedBookingErr(err error) bool {
	return errors.Is(err, constants.NoCarAvailable) ||
		errors.Is(err, constants.BookingOverlap) ||
		errors.Is(err, constants.RecordDoesntExists) ||
		errors.Is(err, constants.ErrValidation)
}

// enrich fills in the car's branch and category for a booking using the repo.
func (s *service) enrich(ctx context.Context, b models.Booking) (Result, error) {
	car, err := s.repo.GetCarByID(ctx, b.CarID)
	if err != nil {
		return Result{}, err
	}
	return Result{Booking: b, CarBranchID: car.BranchID, CarCategory: car.Category}, nil
}

// enrichWith is enrich scoped to a transaction-bound DAO.
func (s *service) enrichWith(ctx context.Context, store internal.DAO, b models.Booking) (Result, error) {
	car, err := store.GetCarByID(ctx, b.CarID)
	if err != nil {
		return Result{}, err
	}
	return Result{Booking: b, CarBranchID: car.BranchID, CarCategory: car.Category}, nil
}
