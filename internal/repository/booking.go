package repository

import (
	"context"
	"time"

	"go-starter/internal/constants"
	"go-starter/internal/models"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// skipLockedCars builds `FOR UPDATE OF c SKIP LOCKED`. The `OF c` restricts the
// row lock to the cars table (alias c) so the joined price rows are not locked,
// and SKIP LOCKED makes a concurrent transaction step over cars this one already
// holds instead of blocking on them.
func skipLockedCars() clause.Locking {
	return clause.Locking{
		Strength: "UPDATE",
		Table:    clause.Table{Name: "c"},
		Options:  "SKIP LOCKED",
	}
}

// candidateRow is the projection returned by the availability query. It is local
// to this method and never leaves the package.
type candidateRow struct {
	CarID    int64  `gorm:"column:car_id"`
	BranchID int64  `gorm:"column:branch_id"`
	Category string `gorm:"column:category"`
	Amount   int64  `gorm:"column:amount"`
	Unit     string `gorm:"column:unit"`
}

/*
FindAvailableCars returns every car of the requested category that has no
CONFIRMED booking overlapping [startTime, endTime), joined with its branch's
current price and ordered cheapest-first.

Concurrency: this MUST run inside Atomic. The query takes FOR UPDATE SKIP LOCKED
on the matched car rows, so if two transactions look for the same category at the
same time, each locks a different set of car rows — the second transaction skips
any car the first has already locked. Combined with the caller booking the first
(cheapest) row it gets, two users are never handed the same car.

Overlap rule (half-open intervals): an existing booking [s, e) conflicts with the
request [startTime, endTime) iff  s < endTime AND startTime < e.
*/
func (c client) FindAvailableCars(ctx context.Context, category models.CarCategory, startTime, endTime time.Time) ([]models.CandidateCar, error) {
	var rows []candidateRow

	err := c.db.WithContext(ctx).
		Table(Cars.ToString()+" AS c").
		Select("c.id AS car_id, c.branch_id AS branch_id, c.category AS category, p.amount AS amount, p.unit AS unit").
		Joins("JOIN "+PriceConfigurations.ToString()+" AS p ON p.branch_id = c.branch_id AND p.category = c.category").
		Where("c.category = ?", category.String()).
		Where(`NOT EXISTS (
			SELECT 1 FROM `+Bookings.ToString()+` AS b
			WHERE b.car_id = c.id
			  AND b.status = ?
			  AND b.start_time < ?
			  AND ? < b.end_time
		)`, string(models.BookingConfirmed), endTime, startTime).
		Order("p.amount ASC, c.id ASC").
		Clauses(skipLockedCars()).
		Find(&rows).Error
	if err != nil {
		return nil, errors.Wrap(err, "find available cars")
	}

	candidates := make([]models.CandidateCar, 0, len(rows))
	for _, r := range rows {
		candidates = append(candidates, models.CandidateCar{
			CarID:    r.CarID,
			BranchID: r.BranchID,
			Category: models.CarCategory(r.Category),
			Amount:   r.Amount,
			Unit:     models.PriceUnit(r.Unit),
		})
	}
	return candidates, nil
}

func (c client) CreateBooking(ctx context.Context, booking models.Booking) (models.Booking, error) {
	row := fromBooking(booking)
	if err := c.db.WithContext(ctx).Table(Bookings.ToString()).Create(&row).Error; err != nil {
		return models.Booking{}, translateBookingWriteErr(err)
	}
	return row.ToEntity(), nil
}

func (c client) GetBookingByIdempotencyKey(ctx context.Context, key string) (models.Booking, error) {
	var row bookingRow
	err := c.db.WithContext(ctx).Table(Bookings.ToString()).
		First(&row, "idempotency_key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Booking{}, constants.RecordDoesntExists
	}
	if err != nil {
		return models.Booking{}, errors.Wrap(err, "get booking by idempotency key")
	}
	return row.ToEntity(), nil
}

func (c client) ListUserBookings(ctx context.Context, query models.BookingQuery) ([]models.Booking, int64, error) {
	base := c.db.WithContext(ctx).Table(Bookings.ToString()).Where("user_id = ?", query.UserID)
	if query.Status != "" {
		base = base.Where("status = ?", string(query.Status))
	}
	if query.StartTime != nil {
		base = base.Where("start_time >= ?", *query.StartTime)
	}
	if query.EndTime != nil {
		base = base.Where("end_time <= ?", *query.EndTime)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, errors.Wrap(err, "count user bookings")
	}

	order := "created_at ASC"
	if query.SortDesc {
		order = "created_at DESC"
	}

	q := base.Order(order)
	if query.Limit > 0 {
		q = q.Limit(query.Limit)
	}
	if query.Offset > 0 {
		q = q.Offset(query.Offset)
	}

	var rows []bookingRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, 0, errors.Wrap(err, "list user bookings")
	}

	bookings := make([]models.Booking, 0, len(rows))
	for _, r := range rows {
		bookings = append(bookings, r.ToEntity())
	}
	return bookings, total, nil
}
