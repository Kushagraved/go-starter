// Package booking holds the JSON request/response shapes for booking endpoints.
package booking

// CreateBookingRequest is the body of POST /v1/bookings. The idempotency key is
// sent as a header (Idempotency-Key), not in the body.
type CreateBookingRequest struct {
	UserID    int64  `json:"user_id" binding:"required"`
	Type      string `json:"type" binding:"required"`
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
}

// CarInfo is the allocated car embedded in a booking response.
type CarInfo struct {
	ID       int64  `json:"id"`
	BranchID int64  `json:"branch_id"`
	Type     string `json:"type"`
}

// BookingResponse is a booking in an API response.
type BookingResponse struct {
	BookingID int64   `json:"booking_id"`
	UserID    int64   `json:"user_id"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
	Amount    int64   `json:"amount"`
	Unit      string  `json:"unit"`
	Status    string  `json:"status"`
	Car       CarInfo `json:"car"`
}

// ListMeta is the pagination envelope for list responses.
type ListMeta struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasNext bool `json:"has_next"`
}

// ListBookingsResponse is the body of GET /v1/users/:user_id/bookings.
type ListBookingsResponse struct {
	UserID   int64             `json:"user_id"`
	Bookings []BookingResponse `json:"bookings"`
	Meta     ListMeta          `json:"meta"`
}
