package middleware

import (
	"time"

	"go-starter/internal/logger"

	"github.com/gin-gonic/gin"
)

// RequestLogger logs method, path, status and latency for every request.
func RequestLogger(log logger.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()

		log.Info(ctx.Request.Context(), "http", "RequestLogger", "request completed",
			logger.NewString("method", ctx.Request.Method),
			logger.NewString("path", ctx.Request.URL.Path),
			logger.NewInt("status", ctx.Writer.Status()),
			logger.NewInt64("latency_ms", time.Since(start).Milliseconds()),
		)
	}
}
