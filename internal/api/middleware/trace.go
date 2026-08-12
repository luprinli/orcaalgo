package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TraceMiddleware extracts or generates a request ID for distributed tracing.
// It reads the X-Request-ID header (falling back to X-Correlation-ID), generates
// a new UUID if neither is present, then stores the ID in the gin context and
// sets it on the response header.
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = c.GetHeader("X-Correlation-ID")
		}
		if id == "" {
			id = uuid.New().String()
		}

		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), "request_id", id))

		c.Next()
	}
}

// GetRequestID retrieves the request ID previously stored by TraceMiddleware.
// Returns an empty string if no request ID was set.
func GetRequestID(c *gin.Context) string {
	if id, ok := c.Get("request_id"); ok {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}
