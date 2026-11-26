package models

import "time"

// User is the actor that requests bookings.
type User struct {
	ID        int64 // Postgres bigserial, assigned by the DB
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
