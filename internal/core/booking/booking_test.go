package booking_test

import (
	"context"
	"time"

	"go-starter/internal"
	"go-starter/internal/constants"
	"go-starter/internal/core/booking"
	"go-starter/internal/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Booking Service Create", func() {
	var (
		ctrl *gomock.Controller
		repo *internal.MockRepository
		svc  booking.Service

		ctx   context.Context
		start time.Time
		end   time.Time
		req   booking.CreateRequest
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = internal.NewMockRepository(ctrl)
		svc = booking.NewService(repo, nil)

		ctx = context.Background()
		start = time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC)
		end = start.Add(2 * time.Hour)
		req = booking.CreateRequest{
			UserID:         1,
			Category:       models.CategorySedan,
			StartTime:      start,
			EndTime:        end,
			IdempotencyKey: "key-1",
		}
	})

	AfterEach(func() { ctrl.Finish() })

	// runAtomic makes the mocked Atomic execute the closure with the same mock as
	// the transaction-scoped DAO, so calls inside the closure hit these expectations.
	runAtomic := func() {
		repo.EXPECT().
			Atomic(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, fn func(internal.DAO) error) error {
				return fn(repo)
			})
	}

	Describe("happy path", func() {
		It("books the cheapest available car and confirms it", func() {
			repo.EXPECT().GetBookingByIdempotencyKey(gomock.Any(), "key-1").
				Return(models.Booking{}, constants.RecordDoesntExists)
			repo.EXPECT().GetUserByID(gomock.Any(), int64(1)).
				Return(models.User{ID: 1, Name: "Alice"}, nil)
			runAtomic()
			// inside the transaction: re-check key, then find cars.
			repo.EXPECT().GetBookingByIdempotencyKey(gomock.Any(), "key-1").
				Return(models.Booking{}, constants.RecordDoesntExists)
			repo.EXPECT().FindAvailableCars(gomock.Any(), models.CategorySedan, start, end).
				Return([]models.CandidateCar{
					// Repository returns these already ordered cheapest-first.
					{CarID: 7, BranchID: 3, Category: models.CategorySedan, Amount: 500, Unit: models.UnitPerHour},
					{CarID: 9, BranchID: 4, Category: models.CategorySedan, Amount: 800, Unit: models.UnitPerHour},
				}, nil)
			repo.EXPECT().CreateBooking(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, b models.Booking) (models.Booking, error) {
					// The service must pick the cheapest (car 7 @ 500).
					Expect(b.CarID).To(Equal(int64(7)))
					Expect(b.Amount).To(Equal(int64(500)))
					Expect(b.Status).To(Equal(models.BookingConfirmed))
					Expect(b.IdempotencyKey).To(Equal("key-1"))
					b.ID = 100
					return b, nil
				})

			result, err := svc.Create(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Booking.ID).To(Equal(int64(100)))
			Expect(result.Booking.CarID).To(Equal(int64(7)))
			Expect(result.Booking.Amount).To(Equal(int64(500)))
			Expect(result.CarBranchID).To(Equal(int64(3)))
			Expect(result.CarCategory).To(Equal(models.CategorySedan))
		})
	})

	Describe("idempotency", func() {
		It("returns the existing booking without allocating again (fast path)", func() {
			existing := models.Booking{
				ID: 42, UserID: 1, CarID: 7, Amount: 500,
				Unit: models.UnitPerHour, Status: models.BookingConfirmed, IdempotencyKey: "key-1",
			}
			repo.EXPECT().GetBookingByIdempotencyKey(gomock.Any(), "key-1").
				Return(existing, nil)
			// enrich fetches the car for the response.
			repo.EXPECT().GetCarByID(gomock.Any(), int64(7)).
				Return(models.Car{ID: 7, BranchID: 3, Category: models.CategorySedan}, nil)
			// No Atomic, no FindAvailableCars, no CreateBooking expected.

			result, err := svc.Create(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Booking.ID).To(Equal(int64(42)))
			Expect(result.CarBranchID).To(Equal(int64(3)))
		})

		It("returns the winner when a concurrent request took the key mid-transaction", func() {
			winner := models.Booking{ID: 55, UserID: 1, CarID: 7, IdempotencyKey: "key-1", Status: models.BookingConfirmed}

			repo.EXPECT().GetBookingByIdempotencyKey(gomock.Any(), "key-1").
				Return(models.Booking{}, constants.RecordDoesntExists)
			repo.EXPECT().GetUserByID(gomock.Any(), int64(1)).
				Return(models.User{ID: 1}, nil)
			runAtomic()
			// Inside the transaction the key now exists: return it, skip allocation.
			repo.EXPECT().GetBookingByIdempotencyKey(gomock.Any(), "key-1").
				Return(winner, nil)
			repo.EXPECT().GetCarByID(gomock.Any(), int64(7)).
				Return(models.Car{ID: 7, BranchID: 3, Category: models.CategorySedan}, nil)

			result, err := svc.Create(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Booking.ID).To(Equal(int64(55)))
		})

		It("resolves a duplicate-key insert race to the winning booking", func() {
			winner := models.Booking{ID: 77, UserID: 1, CarID: 7, IdempotencyKey: "key-1", Status: models.BookingConfirmed}

			repo.EXPECT().GetBookingByIdempotencyKey(gomock.Any(), "key-1").
				Return(models.Booking{}, constants.RecordDoesntExists)
			repo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(models.User{ID: 1}, nil)
			runAtomic()
			repo.EXPECT().GetBookingByIdempotencyKey(gomock.Any(), "key-1").
				Return(models.Booking{}, constants.RecordDoesntExists)
			repo.EXPECT().FindAvailableCars(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]models.CandidateCar{{CarID: 7, BranchID: 3, Category: models.CategorySedan, Amount: 500, Unit: models.UnitPerHour}}, nil)
			// The insert loses the race on the unique idempotency key.
			repo.EXPECT().CreateBooking(gomock.Any(), gomock.Any()).
				Return(models.Booking{}, constants.DuplicateIdempotencyKey)
			// Service recovers by returning the committed winner.
			repo.EXPECT().GetBookingByIdempotencyKey(gomock.Any(), "key-1").Return(winner, nil)
			repo.EXPECT().GetCarByID(gomock.Any(), int64(7)).
				Return(models.Car{ID: 7, BranchID: 3, Category: models.CategorySedan}, nil)

			result, err := svc.Create(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Booking.ID).To(Equal(int64(77)))
		})
	})

	Describe("concurrency / overlap", func() {
		It("propagates BookingOverlap when the exclusion constraint rejects the insert", func() {
			repo.EXPECT().GetBookingByIdempotencyKey(gomock.Any(), "key-1").
				Return(models.Booking{}, constants.RecordDoesntExists)
			repo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(models.User{ID: 1}, nil)
			runAtomic()
			repo.EXPECT().GetBookingByIdempotencyKey(gomock.Any(), "key-1").
				Return(models.Booking{}, constants.RecordDoesntExists)
			repo.EXPECT().FindAvailableCars(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]models.CandidateCar{{CarID: 7, BranchID: 3, Category: models.CategorySedan, Amount: 500, Unit: models.UnitPerHour}}, nil)
			repo.EXPECT().CreateBooking(gomock.Any(), gomock.Any()).
				Return(models.Booking{}, constants.BookingOverlap)

			_, err := svc.Create(ctx, req)
			Expect(err).To(MatchError(constants.BookingOverlap))
		})
	})

	Describe("no availability", func() {
		It("returns NoCarAvailable when no car matches the window", func() {
			repo.EXPECT().GetBookingByIdempotencyKey(gomock.Any(), "key-1").
				Return(models.Booking{}, constants.RecordDoesntExists)
			repo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(models.User{ID: 1}, nil)
			runAtomic()
			repo.EXPECT().GetBookingByIdempotencyKey(gomock.Any(), "key-1").
				Return(models.Booking{}, constants.RecordDoesntExists)
			repo.EXPECT().FindAvailableCars(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]models.CandidateCar{}, nil)

			_, err := svc.Create(ctx, req)
			Expect(err).To(MatchError(constants.NoCarAvailable))
		})
	})

	Describe("unknown user", func() {
		It("propagates RecordDoesntExists and never opens a transaction", func() {
			repo.EXPECT().GetBookingByIdempotencyKey(gomock.Any(), "key-1").
				Return(models.Booking{}, constants.RecordDoesntExists)
			repo.EXPECT().GetUserByID(gomock.Any(), int64(1)).
				Return(models.User{}, constants.RecordDoesntExists)

			_, err := svc.Create(ctx, req)
			Expect(err).To(MatchError(constants.RecordDoesntExists))
		})
	})

	Describe("validation", func() {
		DescribeTable("rejects bad input with a validation error before touching the repo",
			func(mutate func(*booking.CreateRequest)) {
				bad := req
				mutate(&bad)
				_, err := svc.Create(ctx, bad)
				Expect(err).To(MatchError(constants.ErrValidation))
			},
			Entry("missing user", func(r *booking.CreateRequest) { r.UserID = 0 }),
			Entry("invalid category", func(r *booking.CreateRequest) { r.Category = "TRUCK" }),
			Entry("missing idempotency key", func(r *booking.CreateRequest) { r.IdempotencyKey = "" }),
			Entry("end before start", func(r *booking.CreateRequest) { r.EndTime = r.StartTime.Add(-time.Hour) }),
			Entry("end equals start", func(r *booking.CreateRequest) { r.EndTime = r.StartTime }),
			Entry("zero start", func(r *booking.CreateRequest) { r.StartTime = time.Time{} }),
		)
	})
})

