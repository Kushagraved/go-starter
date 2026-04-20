package controller

import (
	"go-starter/internal/core/task"
	"go-starter/internal/logger"
)

// Services groups all domain service instances injected into the controller layer.
type Services struct {
	Task task.Service
}

// Server is the concrete HTTP handler struct.
type Server struct {
	services Services
	logger   logger.Logger
}

// New constructs the server (mirrors the original pattern).
func New(services Services, log logger.Logger) *Server {
	return &Server{services: services, logger: log}
}

// throwError writes a standardised error JSON response.
func throwError(ctx interface {
	JSON(int, interface{})
}, code int, errCode, msg string) {
	ctx.JSON(code, map[string]interface{}{
		"error":   errCode,
		"message": msg,
	})
}
