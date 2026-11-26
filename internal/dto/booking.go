package dto

import (
	"time"

	apibooking "go-starter/api/booking"
	"go-starter/internal/core/booking"
)

// ToBookingResponse maps a booking.Result (booking + its car's branch/category)
// to the API shape.
func ToBookingResponse(r booking.Result) apibooking.BookingResponse {
	b := r.Booking
	return apibooking.BookingResponse{
		BookingID: b.ID,
		UserID:    b.UserID,
		StartTime: b.StartTime.Format(time.RFC3339),
		EndTime:   b.EndTime.Format(time.RFC3339),
		Amount:    b.Amount,
		Unit:      string(b.Unit),
		Status:    string(b.Status),
		Car: apibooking.CarInfo{
			ID:       b.CarID,
			BranchID: r.CarBranchID,
			Type:     r.CarCategory.String(),
		},
	}
}

// ToBookingResponses maps a page of results.
func ToBookingResponses(results []booking.Result) []apibooking.BookingResponse {
	out := make([]apibooking.BookingResponse, 0, len(results))
	for _, r := range results {
		out = append(out, ToBookingResponse(r))
	}
	return out
}
