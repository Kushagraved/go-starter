package router

import (
	"go-starter/internal/controller"
	"go-starter/internal/middleware"
	"go-starter/internal/logger"

	"github.com/gin-gonic/gin"
)

func Init(srv *controller.Server, log logger.Logger) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestLogger(log))

	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.Group("/v1")
	registerTaskRoutes(v1, srv)

	return r
}
