// This file holds the private database row structs. They carry the GORM column
// tags and never leave the repository package: every exported method converts to
// and from the models.* domain entities via ToEntity / from* helpers.
package repository

import (
	"encoding/json"
	"time"

	"go-starter/internal/models"
)

type branchRow struct {
	ID        int64     `gorm:"column:id"` // bigserial; DB assigns on insert
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (r branchRow) ToEntity() models.Branch {
	return models.Branch{
		ID:        r.ID,
		Name:      r.Name,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// fromBranch omits ID so Postgres assigns the bigserial value.
func fromBranch(b models.Branch) branchRow {
	return branchRow{Name: b.Name}
}

type carRow struct {
	ID        int64     `gorm:"column:id"` // bigserial; DB assigns on insert
	BranchID  int64     `gorm:"column:branch_id"`
	Category  string    `gorm:"column:category"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (r carRow) ToEntity() models.Car {
	return models.Car{
		ID:        r.ID,
		BranchID:  r.BranchID,
		Category:  models.CarCategory(r.Category),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// fromCar omits ID so Postgres assigns the bigserial value.
func fromCar(c models.Car) carRow {
	return carRow{
		BranchID: c.BranchID,
		Category: c.Category.String(),
	}
}

type priceConfigurationRow struct {
	ID       int64  `gorm:"column:id"` // bigserial; DB assigns on insert
	BranchID int64  `gorm:"column:branch_id"`
	Category string `gorm:"column:category"`
	Amount   int64  `gorm:"column:amount"`
	Unit     string `gorm:"column:unit"`
	// AuditTrail is the []models.PriceChange history, stored as JSON text. It is
	// declared as a string with an explicit jsonb column type so the driver binds
	// it as JSON rather than as bytea (which Postgres will not implicitly cast).
	AuditTrail string    `gorm:"column:audit_trail;type:jsonb"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

// ToEntity unmarshals the JSONB audit trail; a malformed trail degrades to an
// empty slice rather than failing the read.
func (r priceConfigurationRow) ToEntity() models.PriceConfiguration {
	var trail []models.PriceChange
	if r.AuditTrail != "" {
		_ = json.Unmarshal([]byte(r.AuditTrail), &trail)
	}
	return models.PriceConfiguration{
		ID:         r.ID,
		BranchID:   r.BranchID,
		Category:   models.CarCategory(r.Category),
		Amount:     r.Amount,
		Unit:       models.PriceUnit(r.Unit),
		AuditTrail: trail,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

type userRow struct {
	ID        int64     `gorm:"column:id"` // bigserial; DB assigns on insert
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (r userRow) ToEntity() models.User {
	return models.User{
		ID:        r.ID,
		Name:      r.Name,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// fromUser omits ID so Postgres assigns the bigserial value.
func fromUser(u models.User) userRow {
	return userRow{Name: u.Name}
}

type bookingRow struct {
	ID             int64     `gorm:"column:id"` // bigserial; DB assigns on insert
	UserID         int64     `gorm:"column:user_id"`
	CarID          int64     `gorm:"column:car_id"`
	StartTime      time.Time `gorm:"column:start_time"`
	EndTime        time.Time `gorm:"column:end_time"`
	Amount         int64     `gorm:"column:amount"`
	Unit           string    `gorm:"column:unit"`
	Status         string    `gorm:"column:status"`
	IdempotencyKey string    `gorm:"column:idempotency_key"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (r bookingRow) ToEntity() models.Booking {
	return models.Booking{
		ID:             r.ID,
		UserID:         r.UserID,
		CarID:          r.CarID,
		StartTime:      r.StartTime,
		EndTime:        r.EndTime,
		Amount:         r.Amount,
		Unit:           models.PriceUnit(r.Unit),
		Status:         models.BookingStatus(r.Status),
		IdempotencyKey: r.IdempotencyKey,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

// fromBooking omits ID so Postgres assigns the bigserial value.
func fromBooking(b models.Booking) bookingRow {
	return bookingRow{
		UserID:         b.UserID,
		CarID:          b.CarID,
		StartTime:      b.StartTime,
		EndTime:        b.EndTime,
		Amount:         b.Amount,
		Unit:           string(b.Unit),
		Status:         string(b.Status),
		IdempotencyKey: b.IdempotencyKey,
	}
}
