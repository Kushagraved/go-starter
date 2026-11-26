// Package controller is the HTTP layer: bind the request, call a core service,
// shape the response. No business logic, no database access.
package controller

import (
	"net/http"

	"go-starter/internal/constants"
	"go-starter/internal/core/booking"
	"go-starter/internal/core/onboarding"
	"go-starter/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// Services groups the core services the handlers call. Add one field per domain;
// cmd/server/main.go is the only place that populates it.
//
//	type Services struct {
//	    User  user.Service
//	    Order order.Service
//	}
type Services struct {
	Onboarding onboarding.Service
	Booking    booking.Service
}

// Server holds the handlers. Give each domain its own file (user.go, order.go)
// with methods on *Server.
type Server struct {
	services Services
	logger   logger.Logger
}

// New constructs the server with its dependencies.
func New(services Services, loggerObj logger.Logger) *Server {
	return &Server{services: services, logger: loggerObj}
}

// Health is the liveness endpoint.
func (s *Server) Health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// RespondError maps a domain error onto an HTTP status: a missing record is a 404,
// lock contention is a 409, anything else is logged and returned as a 500.
// Call this instead of writing status codes by hand in every handler.
func (s *Server) RespondError(ctx *gin.Context, handler string, err error) {
	switch {
	case errors.Is(err, constants.ErrValidation):
		ThrowError(ctx, http.StatusBadRequest, constants.ErrBadRequest, err.Error())
	case errors.Is(err, constants.RecordDoesntExists):
		ThrowError(ctx, http.StatusNotFound, constants.ErrNotFound, "resource not found")
	case errors.Is(err, constants.PriceConfigNotFound):
		ThrowError(ctx, http.StatusNotFound, constants.ErrNotFound, "price configuration not found")
	case errors.Is(err, constants.NoCarAvailable):
		ThrowError(ctx, http.StatusConflict, constants.ErrConflict, "no car available for the requested window")
	case errors.Is(err, constants.BookingOverlap):
		ThrowError(ctx, http.StatusConflict, constants.ErrConflict, "car was just taken, please retry")
	case errors.Is(err, constants.LockAlreadyHeld):
		ThrowError(ctx, http.StatusConflict, constants.ErrConflict, "resource is busy, retry shortly")
	default:
		s.logger.Error(ctx.Request.Context(), "Controller", handler, "request failed", logger.NewError(err))
		ThrowError(ctx, http.StatusInternalServerError, constants.ErrInternalServer, "internal server error")
	}
}

// ThrowError writes the standard error envelope.
func ThrowError(ctx *gin.Context, statusCode int, errCode, message string) {
	ctx.JSON(statusCode, gin.H{
		"error":   errCode,
		"message": message,
	})
}
