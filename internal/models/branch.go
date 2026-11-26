package models

import "time"

// Branch is a physical location in the city that owns a fleet of cars and its own
// per-category pricing.
type Branch struct {
	ID        int64 // Postgres bigserial, assigned by the DB
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
