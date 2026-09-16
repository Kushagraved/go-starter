package middleware

import (
	"context"

	"go-starter/internal/constants"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

const HeaderRequestID = "X-Request-ID"

// RequestID injects a request ID and a trace ID into the request context and
// echoes the request ID back on the response header. Putting them on
// *http.Request.Context means every downstream log line picks them up
// automatically via the logger.
func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := ctx.GetHeader(HeaderRequestID)
		if requestID == "" {
			id, err := uuid.NewV4()
			if err == nil {
				requestID = id.String()
			}
		}

		traceID := requestID
		if traceID == "" {
			if id, err := uuid.NewV4(); err == nil {
				traceID = id.String()
			}
		}

		ctx.Set(HeaderRequestID, requestID)
		ctx.Header(HeaderRequestID, requestID)

		reqCtx := context.WithValue(ctx.Request.Context(), constants.RequestID, requestID)
		reqCtx = context.WithValue(reqCtx, constants.TraceID, traceID)
		ctx.Request = ctx.Request.WithContext(reqCtx)

		ctx.Next()
	}
}
