// Package router maps URLs to controller methods. Give each domain its own
// <domain>_routes.go file and register it from Init.
package router

import (
	"go-starter/internal/controller"
	"go-starter/internal/logger"
	"go-starter/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Init builds the HTTP handler.
//
// NOTE: no authentication is applied here. Add an auth middleware to the v1 group
// before exposing this service to untrusted traffic.
func Init(srv *controller.Server, loggerObj logger.Logger) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.RequestID())
	engine.Use(middleware.RequestLogger(loggerObj))

	engine.GET("/health", srv.Health)

	v1 := engine.Group("/v1")
	registerOnboardingRoutes(v1, srv)
	registerBookingRoutes(v1, srv)

	return engine
}
