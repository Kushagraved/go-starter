package models

import "time"

// BookingStatus is the lifecycle state of a booking. Only CONFIRMED exists today;
// the type leaves room for CANCELLED/COMPLETED without a schema change.
type BookingStatus string

const (
	BookingConfirmed BookingStatus = "CONFIRMED"
)

// Booking blocks one car for a half-open time window [StartTime, EndTime). Two
// bookings for the same car may not overlap. Amount/Unit snapshot the price the
// booking was made at, independent of later pricing changes.
type Booking struct {
	ID             int64 // Postgres bigserial, assigned by the DB
	UserID         int64
	CarID          int64
	StartTime      time.Time
	EndTime        time.Time
	Amount         int64
	Unit           PriceUnit
	Status         BookingStatus
	IdempotencyKey string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CandidateCar is a car eligible for a booking request together with the price it
// would be booked at. The repository returns these ordered by Amount ascending so
// the core layer can pick the cheapest.
type CandidateCar struct {
	CarID    int64
	BranchID int64
	Category CarCategory
	Amount   int64
	Unit     PriceUnit
}

// BookingQuery is the filter and pagination for listing a user's bookings.
// Zero-value fields mean "unfiltered" for that dimension.
type BookingQuery struct {
	UserID    int64
	Status    BookingStatus // empty means any status
	StartTime *time.Time    // inclusive lower bound on booking start, optional
	EndTime   *time.Time    // exclusive upper bound on booking end, optional
	Limit     int
	Offset    int
	SortDesc  bool // sort by created_at; true = newest first
}
