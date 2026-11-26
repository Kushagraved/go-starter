// Package onboarding holds the JSON request/response shapes for the onboarding
// endpoints. Tags only, no logic.
package onboarding

// CreateBranchRequest is the body of POST /v1/branches.
type CreateBranchRequest struct {
	Name string `json:"name" binding:"required"`
}

// BranchResponse is a branch in an API response.
type BranchResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// CreateCarRequest is the body of POST /v1/branches/:branch_id/cars.
type CreateCarRequest struct {
	Category string `json:"category" binding:"required"`
}

// CarResponse is a car in an API response.
type CarResponse struct {
	ID       int64  `json:"id"`
	BranchID int64  `json:"branch_id"`
	Category string `json:"category"`
}

// SetPriceRequest is the body of PUT /v1/branches/:branch_id/prices.
type SetPriceRequest struct {
	Category string `json:"category" binding:"required"`
	Amount   int64  `json:"amount" binding:"required"`
}

// PriceChangeResponse is one audit-trail entry.
type PriceChangeResponse struct {
	UpdatedAt  string `json:"updated_at"`
	FromAmount int64  `json:"from_amount"`
	ToAmount   int64  `json:"to_amount"`
}

// PriceConfigurationResponse is a price configuration in an API response.
type PriceConfigurationResponse struct {
	ID         int64                 `json:"id"`
	BranchID   int64                 `json:"branch_id"`
	Category   string                `json:"category"`
	Amount     int64                 `json:"amount"`
	Unit       string                `json:"unit"`
	AuditTrail []PriceChangeResponse `json:"audit_trail"`
}

// CreateUserRequest is the body of POST /v1/users.
type CreateUserRequest struct {
	Name string `json:"name" binding:"required"`
}

// UserResponse is a user in an API response.
type UserResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
