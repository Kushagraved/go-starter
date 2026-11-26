package controller

import (
	"net/http"
	"strconv"
	"time"

	apibooking "go-starter/api/booking"
	"go-starter/internal/constants"
	"go-starter/internal/core/booking"
	"go-starter/internal/dto"
	"go-starter/internal/models"

	"github.com/gin-gonic/gin"
)

const idempotencyHeader = "Idempotency-Key"

func (s *Server) CreateBooking(ctx *gin.Context) {
	idempotencyKey := ctx.GetHeader(idempotencyHeader)
	if idempotencyKey == "" {
		ThrowError(ctx, http.StatusBadRequest, constants.ErrBadRequest, "Idempotency-Key header is required")
		return
	}

	var req apibooking.CreateBookingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ThrowError(ctx, http.StatusBadRequest, constants.ErrBadRequest, err.Error())
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		ThrowError(ctx, http.StatusBadRequest, constants.ErrBadRequest, "start_time must be RFC3339")
		return
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		ThrowError(ctx, http.StatusBadRequest, constants.ErrBadRequest, "end_time must be RFC3339")
		return
	}

	result, err := s.services.Booking.Create(ctx.Request.Context(), booking.CreateRequest{
		UserID:         req.UserID,
		Category:       models.CarCategory(req.Type),
		StartTime:      startTime,
		EndTime:        endTime,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		s.RespondError(ctx, "CreateBooking", err)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"success": true, "data": dto.ToBookingResponse(result)})
}

func (s *Server) ListUserBookings(ctx *gin.Context) {
	userID, err := parseInt64Param(ctx, "user_id")
	if err != nil {
		ThrowError(ctx, http.StatusBadRequest, constants.ErrBadRequest, "invalid user_id")
		return
	}

	query := models.BookingQuery{
		UserID:   userID,
		Limit:    parseIntQuery(ctx, "limit", 20),
		Offset:   parseIntQuery(ctx, "offset", 0),
		SortDesc: ctx.DefaultQuery("sort", "desc") != "asc",
	}
	if status := ctx.Query("status"); status != "" {
		query.Status = models.BookingStatus(status)
	}
	if v := ctx.Query("start_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			query.StartTime = &t
		}
	}
	if v := ctx.Query("end_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			query.EndTime = &t
		}
	}

	results, total, err := s.services.Booking.ListByUser(ctx.Request.Context(), query)
	if err != nil {
		s.RespondError(ctx, "ListUserBookings", err)
		return
	}

	// has_next: there are more rows beyond the current page window.
	hasNext := int64(query.Offset+len(results)) < total

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": apibooking.ListBookingsResponse{
			UserID:   userID,
			Bookings: dto.ToBookingResponses(results),
			Meta: apibooking.ListMeta{
				Limit:   query.Limit,
				Offset:  query.Offset,
				HasNext: hasNext,
			},
		},
	})
}

// parseIntQuery reads an integer query parameter, falling back to def when absent
// or unparseable.
func parseIntQuery(ctx *gin.Context, name string, def int) int {
	v := ctx.Query(name)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
