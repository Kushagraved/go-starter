package models

import "time"

// CarCategory is the closed set of supported car types. Validate incoming values
// against IsValid before trusting them.
type CarCategory string

const (
	CategorySedan     CarCategory = "SEDAN"
	CategorySUV       CarCategory = "SUV"
	CategoryHatchback CarCategory = "HATCHBACK"
)

// IsValid reports whether c is one of the supported categories.
func (c CarCategory) IsValid() bool {
	switch c {
	case CategorySedan, CategorySUV, CategoryHatchback:
		return true
	default:
		return false
	}
}

func (c CarCategory) String() string { return string(c) }

// Car belongs to exactly one branch and has exactly one category.
type Car struct {
	ID        int64 // Postgres bigserial, assigned by the DB
	BranchID  int64
	Category  CarCategory
	CreatedAt time.Time
	UpdatedAt time.Time
}