var _ = Describe("Booking Service ListByUser", func() {
	var (
		ctrl *gomock.Controller
		repo *internal.MockRepository
		svc  booking.Service
		ctx  context.Context
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = internal.NewMockRepository(ctrl)
		svc = booking.NewService(repo, nil)
		ctx = context.Background()
	})

	AfterEach(func() { ctrl.Finish() })

	It("returns enriched bookings and the total count", func() {
		repo.EXPECT().ListUserBookings(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, q models.BookingQuery) ([]models.Booking, int64, error) {
				// The service applies a default limit when none is given.
				Expect(q.Limit).To(Equal(20))
				return []models.Booking{
					{ID: 1, UserID: 5, CarID: 7, Status: models.BookingConfirmed},
				}, int64(3), nil
			})
		repo.EXPECT().GetCarByID(gomock.Any(), int64(7)).
			Return(models.Car{ID: 7, BranchID: 3, Category: models.CategorySUV}, nil)

		results, total, err := svc.ListByUser(ctx, models.BookingQuery{UserID: 5})
		Expect(err).ToNot(HaveOccurred())
		Expect(total).To(Equal(int64(3)))
		Expect(results).To(HaveLen(1))
		Expect(results[0].CarBranchID).To(Equal(int64(3)))
		Expect(results[0].CarCategory).To(Equal(models.CategorySUV))
	})

	It("caps the limit at 100", func() {
		repo.EXPECT().ListUserBookings(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, q models.BookingQuery) ([]models.Booking, int64, error) {
				Expect(q.Limit).To(Equal(100))
				return nil, 0, nil
			})

		_, _, err := svc.ListByUser(ctx, models.BookingQuery{UserID: 5, Limit: 5000})
		Expect(err).ToNot(HaveOccurred())
	})

	It("rejects a missing user id", func() {
		_, _, err := svc.ListByUser(ctx, models.BookingQuery{UserID: 0})
		Expect(err).To(MatchError(constants.ErrValidation))
	})
})
