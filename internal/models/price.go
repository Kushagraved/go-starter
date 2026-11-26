package models

import "time"

// PriceUnit is the billing unit for a price. Pricing is defined per hour today,
// but the unit is stored explicitly so a booking records the unit it was priced
// under even if the configuration changes later.
type PriceUnit string

const (
	// UnitPerHour is the only supported unit for now: amount is per hour.
	UnitPerHour PriceUnit = "PER_HOUR"
)

// PriceChange is one entry in a PriceConfiguration audit trail: it records a
// single amount transition and when it happened.
type PriceChange struct {
	UpdatedAt  time.Time `json:"updated_at"`
	FromAmount int64     `json:"from_amount"`
	ToAmount   int64     `json:"to_amount"`
}

// PriceConfiguration is the price for one (branch, category) pair. All cars of the
// same category within a branch share this price. Amount is stored in the
// smallest currency unit (e.g. paise/cents) to avoid floating point.
type PriceConfiguration struct {
	ID         int64 // Postgres bigserial, assigned by the DB
	BranchID   int64
	Category   CarCategory
	Amount     int64
	Unit       PriceUnit
	AuditTrail []PriceChange
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
