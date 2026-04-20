package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

const HeaderRequestID = "X-Request-ID"

// RequestID injects a unique request ID into every request context and response header.
func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		rid := ctx.GetHeader(HeaderRequestID)
		if rid == "" {
			id, _ := uuid.NewV4()
			rid = id.String()
		}
		ctx.Set(HeaderRequestID, rid)
		ctx.Header(HeaderRequestID, rid)
		ctx.Next()
	}
}
